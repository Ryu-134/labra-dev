package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"labra-backend/internal/api/auth"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
)

func TestSprint2Phase1Phase2Flow(t *testing.T) {
	secret := "sprint2-secret"
	issuer := "https://cognito-idp.us-west-2.amazonaws.com/us-west-2_demo"
	audience := "labra-control-plane"

	db := setupSprint2TestDB(t)
	previousStore := appStore
	previousAuthValidator := authValidator
	previousTokenIssuer := tokenIssuer
	previousAssumeRoleVerifier := assumeRoleVerifier
	t.Cleanup(func() {
		appStore = previousStore
		authValidator = previousAuthValidator
		tokenIssuer = previousTokenIssuer
		assumeRoleVerifier = previousAssumeRoleVerifier
		_ = db.Close()
	})

	InitAppStore(db)
	validator := auth.HMACValidator{Issuer: issuer, Audience: audience, Secret: []byte(secret)}
	InitAuthRuntime(validator, auth.TokenIssuer{
		Issuer:   issuer,
		Audience: audience,
		Secret:   []byte(secret),
		TTL:      2 * time.Hour,
	})

	externalToken := mintTestJWT(t, secret, issuer, audience, jwt.MapClaims{
		"sub":            "cognito-user-001",
		"email":          "casey@example.com",
		"cognito:groups": []string{"owner"},
	})

	t.Run("create auth session and provision user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/session", nil)
		req.Header.Set("Authorization", "Bearer "+externalToken)
		rr := httptest.NewRecorder()
		PostAuthSessionHandler(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
		}

		var body struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
			User struct {
				ID    int64 `json:"id"`
				IsNew bool  `json:"is_new"`
			} `json:"user"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal auth session response: %v", err)
		}
		if body.Session.Token == "" {
			t.Fatalf("expected session token")
		}
		if !body.User.IsNew {
			t.Fatalf("expected first login to provision new user")
		}

		sessionToken := body.Session.Token

		profileReq := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
		profileReq.Header.Set("Authorization", "Bearer "+sessionToken)
		profileRR := httptest.NewRecorder()
		auth.RequireAuth(validator)(http.HandlerFunc(GetProfileHandler)).ServeHTTP(profileRR, profileReq)
		if profileRR.Code != http.StatusOK {
			t.Fatalf("expected profile 200, got %d body=%s", profileRR.Code, profileRR.Body.String())
		}

		awsPayload := []byte(`{"role_arn":"arn:aws:iam::123456789012:role/labra-dev-access","external_id":"external-id-123","region":"us-west-2"}`)
		awsReq := httptest.NewRequest(http.MethodPost, "/v1/aws-connections", bytes.NewReader(awsPayload))
		awsReq.Header.Set("Content-Type", "application/json")
		awsReq.Header.Set("Authorization", "Bearer "+sessionToken)
		awsRR := httptest.NewRecorder()
		auth.RequireAuth(validator)(http.HandlerFunc(UpsertAWSConnectionHandler)).ServeHTTP(awsRR, awsReq)
		if awsRR.Code != http.StatusCreated {
			t.Fatalf("expected aws connection 201, got %d body=%s", awsRR.Code, awsRR.Body.String())
		}

		systemReq := httptest.NewRequest(http.MethodGet, "/v1/system/services", nil)
		systemReq.Header.Set("Authorization", "Bearer "+sessionToken)
		systemRR := httptest.NewRecorder()
		auth.RequireAuth(validator)(http.HandlerFunc(GetSystemServicesHandler)).ServeHTTP(systemRR, systemReq)
		if systemRR.Code != http.StatusOK {
			t.Fatalf("expected system services 200, got %d body=%s", systemRR.Code, systemRR.Body.String())
		}

		logoutReq := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
		logoutReq.Header.Set("Authorization", "Bearer "+sessionToken)
		logoutRR := httptest.NewRecorder()
		auth.RequireAuth(validator)(http.HandlerFunc(PostLogoutHandler)).ServeHTTP(logoutRR, logoutReq)
		if logoutRR.Code != http.StatusNoContent {
			t.Fatalf("expected logout 204, got %d body=%s", logoutRR.Code, logoutRR.Body.String())
		}
	})

	t.Run("second session reuses provisioned user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/session", nil)
		req.Header.Set("Authorization", "Bearer "+externalToken)
		rr := httptest.NewRecorder()
		PostAuthSessionHandler(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
		}
		var body struct {
			User struct {
				IsNew bool `json:"is_new"`
			} `json:"user"`
		}
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		if body.User.IsNew {
			t.Fatalf("expected existing provisioned user on second login")
		}
	})
}

func setupSprint2TestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS platform_users (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  email TEXT,
	  status TEXT NOT NULL DEFAULT 'active',
	  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_platform_users_email
	  ON platform_users(email)
	  WHERE email IS NOT NULL;

	CREATE TABLE IF NOT EXISTS auth_identities (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  user_id INTEGER NOT NULL,
	  provider TEXT NOT NULL,
	  subject TEXT NOT NULL,
	  email TEXT,
	  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  UNIQUE(provider, subject)
	);

	CREATE TABLE IF NOT EXISTS auth_sessions (
	  session_id TEXT PRIMARY KEY,
	  user_id INTEGER NOT NULL,
	  expires_at INTEGER NOT NULL,
	  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
	  revoked_at INTEGER
	);

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

func mintTestJWT(t *testing.T, secret, issuer, audience string, extra jwt.MapClaims) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"exp": time.Now().Add(2 * time.Hour).Unix(),
	}
	for k, v := range extra {
		claims[k] = v
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return raw
}

func TestPostAuthSessionRejectsMissingToken(t *testing.T) {
	db := setupSprint2TestDB(t)
	defer db.Close()
	InitAppStore(db)
	InitAuthRuntime(auth.HMACValidator{Secret: []byte("x")}, auth.TokenIssuer{Secret: []byte("x")})

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/session", nil)
	rr := httptest.NewRecorder()
	PostAuthSessionHandler(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAssumeRoleValidationRejectsAccountMismatch(t *testing.T) {
	db := setupSprint2TestDB(t)
	defer db.Close()
	InitAppStore(db)

	ctx := auth.WithPrincipal(context.Background(), auth.Principal{UserID: 11, Sub: "sub-11", Roles: []string{"owner"}})
	payload := []byte(`{"role_arn":"arn:aws:iam::123456789012:role/labra-dev-access","external_id":"external-id-123","region":"us-west-2","account_id":"999999999999"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/aws-connections", bytes.NewReader(payload)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	UpsertAWSConnectionHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
