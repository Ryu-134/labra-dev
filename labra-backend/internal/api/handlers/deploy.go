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
	triggerType := determineDeploymentTriggerType(ctx, deploymentID, app.UserID)

	_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "running", "", app.SiteURL, start, 0)
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "clone repository")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("checkout branch %s", app.Branch))
	envVars, envErr := appStore.ListAppEnvVarsForApp(ctx, app.ID)
	if envErr != nil {
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "warn", "unable to load app env vars; continuing without injected env vars")
		envVars = nil
	}
	deploymentEnv := buildDeploymentEnv(envVars)
	_ = deploymentEnv // placeholder for worker runtime integration in Phase 8
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", describeEnvInjection(envVars))
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "install dependencies")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "run build")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "upload static artifacts")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "invalidate CDN cache")

	if app.BuildType != "static" {
		finish := store.UnixNow()
		_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "failed", "unsupported build type", "", start, finish)
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "error", "deployment failed: unsupported build type")
		_ = appStore.RecordAppDeploymentOutcome(ctx, app.ID, "failed", start, finish, triggerType)
		return
	}

	siteURL := strings.TrimSpace(app.SiteURL)
	if siteURL == "" {
		siteURL = fmt.Sprintf("https://%s.preview.labra.local", slugify(app.Name))
	}

	finish := store.UnixNow()
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "deployment completed successfully")
	_, _ = appStore.UpdateDeploymentStatus(ctx, deploymentID, "succeeded", "", siteURL, start, finish)

	release, releaseErr := appStore.CreateReleaseVersion(ctx, store.CreateReleaseVersionInput{
		AppID:            app.ID,
		DeploymentID:     deploymentID,
		ArtifactPath:     buildArtifactPath(app.ID, deploymentID, finish),
		ArtifactChecksum: fmt.Sprintf("sim-%d-%d", app.ID, deploymentID),
	})
	if releaseErr != nil {
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "warn", "release snapshot metadata failed to persist")
	} else {
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("published release v%d", release.VersionNumber))
		if err := appStore.ApplyReleaseRetentionPolicy(ctx, app.ID, releaseRetentionLimit(), release.ID); err != nil {
			_ = appStore.CreateDeploymentLog(ctx, deploymentID, "warn", "release retention policy failed to apply")
		}
	}

	_ = appStore.RecordAppDeploymentOutcome(ctx, app.ID, "succeeded", start, finish, triggerType)
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

func determineDeploymentTriggerType(ctx context.Context, deploymentID, userID int64) string {
	dep, err := appStore.GetDeploymentByIDForUser(ctx, deploymentID, userID)
	if err != nil {
		return "manual"
	}
	v := strings.TrimSpace(strings.ToLower(dep.TriggerType))
	if v == "" {
		return "manual"
	}
	return v
}
