package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"labra-backend/internal/api/store"
)

var envVarKeyPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

type createAppEnvVarRequest struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret *bool  `json:"is_secret"`
}

type updateAppEnvVarRequest struct {
	Key      *string `json:"key"`
	Value    *string `json:"value"`
	IsSecret *bool   `json:"is_secret"`
}

type appEnvVarResponse struct {
	ID        int64  `json:"id"`
	AppID     int64  `json:"app_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsSecret  bool   `json:"is_secret"`
	Masked    bool   `json:"masked"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func ListAppEnvVarsHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := appStore.GetAppByIDForUser(r.Context(), appID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load app")
		return
	}

	envVars, err := appStore.ListAppEnvVarsByAppForUser(r.Context(), appID, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list env vars")
		return
	}

	response := make([]appEnvVarResponse, 0, len(envVars))
	for _, envVar := range envVars {
		response = append(response, toAppEnvVarResponse(envVar))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":   appID,
		"env_vars": response,
	})
}

func CreateAppEnvVarHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body createAppEnvVarRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	key, err := normalizeEnvVarKey(body.Key)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	value := strings.TrimSpace(body.Value)
	if err := validateEnvVarValue(value); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	isSecret := false
	if body.IsSecret != nil {
		isSecret = *body.IsSecret
	}

	envVar, err := appStore.CreateAppEnvVar(r.Context(), appID, userID, store.CreateAppEnvVarInput{
		Key:      key,
		Value:    value,
		IsSecret: isSecret,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeJSONError(w, http.StatusConflict, "env var key already exists for this app")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to create env var")
		return
	}

	writeJSON(w, http.StatusCreated, toAppEnvVarResponse(envVar))
}

func PatchAppEnvVarHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	envVarID, err := readEnvVarIDFromPath(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body updateAppEnvVarRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	current, err := appStore.GetAppEnvVarByIDForUser(r.Context(), appID, envVarID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "env var not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load env var")
		return
	}

	next := store.UpdateAppEnvVarInput{
		Key:      current.Key,
		Value:    current.Value,
		IsSecret: current.IsSecret,
	}

	if body.Key != nil {
		nextKey, err := normalizeEnvVarKey(*body.Key)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		next.Key = nextKey
	}
	if body.Value != nil {
		nextValue := strings.TrimSpace(*body.Value)
		if err := validateEnvVarValue(nextValue); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		next.Value = nextValue
	}
	if body.IsSecret != nil {
		next.IsSecret = *body.IsSecret
	}

	updated, err := appStore.UpdateAppEnvVarForUser(r.Context(), appID, envVarID, userID, next)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "env var not found")
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeJSONError(w, http.StatusConflict, "env var key already exists for this app")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to update env var")
		return
	}

	writeJSON(w, http.StatusOK, toAppEnvVarResponse(updated))
}

func DeleteAppEnvVarHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	envVarID, err := readEnvVarIDFromPath(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := appStore.DeleteAppEnvVarForUser(r.Context(), appID, envVarID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "env var not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to delete env var")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toAppEnvVarResponse(envVar store.AppEnvVar) appEnvVarResponse {
	value := envVar.Value
	masked := false
	if envVar.IsSecret {
		masked = true
		if value != "" {
			value = "********"
		}
	}

	return appEnvVarResponse{
		ID:        envVar.ID,
		AppID:     envVar.AppID,
		Key:       envVar.Key,
		Value:     value,
		IsSecret:  envVar.IsSecret,
		Masked:    masked,
		CreatedAt: envVar.CreatedAt,
		UpdatedAt: envVar.UpdatedAt,
	}
}

func normalizeEnvVarKey(raw string) (string, error) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return "", fmt.Errorf("key is required")
	}
	if len(key) > 128 {
		return "", fmt.Errorf("key exceeds 128 characters")
	}
	if !envVarKeyPattern.MatchString(key) {
		return "", fmt.Errorf("key must match [A-Z_][A-Z0-9_]*")
	}
	return key, nil
}

func validateEnvVarValue(value string) error {
	if len(value) > 8192 {
		return fmt.Errorf("value exceeds 8192 characters")
	}
	return nil
}

func readEnvVarIDFromPath(r *http.Request) (int64, error) {
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segments) < 5 {
		return 0, fmt.Errorf("env var id is required")
	}
	if segments[0] != "v1" || segments[1] != "apps" || segments[3] != "env-vars" {
		return 0, fmt.Errorf("env var id not found in path")
	}

	rawID := strings.TrimSpace(segments[4])
	if rawID == "" {
		return 0, fmt.Errorf("env var id is required")
	}

	envVarID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || envVarID <= 0 {
		return 0, fmt.Errorf("env var id must be a positive integer")
	}
	return envVarID, nil
}
