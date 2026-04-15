package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"labra-backend/internal/api/auth"
	"labra-backend/internal/api/store"
)

var (
	authValidator auth.Validator
	tokenIssuer   auth.TokenIssuer
)

func InitAuthRuntime(validator auth.Validator, issuer auth.TokenIssuer) {
	authValidator = validator
	tokenIssuer = issuer
}

func PostAuthSessionHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}
	if authValidator == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "auth validator is not configured")
		return
	}

	rawToken, err := auth.ParseBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "missing Authorization bearer token")
		return
	}

	externalPrincipal, err := authValidator.ValidateToken(r.Context(), rawToken)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid auth token")
		return
	}
	if strings.TrimSpace(externalPrincipal.Sub) == "" {
		writeJSONError(w, http.StatusUnauthorized, "auth token missing subject")
		return
	}

	user, isNewUser, err := provisionPlatformUser(r, externalPrincipal)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to provision user identity")
		return
	}

	sessionID := auth.GenerateSessionID()
	signedToken, expiresAt, err := tokenIssuer.MintSessionToken(
		user.ID,
		externalPrincipal.Sub,
		externalPrincipal.Email,
		externalPrincipal.Roles,
		sessionID,
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create auth session")
		return
	}

	_, err = appStore.CreateAuthSession(r.Context(), store.CreateAuthSessionInput{
		SessionID: sessionID,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to persist auth session")
		return
	}

	_ = appStore.CreateAuditEvent(r.Context(), store.AuditEventInput{
		ActorUserID: user.ID,
		EventType:   "auth.session.create",
		TargetType:  "auth_session",
		TargetID:    sessionID,
		Status:      "success",
		Message:     "session issued",
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"session": map[string]any{
			"token":      signedToken,
			"session_id": sessionID,
			"expires_at": expiresAt,
		},
		"user": map[string]any{
			"id":         user.ID,
			"email":      user.Email,
			"status":     user.Status,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
			"is_new":     isNewUser,
		},
		"principal": map[string]any{
			"sub":   externalPrincipal.Sub,
			"email": externalPrincipal.Email,
			"roles": externalPrincipal.Roles,
		},
	})
}

func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := resolveUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing auth principal")
		return
	}

	user, err := appStore.GetPlatformUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "user profile not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load user profile")
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	connections, _ := appStore.ListAWSConnectionsByUser(r.Context(), userID)

	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":         user.ID,
			"email":      user.Email,
			"status":     user.Status,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
		"principal": map[string]any{
			"sub":        principal.Sub,
			"email":      principal.Email,
			"roles":      principal.Roles,
			"session_id": principal.SessionID,
			"expires_at": principal.ExpiresAt,
		},
		"aws_connection_count": len(connections),
	})
}

func PostLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.SessionID) == "" {
		writeJSONError(w, http.StatusBadRequest, "session_id missing from auth token")
		return
	}

	err := appStore.RevokeAuthSession(r.Context(), principal.SessionID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeJSONError(w, http.StatusInternalServerError, "failed to revoke session")
		return
	}

	_ = appStore.CreateAuditEvent(r.Context(), store.AuditEventInput{
		ActorUserID: principal.UserID,
		EventType:   "auth.session.logout",
		TargetType:  "auth_session",
		TargetID:    principal.SessionID,
		Status:      "success",
		Message:     "session revoked",
	})

	w.WriteHeader(http.StatusNoContent)
}

func provisionPlatformUser(r *http.Request, p auth.Principal) (store.PlatformUser, bool, error) {
	const provider = "cognito"

	user, err := appStore.GetPlatformUserByIdentity(r.Context(), provider, p.Sub)
	if err == nil {
		_, _ = appStore.UpsertAuthIdentity(r.Context(), store.UpsertAuthIdentityInput{
			UserID:   user.ID,
			Provider: provider,
			Subject:  p.Sub,
			Email:    p.Email,
		})
		return user, false, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.PlatformUser{}, false, err
	}

	user, err = appStore.CreatePlatformUser(r.Context(), store.CreatePlatformUserInput{
		Email:  p.Email,
		Status: "active",
	})
	if err != nil {
		return store.PlatformUser{}, false, err
	}

	_, err = appStore.UpsertAuthIdentity(r.Context(), store.UpsertAuthIdentityInput{
		UserID:   user.ID,
		Provider: provider,
		Subject:  p.Sub,
		Email:    p.Email,
	})
	if err != nil {
		return store.PlatformUser{}, false, err
	}

	_ = appStore.CreateAuditEvent(r.Context(), store.AuditEventInput{
		ActorUserID: user.ID,
		EventType:   "auth.user.provisioned",
		TargetType:  "platform_user",
		TargetID:    strings.TrimSpace(p.Sub),
		Status:      "success",
		Message:     "new platform user provisioned",
		Metadata:    "{\"provider\":\"cognito\"}",
	})

	return user, true, nil
}

func defaultTokenIssuer(secret, issuer, audience string) auth.TokenIssuer {
	return auth.TokenIssuer{
		Issuer:   strings.TrimSpace(issuer),
		Audience: strings.TrimSpace(audience),
		Secret:   []byte(strings.TrimSpace(secret)),
		TTL:      12 * time.Hour,
	}
}
