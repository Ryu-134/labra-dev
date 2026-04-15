package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const principalContextKey contextKey = "principal"

var (
	ErrMissingAuthHeader = errors.New("missing Authorization header")
	ErrInvalidAuthHeader = errors.New("Authorization header must be Bearer <token>")
)

type Principal struct {
	UserID    int64    `json:"user_id"`
	Sub       string   `json:"sub"`
	Email     string   `json:"email,omitempty"`
	Roles     []string `json:"roles,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
	ExpiresAt int64    `json:"expires_at,omitempty"`
}

type Validator interface {
	ValidateToken(ctx context.Context, rawToken string) (Principal, error)
}

type HMACValidator struct {
	Issuer   string
	Audience string
	Secret   []byte
}

type TokenIssuer struct {
	Issuer   string
	Audience string
	Secret   []byte
	TTL      time.Duration
}

func (v HMACValidator) ValidateToken(_ context.Context, rawToken string) (Principal, error) {
	if len(v.Secret) == 0 {
		return Principal{}, errors.New("jwt secret is not configured")
	}

	parseOptions := []jwt.ParserOption{}
	if strings.TrimSpace(v.Issuer) != "" {
		parseOptions = append(parseOptions, jwt.WithIssuer(v.Issuer))
	}
	if strings.TrimSpace(v.Audience) != "" {
		parseOptions = append(parseOptions, jwt.WithAudience(v.Audience))
	}

	tok, err := jwt.Parse(rawToken, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %q", token.Method.Alg())
		}
		return v.Secret, nil
	}, parseOptions...)
	if err != nil {
		return Principal{}, err
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, errors.New("invalid JWT claims")
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	sessionID, _ := claims["session_id"].(string)
	expiresAt := extractExpiry(claims)

	uid, err := extractUserID(claims)
	if err != nil {
		return Principal{}, err
	}

	return Principal{
		UserID:    uid,
		Sub:       strings.TrimSpace(sub),
		Email:     strings.TrimSpace(email),
		Roles:     extractRoles(claims),
		SessionID: strings.TrimSpace(sessionID),
		ExpiresAt: expiresAt,
	}, nil
}

func (i TokenIssuer) MintSessionToken(userID int64, sub, email string, roles []string, sessionID string) (string, int64, error) {
	if len(i.Secret) == 0 {
		return "", 0, errors.New("token issuer secret is not configured")
	}
	if userID <= 0 {
		return "", 0, errors.New("user_id must be positive")
	}

	ttl := i.TTL
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	expiresAt := time.Now().Add(ttl).Unix()

	claims := jwt.MapClaims{
		"iss":        i.Issuer,
		"aud":        i.Audience,
		"sub":        strings.TrimSpace(sub),
		"user_id":    userID,
		"email":      strings.TrimSpace(email),
		"roles":      roles,
		"session_id": strings.TrimSpace(sessionID),
		"iat":        time.Now().Unix(),
		"exp":        expiresAt,
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := tok.SignedString(i.Secret)
	if err != nil {
		return "", 0, err
	}
	return raw, expiresAt, nil
}

func GenerateSessionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func RequireAuth(v Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken, err := ParseBearerToken(r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			principal, err := v.ValidateToken(r.Context(), rawToken)
			if err != nil {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
		})
	}
}

func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		r := strings.TrimSpace(strings.ToLower(role))
		if r == "" {
			continue
		}
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, "missing auth principal", http.StatusUnauthorized)
				return
			}

			for _, role := range principal.Roles {
				if _, exists := allowed[strings.ToLower(strings.TrimSpace(role))]; exists {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "insufficient role", http.StatusForbidden)
		})
	}
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return principal, ok
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return withPrincipal(ctx, principal)
}

func withPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func ParseBearerToken(header string) (string, error) {
	raw := strings.TrimSpace(header)
	if raw == "" {
		return "", ErrMissingAuthHeader
	}
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrInvalidAuthHeader
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", ErrInvalidAuthHeader
	}
	return token, nil
}

func extractUserID(claims jwt.MapClaims) (int64, error) {
	rawUserID, ok := claims["user_id"]
	if !ok {
		return 0, nil
	}

	switch v := rawUserID.(type) {
	case float64:
		if v <= 0 {
			return 0, errors.New("user_id claim must be positive")
		}
		return int64(v), nil
	case int64:
		if v <= 0 {
			return 0, errors.New("user_id claim must be positive")
		}
		return v, nil
	case string:
		id, convErr := parsePositiveInt(v)
		if convErr != nil {
			return 0, errors.New("user_id claim must be numeric")
		}
		return id, nil
	default:
		return 0, errors.New("user_id claim has unsupported type")
	}
}

func extractExpiry(claims jwt.MapClaims) int64 {
	raw, ok := claims["exp"]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case string:
		n, err := parsePositiveInt(v)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func parsePositiveInt(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, errors.New("invalid integer")
	}
	var id int64
	for _, ch := range trimmed {
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid integer")
		}
		id = id*10 + int64(ch-'0')
	}
	if id <= 0 {
		return 0, errors.New("must be positive")
	}
	return id, nil
}

func extractRoles(claims jwt.MapClaims) []string {
	roles := make([]string, 0)
	appendRole := func(v string) {
		role := strings.TrimSpace(v)
		if role == "" {
			return
		}
		roles = append(roles, role)
	}

	if raw, ok := claims["roles"]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					appendRole(s)
				}
			}
		case []string:
			for _, item := range v {
				appendRole(item)
			}
		case string:
			appendRole(v)
		}
	}

	if raw, ok := claims["cognito:groups"]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					appendRole(s)
				}
			}
		case []string:
			for _, item := range v {
				appendRole(item)
			}
		case string:
			appendRole(v)
		}
	}

	return roles
}
