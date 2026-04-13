package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

func dedupeEligibleAppsWithLedger(
	r *http.Request,
	deliveryID string,
	eventType string,
	payload githubPushEvent,
	eligibleApps []map[string]any,
) ([]map[string]any, int, error) {
	if strings.TrimSpace(deliveryID) == "" {
		return nil, 0, errWebhook("missing X-GitHub-Delivery header")
	}

	uniqueApps := make([]map[string]any, 0, len(eligibleApps))
	duplicateCount := 0
	commitSHA := strings.TrimSpace(payload.After)

	for _, app := range eligibleApps {
		appID, err := mustInt64(app["id"])
		if err != nil {
			return nil, duplicateCount, fmt.Errorf("invalid eligible app id: %w", err)
		}

		claimed, err := appStore.ClaimWebhookDelivery(r.Context(), appID, deliveryID, eventType, commitSHA)
		if err != nil {
			return nil, duplicateCount, fmt.Errorf("failed to claim webhook delivery: %w", err)
		}
		if !claimed {
			duplicateCount++
			continue
		}

		uniqueApps = append(uniqueApps, app)
	}

	return uniqueApps, duplicateCount, nil
}
