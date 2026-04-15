package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"labra-backend/internal/api/store"
)

func enqueueWebhookDeployments(
	r *http.Request,
	payload githubPushEvent,
	eligibleApps []map[string]any,
) ([]map[string]any, error) {
	triggered := make([]map[string]any, 0, len(eligibleApps))

	branch, _ := extractBranch(payload.Ref)
	commitSHA := strings.TrimSpace(payload.After)
	if commitSHA == "" {
		commitSHA = strings.TrimSpace(payload.HeadCommit.ID)
	}
	commitMessage := strings.TrimSpace(payload.HeadCommit.Message)
	commitAuthor := strings.TrimSpace(payload.HeadCommit.Author.Name)
	deliveryID := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))

	for _, eligible := range eligibleApps {
		appID, err := mustInt64(eligible["id"])
		if err != nil {
			return nil, fmt.Errorf("invalid eligible app id: %w", err)
		}
		userID, err := mustInt64(eligible["user_id"])
		if err != nil {
			return nil, fmt.Errorf("invalid eligible app user_id: %w", err)
		}

		app, err := appStore.GetAppByIDForUser(r.Context(), appID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to load app for webhook deploy: %w", err)
		}

		deployment, err := appStore.CreateDeployment(r.Context(), store.CreateDeploymentInput{
			AppID:         app.ID,
			UserID:        app.UserID,
			Status:        "queued",
			TriggerType:   "webhook",
			CommitSHA:     commitSHA,
			CommitMessage: commitMessage,
			CommitAuthor:  commitAuthor,
			Branch:        branch,
			SiteURL:       app.SiteURL,
			CorrelationID: fmt.Sprintf("webhook-%s-%d-%d", deliveryID, app.ID, time.Now().UnixNano()),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create webhook deployment: %w", err)
		}

		_ = appStore.CreateDeploymentLog(r.Context(), deployment.ID, "info", "deployment queued by webhook trigger")
		triggerDeployment(deployment.ID, app)

		triggered = append(triggered, map[string]any{
			"app_id":        app.ID,
			"deployment_id": deployment.ID,
		})
	}

	return triggered, nil
}
