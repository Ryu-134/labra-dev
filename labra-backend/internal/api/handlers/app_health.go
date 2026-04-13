package handlers

import (
	"errors"
	"net/http"
	"strings"

	"labra-backend/internal/api/store"
)

type healthDeploymentSummary struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	TriggerType string `json:"trigger_type"`
	SiteURL     string `json:"site_url,omitempty"`
	CommitSHA   string `json:"commit_sha,omitempty"`
	UpdatedAt   int64  `json:"updated_at"`
}

type appHealthResponse struct {
	AppID                int64                    `json:"app_id"`
	AppName              string                   `json:"app_name"`
	RepoFullName         string                   `json:"repo_full_name"`
	Branch               string                   `json:"branch"`
	CurrentURL           string                   `json:"current_url"`
	LatestDeployStatus   string                   `json:"latest_deploy_status"`
	LatestDeploy         *healthDeploymentSummary `json:"latest_deploy,omitempty"`
	LastSuccessfulDeploy *healthDeploymentSummary `json:"last_successful_deploy,omitempty"`
	Metrics              healthMetricsResponse    `json:"metrics"`
	AlarmState           *string                  `json:"alarm_state,omitempty"`
	HealthIndicator      string                   `json:"health_indicator"`
}

type healthMetricsResponse struct {
	SuccessCount int64   `json:"success_count"`
	FailureCount int64   `json:"failure_count"`
	TotalCount   int64   `json:"total_count"`
	SuccessRate  float64 `json:"success_rate"`
}

func GetAppHealthSummaryHandler(w http.ResponseWriter, r *http.Request) {
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

	metrics, err := appStore.GetAppHealthMetricsForUser(r.Context(), appID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load health metrics")
		return
	}

	latest, latestErr := appStore.GetLatestDeploymentByAppForUser(r.Context(), appID, userID)
	var latestSummary *healthDeploymentSummary
	if latestErr == nil {
		latestSummary = toHealthDeploymentSummary(latest)
	} else if !errors.Is(latestErr, store.ErrNotFound) {
		writeJSONError(w, http.StatusInternalServerError, "failed to load latest deployment")
		return
	}

	lastSuccess, successErr := appStore.GetLastSuccessfulDeploymentByAppForUser(r.Context(), appID, userID)
	var lastSuccessSummary *healthDeploymentSummary
	if successErr == nil {
		lastSuccessSummary = toHealthDeploymentSummary(lastSuccess)
	} else if !errors.Is(successErr, store.ErrNotFound) {
		writeJSONError(w, http.StatusInternalServerError, "failed to load last successful deployment")
		return
	}

	envVars, envErr := appStore.ListAppEnvVarsForApp(r.Context(), appID)
	if envErr != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load app env vars")
		return
	}
	alarmState := resolveAlarmState(envVars)

	currentURL := strings.TrimSpace(app.SiteURL)
	if latestSummary != nil && strings.TrimSpace(latestSummary.SiteURL) != "" {
		currentURL = latestSummary.SiteURL
	}
	if currentURL == "" && lastSuccessSummary != nil {
		currentURL = lastSuccessSummary.SiteURL
	}

	total := metrics.SuccessCount + metrics.FailureCount
	successRate := 0.0
	if total > 0 {
		successRate = (float64(metrics.SuccessCount) / float64(total)) * 100
	}

	latestStatus := "unknown"
	if latestSummary != nil {
		latestStatus = latestSummary.Status
	}

	writeJSON(w, http.StatusOK, appHealthResponse{
		AppID:                app.ID,
		AppName:              app.Name,
		RepoFullName:         app.RepoFullName,
		Branch:               app.Branch,
		CurrentURL:           currentURL,
		LatestDeployStatus:   latestStatus,
		LatestDeploy:         latestSummary,
		LastSuccessfulDeploy: lastSuccessSummary,
		Metrics: healthMetricsResponse{
			SuccessCount: metrics.SuccessCount,
			FailureCount: metrics.FailureCount,
			TotalCount:   total,
			SuccessRate:  successRate,
		},
		AlarmState:      alarmState,
		HealthIndicator: computeHealthIndicator(latestSummary),
	})
}

func toHealthDeploymentSummary(dep store.Deployment) *healthDeploymentSummary {
	return &healthDeploymentSummary{
		ID:          dep.ID,
		Status:      dep.Status,
		TriggerType: dep.TriggerType,
		SiteURL:     dep.SiteURL,
		CommitSHA:   dep.CommitSHA,
		UpdatedAt:   dep.UpdatedAt,
	}
}

func resolveAlarmState(envVars []store.AppEnvVar) *string {
	for _, envVar := range envVars {
		if envVar.IsSecret {
			continue
		}
		key := strings.TrimSpace(strings.ToUpper(envVar.Key))
		if key == "LABRA_ALARM_STATE" || key == "ALARM_STATE" {
			v := strings.TrimSpace(envVar.Value)
			if v == "" {
				return nil
			}
			return &v
		}
	}
	return nil
}

func computeHealthIndicator(latest *healthDeploymentSummary) string {
	if latest == nil {
		return "unknown"
	}

	switch strings.ToLower(strings.TrimSpace(latest.Status)) {
	case "succeeded":
		return "healthy"
	case "queued", "running":
		return "degraded"
	case "failed":
		return "unhealthy"
	default:
		return "unknown"
	}
}
