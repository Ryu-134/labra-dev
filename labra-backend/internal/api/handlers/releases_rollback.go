package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"labra-backend/internal/api/store"
)

type createRollbackRequest struct {
	TargetReleaseID *int64 `json:"target_release_id"`
	Reason          string `json:"reason"`
}

func GetAppReleasesHandler(w http.ResponseWriter, r *http.Request) {
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

	releases, err := appStore.ListReleaseVersionsByAppForUser(r.Context(), appID, userID, 100)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list releases")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":             appID,
		"current_release_id": app.CurrentReleaseID,
		"releases":           releases,
	})
}

func GetAppRollbacksHandler(w http.ResponseWriter, r *http.Request) {
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

	events, err := appStore.ListRollbackEventsByAppForUser(r.Context(), appID, userID, 50)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list rollback history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":    appID,
		"rollbacks": events,
	})
}

func CreateRollbackHandler(w http.ResponseWriter, r *http.Request) {
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

	var body createRollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	reason := strings.TrimSpace(body.Reason)

	currentRelease, err := resolveCurrentRelease(r.Context(), appID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusBadRequest, "no release available for rollback")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve current release")
		return
	}

	var targetRelease store.ReleaseVersion
	if body.TargetReleaseID != nil {
		targetID := *body.TargetReleaseID
		if targetID <= 0 {
			writeJSONError(w, http.StatusBadRequest, "target_release_id must be a positive integer")
			return
		}
		targetRelease, err = appStore.GetReleaseVersionByIDForUser(r.Context(), appID, targetID, userID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "target release not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to load target release")
			return
		}
	} else {
		targetRelease, err = appStore.GetPreviousReleaseVersionByAppForUser(r.Context(), appID, userID, currentRelease.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeJSONError(w, http.StatusBadRequest, "no previous release available for rollback")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to resolve previous release")
			return
		}
	}

	if targetRelease.ID == currentRelease.ID {
		writeJSONError(w, http.StatusBadRequest, "target release is already current")
		return
	}

	deployment, err := appStore.CreateDeployment(r.Context(), store.CreateDeploymentInput{
		AppID:         app.ID,
		UserID:        userID,
		Status:        "queued",
		TriggerType:   "rollback",
		Branch:        app.Branch,
		SiteURL:       app.SiteURL,
		CommitMessage: fmt.Sprintf("rollback to release v%d", targetRelease.VersionNumber),
		CorrelationID: fmt.Sprintf("rollback-%d-%d", app.ID, time.Now().UnixNano()),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create rollback deployment")
		return
	}

	_ = appStore.CreateDeploymentLog(r.Context(), deployment.ID, "info", fmt.Sprintf("rollback queued to release v%d", targetRelease.VersionNumber))
	if err := appStore.CreateDeploymentRollbackPayload(r.Context(), deployment.ID, currentRelease.ID, targetRelease.ID, reason); err != nil {
		_, _ = appStore.UpdateDeploymentOutcome(r.Context(), deployment.ID, "failed", "unable to persist rollback payload", "queue", false, "", 0, store.UnixNow())
		_ = appStore.CreateDeploymentLog(r.Context(), deployment.ID, "error", "rollback payload persistence failed")
		writeJSONError(w, http.StatusInternalServerError, "failed to persist rollback payload")
		return
	}
	job, err := enqueueDeploymentJob(r.Context(), deployment)
	if err != nil {
		_, _ = appStore.UpdateDeploymentOutcome(r.Context(), deployment.ID, "failed", "unable to enqueue rollback deployment job", "queue", false, "", 0, store.UnixNow())
		_ = appStore.CreateDeploymentLog(r.Context(), deployment.ID, "error", "rollback deployment queue enqueue failed")
		writeJSONError(w, http.StatusInternalServerError, "failed to enqueue rollback deployment job")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"deployment":        deployment,
		"job":               job,
		"from_release_id":   currentRelease.ID,
		"target_release_id": targetRelease.ID,
	})
}

func resolveCurrentRelease(ctx context.Context, appID, userID int64) (store.ReleaseVersion, error) {
	currentRelease, err := appStore.GetCurrentReleaseVersionByAppForUser(ctx, appID, userID)
	if err == nil {
		return currentRelease, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.ReleaseVersion{}, err
	}

	releases, listErr := appStore.ListReleaseVersionsByAppForUser(ctx, appID, userID, 1)
	if listErr != nil {
		return store.ReleaseVersion{}, listErr
	}
	if len(releases) == 0 {
		return store.ReleaseVersion{}, store.ErrNotFound
	}

	if setErr := appStore.SetCurrentReleaseVersionForAppForUser(ctx, appID, releases[0].ID, userID); setErr != nil {
		return store.ReleaseVersion{}, setErr
	}
	return releases[0], nil
}
