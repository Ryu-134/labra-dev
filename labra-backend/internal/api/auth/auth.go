package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const principalContextKey contextKey = "principal"

var (
	ErrMissingAuthHeader = errors.New("missing Authorization header")
	ErrInvalidAuthHeader = errors.New("Authorization header must be Bearer <token>")
)

type Principal struct {
	UserID int64    `json:"user_id"`
	Sub    string   `json:"sub"`
	Email  string   `json:"email,omitempty"`
	Roles  []string `json:"roles,omitempty"`
}

type Validator interface {
	ValidateToken(ctx context.Context, rawToken string) (Principal, error)
}

type HMACValidator struct {
	Issuer   string
	Audience string
	Secret   []byte
}

func (v HMACValidator) ValidateToken(_ context.Context, rawToken string) (Principal, error) {
	if len(v.Secret) == 0 {
		return Principal{}, errors.New("jwt secret is not configured")
	}

	tok, err := jwt.Parse(rawToken, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %q", token.Method.Alg())
		}
		return v.Secret, nil
	}, jwt.WithIssuer(v.Issuer), jwt.WithAudience(v.Audience))
	if err != nil {
		return Principal{}, err
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, errors.New("invalid JWT claims")
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	uid, err := extractUserID(claims)
	if err != nil {
		return Principal{}, err
	}

	return Principal{
		UserID: uid,
		Sub:    strings.TrimSpace(sub),
		Email:  strings.TrimSpace(email),
		Roles:  extractRoles(claims),
	}, nil
}

func RequireAuth(v Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken, err := readBearerToken(r.Header.Get("Authorization"))
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

func withPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func readBearerToken(header string) (string, error) {
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
		return 0, errors.New("user_id claim is required")
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

func parsePositiveInt(raw string) (int64, error) {
	var id int64
	for _, ch := range strings.TrimSpace(raw) {
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
