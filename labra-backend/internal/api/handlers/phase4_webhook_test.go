package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"labra-backend/internal/api/store"

	_ "github.com/mattn/go-sqlite3"
)

func TestGitHubWebhookRejectsInvalidSignature(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	createTestApp(t, 1, "demo", "owner/repo", "main")

	payload := webhookPayload("refs/heads/main", "owner/repo", "abc123", "feat: test", "Casey")
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/github", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "delivery-invalid-sig")
	req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")

	rr := httptest.NewRecorder()
	GitHubWebhookHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	if got := deploymentCount(t, db); got != 0 {
		t.Fatalf("expected 0 deployments, got %d", got)
	}
}

func TestGitHubWebhookIgnoresWrongBranch(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	createTestApp(t, 1, "demo", "owner/repo", "main")

	payload := webhookPayload("refs/heads/feature", "owner/repo", "abc123", "feat: wrong branch", "Casey")
	req := signedWebhookRequest(payload, "delivery-wrong-branch", "test-secret")

	rr := httptest.NewRecorder()
	GitHubWebhookHandler(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusAccepted, rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if ignored, _ := body["ignored"].(bool); !ignored {
		t.Fatalf("expected ignored=true, got body=%v", body)
	}
	if got := numberAsInt(body["triggered_count"]); got != 0 {
		t.Fatalf("expected triggered_count=0, got %d", got)
	}

	if got := deploymentCount(t, db); got != 0 {
		t.Fatalf("expected 0 deployments, got %d", got)
	}
}

func TestGitHubWebhookDedupeAndAutoTrigger(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	createTestApp(t, 1, "demo", "owner/repo", "main")

	payload := webhookPayload("refs/heads/main", "owner/repo", "abc123", "feat: deploy me", "Casey")
	deliveryID := "delivery-dup-1"

	req1 := signedWebhookRequest(payload, deliveryID, "test-secret")
	rr1 := httptest.NewRecorder()
	GitHubWebhookHandler(rr1, req1)

	if rr1.Code != http.StatusAccepted {
		t.Fatalf("first request expected status %d, got %d, body=%s", http.StatusAccepted, rr1.Code, rr1.Body.String())
	}
	var body1 map[string]any
	if err := json.Unmarshal(rr1.Body.Bytes(), &body1); err != nil {
		t.Fatalf("first response unmarshal: %v", err)
	}
	if got := numberAsInt(body1["triggered_count"]); got != 1 {
		t.Fatalf("expected first triggered_count=1, got %d body=%v", got, body1)
	}
	if got := numberAsInt(body1["duplicate_count"]); got != 0 {
		t.Fatalf("expected first duplicate_count=0, got %d body=%v", got, body1)
	}

	req2 := signedWebhookRequest(payload, deliveryID, "test-secret")
	rr2 := httptest.NewRecorder()
	GitHubWebhookHandler(rr2, req2)

	if rr2.Code != http.StatusAccepted {
		t.Fatalf("second request expected status %d, got %d, body=%s", http.StatusAccepted, rr2.Code, rr2.Body.String())
	}
	var body2 map[string]any
	if err := json.Unmarshal(rr2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("second response unmarshal: %v", err)
	}
	if got := numberAsInt(body2["triggered_count"]); got != 0 {
		t.Fatalf("expected second triggered_count=0, got %d body=%v", got, body2)
	}
	if got := numberAsInt(body2["duplicate_count"]); got != 1 {
		t.Fatalf("expected second duplicate_count=1, got %d body=%v", got, body2)
	}

	if got := deploymentCount(t, db); got != 1 {
		t.Fatalf("expected exactly 1 deployment after duplicate delivery, got %d", got)
	}
}

func TestGetAppDeploysHandlerReturnsWebhookMetadata(t *testing.T) {
	db := setupPhase4TestDB(t)
	defer db.Close()

	app := createTestApp(t, 1, "demo", "owner/repo", "main")

	payload := webhookPayload("refs/heads/main", "owner/repo", "abc123def456", "feat: metadata", "Casey")
	req := signedWebhookRequest(payload, "delivery-history-1", "test-secret")
	rr := httptest.NewRecorder()
	GitHubWebhookHandler(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected webhook status %d, got %d, body=%s", http.StatusAccepted, rr.Code, rr.Body.String())
	}

	historyReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/apps/%d/deploys", app.ID), nil)
	historyReq.Header.Set("X-User-ID", "1")
	historyRR := httptest.NewRecorder()
	GetAppDeploysHandler(historyRR, historyReq)

	if historyRR.Code != http.StatusOK {
		t.Fatalf("expected history status %d, got %d, body=%s", http.StatusOK, historyRR.Code, historyRR.Body.String())
	}

	var body struct {
		Deployments []struct {
			TriggerType   string `json:"trigger_type"`
			CommitSHA     string `json:"commit_sha"`
			CommitMessage string `json:"commit_message"`
			CommitAuthor  string `json:"commit_author"`
			Status        string `json:"status"`
		} `json:"deployments"`
	}
	if err := json.Unmarshal(historyRR.Body.Bytes(), &body); err != nil {
		t.Fatalf("history unmarshal: %v", err)
	}

	if len(body.Deployments) == 0 {
		t.Fatalf("expected at least one deployment in history")
	}

	dep := body.Deployments[0]
	if dep.TriggerType != "webhook" {
		t.Fatalf("expected trigger_type=webhook, got %q", dep.TriggerType)
	}
	if dep.CommitSHA == "" {
		t.Fatalf("expected commit_sha to be populated")
	}
	if dep.CommitMessage == "" {
		t.Fatalf("expected commit_message to be populated")
	}
	if dep.CommitAuthor == "" {
		t.Fatalf("expected commit_author to be populated")
	}
}

func setupPhase4TestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS apps (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  user_id INTEGER NOT NULL,
	  name TEXT NOT NULL,
	  repo_full_name TEXT NOT NULL,
	  branch TEXT NOT NULL DEFAULT 'main',
	  build_type TEXT NOT NULL DEFAULT 'static',
	  output_dir TEXT NOT NULL DEFAULT 'dist',
	  root_dir TEXT,
	  site_url TEXT,
	  auto_deploy_enabled INTEGER NOT NULL DEFAULT 1,
	  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
	);
	CREATE TABLE IF NOT EXISTS deployments (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  app_id INTEGER NOT NULL,
	  user_id INTEGER NOT NULL,
	  status TEXT NOT NULL,
	  trigger_type TEXT NOT NULL,
	  commit_sha TEXT,
	  commit_message TEXT,
	  commit_author TEXT,
	  branch TEXT,
	  site_url TEXT,
	  failure_reason TEXT,
	  correlation_id TEXT,
	  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  started_at INTEGER,
	  finished_at INTEGER
	);
	CREATE TABLE IF NOT EXISTS deployment_logs (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  deployment_id INTEGER NOT NULL,
	  log_level TEXT NOT NULL,
	  message TEXT NOT NULL,
	  created_at INTEGER NOT NULL DEFAULT (unixepoch())
	);
	CREATE TABLE IF NOT EXISTS webhook_deliveries (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  app_id INTEGER NOT NULL,
	  delivery_id TEXT NOT NULL,
	  event_type TEXT NOT NULL,
	  commit_sha TEXT,
	  received_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  UNIQUE(app_id, delivery_id)
	);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	InitAppStore(db)
	InitWebhook("test-secret")
	runDeploymentAsync = false

	return db
}

func createTestApp(t *testing.T, userID int64, name, repo, branch string) store.App {
	t.Helper()

	app, err := appStore.CreateApp(
		context.Background(),
		store.CreateAppInput{
			UserID:            userID,
			Name:              name,
			RepoFullName:      repo,
			Branch:            branch,
			BuildType:         "static",
			OutputDir:         "dist",
			AutoDeployEnabled: true,
		},
	)
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	return app
}

func signedWebhookRequest(payload []byte, deliveryID, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/github", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", deliveryID)
	req.Header.Set("X-Hub-Signature-256", signPayload(secret, payload))
	return req
}

func webhookPayload(ref, repo, sha, message, author string) []byte {
	return []byte(fmt.Sprintf(`{"ref":%q,"after":%q,"repository":{"full_name":%q},"head_commit":{"id":%q,"message":%q,"author":{"name":%q}}}`,
		ref,
		sha,
		repo,
		sha,
		message,
		author,
	))
}

func signPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func deploymentCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM deployments`).Scan(&count); err != nil {
		t.Fatalf("count deployments: %v", err)
	}
	return count
}

func numberAsInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}
