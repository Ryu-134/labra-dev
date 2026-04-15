package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSprint1ReadinessAndAWSConnectionsFlow(t *testing.T) {
	db := setupSprint1TestDB(t)
	previousStore := appStore
	previousProbe := readinessProbe
	t.Cleanup(func() {
		appStore = previousStore
		readinessProbe = previousProbe
		_ = db.Close()
	})

	InitAppStore(db)
	InitReadiness(func(ctx context.Context) error { return db.PingContext(ctx) })

	t.Run("ready endpoint returns success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()
		HandleReadiness(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("upsert and list aws connections", func(t *testing.T) {
		payload := []byte(`{"role_arn":"arn:aws:iam::123456789012:role/labra-dev-access","external_id":"ext-id-12345","region":"us-west-2"}`)
		req := httptest.NewRequest(http.MethodPost, "/v1/aws-connections", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "7")
		rr := httptest.NewRecorder()

		UpsertAWSConnectionHandler(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d, body=%s", rr.Code, rr.Body.String())
		}

		listReq := httptest.NewRequest(http.MethodGet, "/v1/aws-connections", nil)
		listReq.Header.Set("X-User-ID", "7")
		listRR := httptest.NewRecorder()
		ListAWSConnectionsHandler(listRR, listReq)
		if listRR.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body=%s", listRR.Code, listRR.Body.String())
		}

		var body struct {
			Connections []struct {
				RoleARN   string `json:"role_arn"`
				AccountID string `json:"account_id"`
				Region    string `json:"region"`
			} `json:"aws_connections"`
		}
		if err := json.Unmarshal(listRR.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal list response: %v", err)
		}
		if len(body.Connections) != 1 {
			t.Fatalf("expected 1 connection, got %d", len(body.Connections))
		}
		if body.Connections[0].AccountID != "123456789012" {
			t.Fatalf("expected derived account ID, got %q", body.Connections[0].AccountID)
		}
	})

	t.Run("invalid role arn returns bad request", func(t *testing.T) {
		payload := []byte(`{"role_arn":"bad-arn","external_id":"ext-id-12345","region":"us-west-2"}`)
		req := httptest.NewRequest(http.MethodPost, "/v1/aws-connections", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "7")
		rr := httptest.NewRecorder()

		UpsertAWSConnectionHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d, body=%s", rr.Code, rr.Body.String())
		}
	})
}

func setupSprint1TestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS aws_connections (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  user_id INTEGER NOT NULL,
	  role_arn TEXT NOT NULL,
	  external_id TEXT NOT NULL,
	  region TEXT NOT NULL,
	  account_id TEXT NOT NULL,
	  status TEXT NOT NULL,
	  last_validated_at INTEGER,
	  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_aws_connections_user_role_region
	  ON aws_connections(user_id, role_arn, region);

	CREATE TABLE IF NOT EXISTS audit_events (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  actor_user_id INTEGER NOT NULL,
	  event_type TEXT NOT NULL,
	  target_type TEXT NOT NULL,
	  target_id TEXT,
	  status TEXT NOT NULL,
	  message TEXT,
	  metadata_json TEXT,
	  created_at INTEGER NOT NULL DEFAULT (unixepoch())
	);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	return db
}

func TestReadinessFailureReturnsServiceUnavailable(t *testing.T) {
	previousProbe := readinessProbe
	t.Cleanup(func() {
		readinessProbe = previousProbe
	})

	InitReadiness(func(context.Context) error { return fmt.Errorf("db unavailable") })

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rr := httptest.NewRecorder()
	HandleReadiness(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d, body=%s", rr.Code, rr.Body.String())
	}
}
