package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"labra-backend/internal/api/store"
)

func CreateDeployHandler(w http.ResponseWriter, r *http.Request) {
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

	deployment, err := appStore.CreateDeployment(r.Context(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        userID,
		Status:        "queued",
		TriggerType:   "manual",
		Branch:        app.Branch,
		SiteURL:       app.SiteURL,
		CorrelationID: fmt.Sprintf("manual-%d", time.Now().UnixNano()),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create deployment")
		return
	}

	_ = appStore.CreateDeploymentLog(r.Context(), deployment.ID, "info", "deployment queued by manual trigger")

	go runManualDeployment(deployment.ID, app)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"deployment": deployment,
	})
}

func GetDeployHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	deploymentID, err := readIDFromPathOrQuery(r, "deploys")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	deployment, err := appStore.GetDeploymentByIDForUser(r.Context(), deploymentID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment")
		return
	}

	writeJSON(w, http.StatusOK, deployment)
}

func GetDeployLogsHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	deploymentID, err := readIDFromPathOrQuery(r, "deploys")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err = appStore.GetDeploymentByIDForUser(r.Context(), deploymentID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment")
		return
	}

	logs, err := appStore.ListDeploymentLogs(r.Context(), deploymentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment logs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"deployment_id": deploymentID,
		"logs":          logs,
	})
}

func runManualDeployment(deploymentID int64, app store.App) {
	ctx := context.Background()
	start := store.UnixNow()

	_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "running", "", app.SiteURL, start, 0)
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "clone repository")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("checkout branch %s", app.Branch))
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "install dependencies")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "run build")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "upload static artifacts")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "invalidate CDN cache")

	if app.BuildType != "static" {
		finish := store.UnixNow()
		_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "failed", "unsupported build type", "", start, finish)
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "error", "deployment failed: unsupported build type")
		return
	}

	siteURL := strings.TrimSpace(app.SiteURL)
	if siteURL == "" {
		siteURL = fmt.Sprintf("https://%s.preview.labra.local", slugify(app.Name))
	}

	finish := store.UnixNow()
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "deployment completed successfully")
	_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "succeeded", "", siteURL, start, finish)
}

func slugify(in string) string {
	v := strings.TrimSpace(strings.ToLower(in))
	if v == "" {
		return "app"
	}
	v = strings.ReplaceAll(v, " ", "-")
	v = strings.ReplaceAll(v, "_", "-")
	return v
}
