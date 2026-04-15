package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"labra-backend/internal/api/auth"
	"labra-backend/internal/api/store"
)

var (
	roleARNPattern = regexp.MustCompile(`^arn:aws:iam::([0-9]{12}):role\/[A-Za-z0-9+=,.@_\/-]+$`)
	regionPattern  = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-[0-9]+$`)
)

type upsertAWSConnectionRequest struct {
	RoleARN    string `json:"role_arn"`
	ExternalID string `json:"external_id"`
	Region     string `json:"region"`
	AccountID  string `json:"account_id,omitempty"`
}

func UpsertAWSConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := resolveUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing auth principal or X-User-ID header")
		return
	}

	var body upsertAWSConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	normalized, err := normalizeAWSConnection(body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		_ = appStore.CreateAuditEvent(r.Context(), store.AuditEventInput{
			ActorUserID: userID,
			EventType:   "aws_connection.upsert",
			TargetType:  "aws_connection",
			Status:      "failed",
			Message:     err.Error(),
		})
		return
	}
	normalized.UserID = userID
	normalized.Status = "validated"
	normalized.LastValidatedAt = store.UnixNow()

	connection, err := appStore.UpsertAWSConnection(r.Context(), normalized)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save aws connection")
		_ = appStore.CreateAuditEvent(r.Context(), store.AuditEventInput{
			ActorUserID: userID,
			EventType:   "aws_connection.upsert",
			TargetType:  "aws_connection",
			Status:      "failed",
			Message:     "failed to save aws connection",
		})
		return
	}

	_ = appStore.CreateAuditEvent(r.Context(), store.AuditEventInput{
		ActorUserID: userID,
		EventType:   "aws_connection.upsert",
		TargetType:  "aws_connection",
		TargetID:    fmt.Sprintf("%d", connection.ID),
		Status:      "success",
		Message:     "aws connection validated and saved",
	})

	writeJSON(w, http.StatusCreated, map[string]any{
		"connection": connection,
	})
}

func ListAWSConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := resolveUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing auth principal or X-User-ID header")
		return
	}

	connections, err := appStore.ListAWSConnectionsByUser(r.Context(), userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load aws connections")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"aws_connections": connections,
	})
}

func resolveUserID(r *http.Request) (int64, bool) {
	if principal, ok := auth.PrincipalFromContext(r.Context()); ok && principal.UserID > 0 {
		return principal.UserID, true
	}
	return readUserID(r)
}

func normalizeAWSConnection(req upsertAWSConnectionRequest) (store.UpsertAWSConnectionInput, error) {
	roleARN := strings.TrimSpace(req.RoleARN)
	externalID := strings.TrimSpace(req.ExternalID)
	region := strings.TrimSpace(req.Region)
	accountID := strings.TrimSpace(req.AccountID)

	if roleARN == "" {
		return store.UpsertAWSConnectionInput{}, fmt.Errorf("role_arn is required")
	}
	matches := roleARNPattern.FindStringSubmatch(roleARN)
	if len(matches) != 2 {
		return store.UpsertAWSConnectionInput{}, fmt.Errorf("role_arn must match arn:aws:iam::<account-id>:role/<role-name>")
	}
	if externalID == "" {
		return store.UpsertAWSConnectionInput{}, fmt.Errorf("external_id is required")
	}
	if len(externalID) < 8 || len(externalID) > 128 {
		return store.UpsertAWSConnectionInput{}, fmt.Errorf("external_id must be between 8 and 128 characters")
	}
	if !regionPattern.MatchString(region) {
		return store.UpsertAWSConnectionInput{}, fmt.Errorf("region must look like us-west-2")
	}
	if accountID == "" {
		accountID = matches[1]
	}
	if len(accountID) != 12 || strings.Trim(accountID, "0123456789") != "" {
		return store.UpsertAWSConnectionInput{}, fmt.Errorf("account_id must be a 12-digit AWS account number")
	}

	return store.UpsertAWSConnectionInput{
		RoleARN:    roleARN,
		ExternalID: externalID,
		Region:     region,
		AccountID:  accountID,
	}, nil
}
