package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"labra-backend/internal/api/store"
)

type deploymentQueueConfig struct {
	PollInterval     time.Duration
	Concurrency      int
	MaxAttempts      int64
	RetryBaseSeconds int64
	RetryMaxSeconds  int64
}

type deploymentFailure struct {
	Message   string
	Category  string
	Retryable bool
}

func (e deploymentFailure) Error() string {
	return e.Message
}

var (
	queueMu     sync.Mutex
	queueCancel context.CancelFunc
	queueDone   chan struct{}
	queueWorker string
)

func StartDeploymentQueueWorker() error {
	queueMu.Lock()
	defer queueMu.Unlock()

	if appStore == nil {
		return fmt.Errorf("store not initialized")
	}
	if queueCancel != nil {
		return nil
	}

	cfg := loadDeploymentQueueConfig()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	workerID := fmt.Sprintf("worker-%d", time.Now().UnixNano())

	queueCancel = cancel
	queueDone = done
	queueWorker = workerID

	go runDeploymentQueueLoop(ctx, done, cfg, workerID)
	return nil
}

func StopDeploymentQueueWorker() {
	queueMu.Lock()
	cancel := queueCancel
	done := queueDone
	queueCancel = nil
	queueDone = nil
	queueWorker = ""
	queueMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func loadDeploymentQueueConfig() deploymentQueueConfig {
	return deploymentQueueConfig{
		PollInterval:     envDurationMs("DEPLOY_QUEUE_POLL_MS", 200, 25, 5000),
		Concurrency:      envInt("DEPLOY_WORKER_CONCURRENCY", 2, 1, 16),
		MaxAttempts:      int64(envInt("DEPLOY_MAX_ATTEMPTS", 3, 1, 10)),
		RetryBaseSeconds: int64(envInt("DEPLOY_RETRY_BASE_SECONDS", 2, 1, 60)),
		RetryMaxSeconds:  int64(envInt("DEPLOY_RETRY_MAX_SECONDS", 60, 5, 600)),
	}
}

func envInt(key string, defaultV, minV, maxV int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultV
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultV
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func envDurationMs(key string, defaultMs, minMs, maxMs int) time.Duration {
	v := envInt(key, defaultMs, minMs, maxMs)
	return time.Duration(v) * time.Millisecond
}

func enqueueDeploymentJob(ctx context.Context, dep store.Deployment) (store.DeploymentJob, error) {
	job, err := appStore.CreateDeploymentJob(ctx, store.CreateDeploymentJobInput{
		DeploymentID: dep.ID,
		AppID:        dep.AppID,
		UserID:       dep.UserID,
		MaxAttempts:  loadDeploymentQueueConfig().MaxAttempts,
	})
	if err == nil {
		return job, nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") {
		return appStore.GetDeploymentJobByDeploymentID(ctx, dep.ID)
	}
	return store.DeploymentJob{}, err
}

func runDeploymentQueueLoop(ctx context.Context, done chan struct{}, cfg deploymentQueueConfig, workerID string) {
	defer close(done)

	sem := make(chan struct{}, cfg.Concurrency)
	sleep := time.NewTicker(cfg.PollInterval)
	defer sleep.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := appStore.ClaimNextRunnableDeploymentJob(ctx, workerID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				select {
				case <-ctx.Done():
					return
				case <-sleep.C:
					continue
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-sleep.C:
				continue
			}
		}

		sem <- struct{}{}
		go func(job store.DeploymentJob) {
			defer func() { <-sem }()
			processDeploymentJob(ctx, job, cfg)
		}(job)
	}
}

func processDeploymentJob(ctx context.Context, job store.DeploymentJob, cfg deploymentQueueConfig) {
	dep, err := appStore.GetDeploymentByIDForUser(ctx, job.DeploymentID, job.UserID)
	if err != nil {
		_, _ = appStore.MarkDeploymentJobFailed(ctx, job.ID, "deployment not found for queued job", "data")
		return
	}

	app, err := appStore.GetAppByIDForUser(ctx, dep.AppID, dep.UserID)
	if err != nil {
		_, _ = appStore.MarkDeploymentJobFailed(ctx, job.ID, "app not found for queued job", "data")
		_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "failed", "app not found for queued job", "data", false, "", dep.StartedAt, store.UnixNow())
		return
	}

	start := dep.StartedAt
	if start <= 0 {
		start = store.UnixNow()
	}
	_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "running", "", "", false, dep.SiteURL, start, 0)
	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "info", fmt.Sprintf("worker picked up job attempt %d/%d", job.AttemptCount, job.MaxAttempts))

	siteURL, execErr := executeDeploymentAttempt(ctx, dep, app, job.AttemptCount)
	if execErr == nil {
		finish := store.UnixNow()
		_, _ = appStore.MarkDeploymentJobSucceeded(ctx, job.ID)
		_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "succeeded", "", "", false, siteURL, start, finish)
		_ = appStore.RecordAppDeploymentOutcome(ctx, dep.AppID, "succeeded", start, finish, normalizedTrigger(dep.TriggerType))
		return
	}

	category, retryable, reason := classifyDeploymentFailure(execErr)
	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "error", fmt.Sprintf("attempt %d failed: %s", job.AttemptCount, reason))

	if retryable && job.AttemptCount < job.MaxAttempts {
		delay := retryDelaySeconds(job.AttemptCount, cfg)
		nextAttempt := store.UnixNow() + delay
		_, _ = appStore.MarkDeploymentJobRetry(ctx, job.ID, nextAttempt, reason, category)
		_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "queued", reason, category, true, dep.SiteURL, start, 0)
		_ = appStore.CreateDeploymentLog(ctx, dep.ID, "warn", fmt.Sprintf("retry scheduled in %ds (attempt %d/%d)", delay, job.AttemptCount+1, job.MaxAttempts))
		return
	}

	finish := store.UnixNow()
	_, _ = appStore.MarkDeploymentJobFailed(ctx, job.ID, reason, category)
	_, _ = appStore.UpdateDeploymentOutcome(ctx, dep.ID, "failed", reason, category, retryable, "", start, finish)
	_ = appStore.RecordAppDeploymentOutcome(ctx, dep.AppID, "failed", start, finish, normalizedTrigger(dep.TriggerType))
}

func executeDeploymentAttempt(ctx context.Context, dep store.Deployment, app store.App, attempt int64) (string, error) {
	switch normalizedTrigger(dep.TriggerType) {
	case "rollback":
		return executeRollbackAttempt(ctx, dep, app)
	default:
		return executeStandardAttempt(ctx, dep.ID, app, attempt)
	}
}

func executeStandardAttempt(ctx context.Context, deploymentID int64, app store.App, attempt int64) (string, error) {
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "clone repository")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("checkout branch %s", app.Branch))

	envVars, envErr := appStore.ListAppEnvVarsForApp(ctx, app.ID)
	if envErr != nil {
		_ = appStore.CreateDeploymentLog(ctx, deploymentID, "warn", "unable to load app env vars; continuing without injected env vars")
		envVars = nil
	}
	deploymentEnv := buildDeploymentEnv(envVars)
	_ = deploymentEnv // placeholder until runner integration
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", describeEnvInjection(envVars))

	if forcedTransientFailures(envVars) >= attempt {
		return "", deploymentFailure{
			Message:   "simulated transient deployment error",
			Category:  "transient",
			Retryable: true,
		}
	}
	if hasEnvFlag(envVars, "LABRA_FORCE_PERMANENT_FAILURE") {
		return "", deploymentFailure{
			Message:   "simulated permanent deployment error",
			Category:  "build",
			Retryable: false,
		}
	}

	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "install dependencies")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "run build")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "upload static artifacts")
	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "invalidate CDN cache")

	if app.BuildType != "static" {
		return "", deploymentFailure{
			Message:   "unsupported build type",
			Category:  "configuration",
			Retryable: false,
		}
	}

	siteURL := strings.TrimSpace(app.SiteURL)
	if siteURL == "" {
		siteURL = fmt.Sprintf("https://%s.preview.labra.local", slugify(app.Name))
	}

	finish := store.UnixNow()
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

	_ = appStore.CreateDeploymentLog(ctx, deploymentID, "info", "deployment completed successfully")
	return siteURL, nil
}

func executeRollbackAttempt(ctx context.Context, dep store.Deployment, app store.App) (string, error) {
	payload, err := appStore.GetDeploymentRollbackPayload(ctx, dep.ID)
	if err != nil {
		return "", deploymentFailure{
			Message:   "rollback payload missing",
			Category:  "data",
			Retryable: false,
		}
	}

	targetRelease, err := appStore.GetReleaseVersionByIDForUser(ctx, app.ID, payload.TargetReleaseID, app.UserID)
	if err != nil {
		return "", deploymentFailure{
			Message:   "target release not found",
			Category:  "data",
			Retryable: false,
		}
	}

	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "info", fmt.Sprintf("loading release v%d artifact", targetRelease.VersionNumber))
	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "info", fmt.Sprintf("switch current release pointer -> %d", targetRelease.ID))

	if err := appStore.SetCurrentReleaseVersionForAppForUser(ctx, app.ID, targetRelease.ID, app.UserID); err != nil {
		return "", deploymentFailure{
			Message:   "unable to switch release pointer",
			Category:  "internal",
			Retryable: true,
		}
	}

	if err := appStore.AttachReleaseToDeployment(ctx, dep.ID, targetRelease.ID); err != nil {
		return "", deploymentFailure{
			Message:   "unable to attach target release to rollback deployment",
			Category:  "internal",
			Retryable: true,
		}
	}

	if _, err := appStore.CreateRollbackEvent(ctx, store.CreateRollbackEventInput{
		AppID:         app.ID,
		UserID:        app.UserID,
		FromReleaseID: payload.FromReleaseID,
		ToReleaseID:   targetRelease.ID,
		DeploymentID:  dep.ID,
		Reason:        payload.Reason,
	}); err != nil && !strings.Contains(strings.ToLower(err.Error()), "unique") {
		return "", deploymentFailure{
			Message:   "unable to persist rollback record",
			Category:  "internal",
			Retryable: true,
		}
	}

	targetURL := strings.TrimSpace(app.SiteURL)
	if targetURL == "" {
		targetDep, err := appStore.GetDeploymentByIDForUser(ctx, targetRelease.DeploymentID, app.UserID)
		if err == nil {
			targetURL = strings.TrimSpace(targetDep.SiteURL)
		}
	}
	if targetURL == "" {
		targetURL = fmt.Sprintf("https://%s.preview.labra.local", slugify(app.Name))
	}

	_ = appStore.CreateDeploymentLog(ctx, dep.ID, "info", fmt.Sprintf("rollback complete: now serving release v%d", targetRelease.VersionNumber))
	return targetURL, nil
}

func classifyDeploymentFailure(err error) (category string, retryable bool, reason string) {
	var f deploymentFailure
	if errors.As(err, &f) {
		return strings.TrimSpace(f.Category), f.Retryable, strings.TrimSpace(f.Message)
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		msg = "deployment failed"
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "timeout") || strings.Contains(lower, "temporar") || strings.Contains(lower, "database is locked") {
		return "transient", true, msg
	}
	return "internal", false, msg
}

func retryDelaySeconds(attempt int64, cfg deploymentQueueConfig) int64 {
	if attempt <= 0 {
		attempt = 1
	}
	delay := cfg.RetryBaseSeconds
	for i := int64(1); i < attempt; i++ {
		delay *= 2
		if delay >= cfg.RetryMaxSeconds {
			return cfg.RetryMaxSeconds
		}
	}
	if delay <= 0 {
		return cfg.RetryBaseSeconds
	}
	if delay > cfg.RetryMaxSeconds {
		return cfg.RetryMaxSeconds
	}
	return delay
}

func normalizedTrigger(v string) string {
	n := strings.TrimSpace(strings.ToLower(v))
	if n == "" {
		return "manual"
	}
	return n
}

func hasEnvFlag(envVars []store.AppEnvVar, key string) bool {
	want := strings.TrimSpace(strings.ToUpper(key))
	for _, envVar := range envVars {
		if strings.TrimSpace(strings.ToUpper(envVar.Key)) != want {
			continue
		}
		value := strings.TrimSpace(strings.ToLower(envVar.Value))
		return value == "1" || value == "true" || value == "yes"
	}
	return false
}

func forcedTransientFailures(envVars []store.AppEnvVar) int64 {
	for _, envVar := range envVars {
		if strings.TrimSpace(strings.ToUpper(envVar.Key)) != "LABRA_FORCE_TRANSIENT_FAILURES" {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(envVar.Value), 10, 64)
		if err != nil || v < 0 {
			return 0
		}
		if v > 10 {
			return 10
		}
		return v
	}
	return 0
}

func GetDeployQueueStatusHandler(w http.ResponseWriter, r *http.Request) {
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

	if _, err := appStore.GetDeploymentByIDForUser(r.Context(), deploymentID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "deployment not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment")
		return
	}

	job, err := appStore.GetDeploymentJobByDeploymentID(r.Context(), deploymentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "deployment job not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load deployment queue status")
		return
	}

	nextRetryIn := int64(0)
	now := store.UnixNow()
	if job.NextAttemptAt > now && (job.Status == "queued" || job.Status == "retrying") {
		nextRetryIn = job.NextAttemptAt - now
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"deployment_id":           deploymentID,
		"job":                     job,
		"next_retry_in_seconds":   nextRetryIn,
		"worker":                  queueWorker,
		"queue_poll_interval_ms":  int(loadDeploymentQueueConfig().PollInterval / time.Millisecond),
		"worker_concurrency":      loadDeploymentQueueConfig().Concurrency,
		"configured_max_attempts": loadDeploymentQueueConfig().MaxAttempts,
	})
}
