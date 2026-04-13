package handlers

import (
	"errors"
	"net/http"

	"labra-backend/internal/api/store"
)

func GetAppDeploysHandler(w http.ResponseWriter, r *http.Request) {
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

	app, err := appStore.GetAppByIDForUser(r.Context(), appID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load app")
		return
	}

	deployments, err := appStore.ListDeploymentsByAppForUser(r.Context(), app.ID, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load app deployments")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":      app.ID,
		"app_name":    app.Name,
		"repo":        app.RepoFullName,
		"branch":      app.Branch,
		"deployments": deployments,
	})
}
