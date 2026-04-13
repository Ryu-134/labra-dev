package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

func init() {
	resolvePushEvent = resolvePushEventWithStore
}

func resolvePushEventWithStore(r *http.Request, payload githubPushEvent) (webhookResolution, error) {
	repoFullName := strings.ToLower(strings.TrimSpace(payload.Repository.FullName))
	if repoFullName == "" {
		return webhookResolution{}, errWebhook("repository.full_name is required")
	}

	branch, ok := extractBranch(payload.Ref)
	if !ok {
		return webhookResolution{
			RepoFullName: repoFullName,
			Ignored:      true,
			Reason:       "ref is not a branch push",
		}, nil
	}

	apps, err := appStore.ListAutoDeployAppsByRepo(r.Context(), repoFullName)
	if err != nil {
		return webhookResolution{}, fmt.Errorf("failed to load apps for repository")
	}

	eligibleApps := make([]map[string]any, 0)
	for _, app := range apps {
		if strings.TrimSpace(app.Branch) != branch {
			continue
		}

		eligibleApps = append(eligibleApps, map[string]any{
			"id":         app.ID,
			"name":       app.Name,
			"user_id":    app.UserID,
			"branch":     app.Branch,
			"build_type": app.BuildType,
		})
	}

	fmt.Printf("[webhook] delivery=%s event=push repo=%s branch=%s matched=%d eligible=%d\n",
		strings.TrimSpace(r.Header.Get("X-GitHub-Delivery")),
		repoFullName,
		branch,
		len(apps),
		len(eligibleApps),
	)

	return webhookResolution{
		RepoFullName: repoFullName,
		Branch:       branch,
		MatchedApps:  len(apps),
		EligibleApps: eligibleApps,
		Ignored:      len(eligibleApps) == 0,
		Reason:       reasonForEligibility(eligibleApps),
	}, nil
}

func reasonForEligibility(eligible []map[string]any) string {
	if len(eligible) == 0 {
		return "no apps configured for repo+branch"
	}
	return ""
}
