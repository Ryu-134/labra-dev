package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
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

	job, err := enqueueDeploymentJob(r.Context(), deployment)
	if err != nil {
		_, _ = appStore.UpdateDeploymentOutcome(r.Context(), deployment.ID, "failed", "unable to enqueue deployment job", "queue", false, "", 0, store.UnixNow())
		_ = appStore.CreateDeploymentLog(r.Context(), deployment.ID, "error", "deployment queue enqueue failed")
		writeJSONError(w, http.StatusInternalServerError, "failed to enqueue deployment job")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"deployment": deployment,
		"job":        job,
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
	dep, err := appStore.GetDeploymentByIDForUser(ctx, deploymentID, app.UserID)
	if err != nil {
		return
	}

	start := store.UnixNow()
	_, _ = appStore.UpdateDeploymentOutcome(ctx, deploymentID, "running", "", "", false, app.SiteURL, start, 0)
	siteURL, execErr := executeStandardAttempt(ctx, dep.ID, app, 1)
	if execErr != nil {
		category, retryable, reason := classifyDeploymentFailure(execErr)
		finish := store.UnixNow()
		_, _ = appStore.UpdateDeploymentOutcome(ctx, deploymentID, "failed", reason, category, retryable, "", start, finish)
		_ = appStore.RecordAppDeploymentOutcome(ctx, app.ID, "failed", start, finish, normalizedTrigger(dep.TriggerType))
		return
	}

	finish := store.UnixNow()
	_, _ = appStore.UpdateDeploymentOutcome(ctx, deploymentID, "succeeded", "", "", false, siteURL, start, finish)
	_ = appStore.RecordAppDeploymentOutcome(ctx, app.ID, "succeeded", start, finish, normalizedTrigger(dep.TriggerType))
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

func describeEnvInjection(envVars []store.AppEnvVar) string {
	if len(envVars) == 0 {
		return "inject 0 env vars"
	}

	secretCount := 0
	for _, envVar := range envVars {
		if envVar.IsSecret {
			secretCount++
		}
	}
	return fmt.Sprintf("inject %d env vars (%d secret)", len(envVars), secretCount)
}

func buildDeploymentEnv(envVars []store.AppEnvVar) map[string]string {
	env := make(map[string]string, len(envVars))
	for _, envVar := range envVars {
		env[envVar.Key] = envVar.Value
	}
	return env
}

func releaseRetentionLimit() int {
	raw := strings.TrimSpace(os.Getenv("RELEASE_RETENTION_LIMIT"))
	if raw == "" {
		return 20
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 20
	}
	if v > 200 {
		return 200
	}
	return v
}

func buildArtifactPath(appID, deploymentID, finishedAt int64) string {
	if finishedAt <= 0 {
		finishedAt = store.UnixNow()
	}
	return fmt.Sprintf("releases/app-%d/deploy-%d-%d.tgz", appID, deploymentID, finishedAt)
}
