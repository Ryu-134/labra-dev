package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateApp(ctx context.Context, in CreateAppInput) (App, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO apps (
			user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch(), unixepoch())
		RETURNING id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, created_at, updated_at
	`, in.UserID, in.Name, in.RepoFullName, in.Branch, in.BuildType, in.OutputDir, nullIfEmpty(in.RootDir), nullIfEmpty(in.SiteURL), boolToInt(in.AutoDeployEnabled))

	var app App
	var autoDeployInt int
	if err := row.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CreatedAt, &app.UpdatedAt); err != nil {
		return App{}, err
	}
	app.AutoDeployEnabled = autoDeployInt == 1
	return app, nil
}

func (s *Store) ListAppsByUser(ctx context.Context, userID int64) ([]App, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, created_at, updated_at
		FROM apps
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]App, 0)
	for rows.Next() {
		var app App
		var autoDeployInt int
		if err := rows.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		app.AutoDeployEnabled = autoDeployInt == 1
		out = append(out, app)
	}
	return out, rows.Err()
}

func (s *Store) GetAppByIDForUser(ctx context.Context, appID, userID int64) (App, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, created_at, updated_at
		FROM apps
		WHERE id = ? AND user_id = ?
	`, appID, userID)

	var app App
	var autoDeployInt int
	if err := row.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CreatedAt, &app.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return App{}, ErrNotFound
		}
		return App{}, err
	}
	app.AutoDeployEnabled = autoDeployInt == 1
	return app, nil
}

func (s *Store) UpdateAppForUser(ctx context.Context, appID, userID int64, in UpdateAppInput) (App, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE apps
		SET name = ?, branch = ?, build_type = ?, output_dir = ?, root_dir = ?, site_url = ?, auto_deploy_enabled = ?, updated_at = unixepoch()
		WHERE id = ? AND user_id = ?
		RETURNING id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, created_at, updated_at
	`, in.Name, in.Branch, in.BuildType, in.OutputDir, nullIfEmpty(in.RootDir), nullIfEmpty(in.SiteURL), boolToInt(in.AutoDeployEnabled), appID, userID)

	var app App
	var autoDeployInt int
	if err := row.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CreatedAt, &app.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return App{}, ErrNotFound
		}
		return App{}, err
	}
	app.AutoDeployEnabled = autoDeployInt == 1
	return app, nil
}

func (s *Store) ListAutoDeployAppsByRepo(ctx context.Context, repoFullName string) ([]App, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, created_at, updated_at
		FROM apps
		WHERE lower(repo_full_name) = lower(?) AND auto_deploy_enabled = 1
		ORDER BY id ASC
	`, repoFullName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]App, 0)
	for rows.Next() {
		var app App
		var autoDeployInt int
		if err := rows.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		app.AutoDeployEnabled = autoDeployInt == 1
		out = append(out, app)
	}
	return out, rows.Err()
}

func (s *Store) CreateDeployment(ctx context.Context, in CreateDeploymentInput) (Deployment, error) {
	var startedAt any
	var finishedAt any
	if in.StartedAt > 0 {
		startedAt = in.StartedAt
	}
	if in.FinishedAt > 0 {
		finishedAt = in.FinishedAt
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO deployments (
			app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id,
			created_at, updated_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch(), unixepoch(), ?, ?)
		RETURNING id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(correlation_id, ''), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, in.AppID, in.UserID, in.Status, in.TriggerType, nullIfEmpty(in.CommitSHA), nullIfEmpty(in.CommitMessage), nullIfEmpty(in.CommitAuthor),
		nullIfEmpty(in.Branch), nullIfEmpty(in.SiteURL), nullIfEmpty(in.FailureReason), nullIfEmpty(in.CorrelationID), startedAt, finishedAt)

	var dep Deployment
	if err := row.Scan(&dep.ID, &dep.AppID, &dep.UserID, &dep.Status, &dep.TriggerType, &dep.CommitSHA, &dep.CommitMessage, &dep.CommitAuthor,
		&dep.Branch, &dep.SiteURL, &dep.FailureReason, &dep.CorrelationID, &dep.CreatedAt, &dep.UpdatedAt, &dep.StartedAt, &dep.FinishedAt); err != nil {
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) GetDeploymentByIDForUser(ctx context.Context, deploymentID, userID int64) (Deployment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(correlation_id, ''), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE id = ? AND user_id = ?
	`, deploymentID, userID)

	var dep Deployment
	if err := row.Scan(&dep.ID, &dep.AppID, &dep.UserID, &dep.Status, &dep.TriggerType, &dep.CommitSHA, &dep.CommitMessage, &dep.CommitAuthor,
		&dep.Branch, &dep.SiteURL, &dep.FailureReason, &dep.CorrelationID, &dep.CreatedAt, &dep.UpdatedAt, &dep.StartedAt, &dep.FinishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Deployment{}, ErrNotFound
		}
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) ListDeploymentsByAppForUser(ctx context.Context, appID, userID int64) ([]Deployment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(correlation_id, ''), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE app_id = ? AND user_id = ?
		ORDER BY created_at DESC
	`, appID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Deployment, 0)
	for rows.Next() {
		var dep Deployment
		if err := rows.Scan(&dep.ID, &dep.AppID, &dep.UserID, &dep.Status, &dep.TriggerType, &dep.CommitSHA, &dep.CommitMessage, &dep.CommitAuthor,
			&dep.Branch, &dep.SiteURL, &dep.FailureReason, &dep.CorrelationID, &dep.CreatedAt, &dep.UpdatedAt, &dep.StartedAt, &dep.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, dep)
	}
	return out, rows.Err()
}

func (s *Store) UpdateDeploymentStatus(ctx context.Context, deploymentID int64, status, reason, siteURL string, startedAt, finishedAt int64) (Deployment, error) {
	var started any
	var finished any
	if startedAt > 0 {
		started = startedAt
	}
	if finishedAt > 0 {
		finished = finishedAt
	}

	row := s.db.QueryRowContext(ctx, `
		UPDATE deployments
		SET status = ?, failure_reason = ?, site_url = ?, updated_at = unixepoch(), started_at = COALESCE(?, started_at), finished_at = COALESCE(?, finished_at)
		WHERE id = ?
		RETURNING id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(correlation_id, ''), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, status, nullIfEmpty(reason), nullIfEmpty(siteURL), started, finished, deploymentID)

	var dep Deployment
	if err := row.Scan(&dep.ID, &dep.AppID, &dep.UserID, &dep.Status, &dep.TriggerType, &dep.CommitSHA, &dep.CommitMessage, &dep.CommitAuthor,
		&dep.Branch, &dep.SiteURL, &dep.FailureReason, &dep.CorrelationID, &dep.CreatedAt, &dep.UpdatedAt, &dep.StartedAt, &dep.FinishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Deployment{}, ErrNotFound
		}
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) CreateDeploymentLog(ctx context.Context, deploymentID int64, level, message string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO deployment_logs (deployment_id, log_level, message, created_at)
		VALUES (?, ?, ?, unixepoch())
	`, deploymentID, level, message)
	return err
}

func (s *Store) ListDeploymentLogs(ctx context.Context, deploymentID int64) ([]DeploymentLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, deployment_id, log_level, message, created_at
		FROM deployment_logs
		WHERE deployment_id = ?
		ORDER BY created_at ASC, id ASC
	`, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeploymentLog, 0)
	for rows.Next() {
		var l DeploymentLog
		if err := rows.Scan(&l.ID, &l.DeploymentID, &l.LogLevel, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) ClaimWebhookDelivery(ctx context.Context, appID int64, deliveryID, eventType, commitSHA string) (bool, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (app_id, delivery_id, event_type, commit_sha, received_at)
		VALUES (?, ?, ?, ?, unixepoch())
	`, appID, deliveryID, eventType, nullIfEmpty(commitSHA))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") {
			return false, nil
		}
		return false, fmt.Errorf("insert webhook delivery: %w", err)
	}
	return true, nil
}

func (s *Store) UpsertAWSConnection(ctx context.Context, in UpsertAWSConnectionInput) (AWSConnection, error) {
	var lastValidatedAt any
	if in.LastValidatedAt > 0 {
		lastValidatedAt = in.LastValidatedAt
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO aws_connections (
			user_id, role_arn, external_id, region, account_id, status, last_validated_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, unixepoch(), unixepoch())
		ON CONFLICT(user_id, role_arn, region) DO UPDATE SET
			external_id = excluded.external_id,
			account_id = excluded.account_id,
			status = excluded.status,
			last_validated_at = excluded.last_validated_at,
			updated_at = unixepoch()
		RETURNING id, user_id, role_arn, external_id, region, account_id, status, COALESCE(last_validated_at, 0), created_at, updated_at
	`, in.UserID, in.RoleARN, in.ExternalID, in.Region, in.AccountID, in.Status, lastValidatedAt)

	var out AWSConnection
	if err := row.Scan(&out.ID, &out.UserID, &out.RoleARN, &out.ExternalID, &out.Region, &out.AccountID, &out.Status, &out.LastValidatedAt, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return AWSConnection{}, err
	}
	return out, nil
}

func (s *Store) ListAWSConnectionsByUser(ctx context.Context, userID int64) ([]AWSConnection, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, role_arn, external_id, region, account_id, status, COALESCE(last_validated_at, 0), created_at, updated_at
		FROM aws_connections
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AWSConnection, 0)
	for rows.Next() {
		var c AWSConnection
		if err := rows.Scan(&c.ID, &c.UserID, &c.RoleARN, &c.ExternalID, &c.Region, &c.AccountID, &c.Status, &c.LastValidatedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateAuditEvent(ctx context.Context, in AuditEventInput) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_events (
			actor_user_id, event_type, target_type, target_id, status, message, metadata_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, unixepoch())
	`, in.ActorUserID, in.EventType, in.TargetType, nullIfEmpty(in.TargetID), in.Status, nullIfEmpty(in.Message), nullIfEmpty(in.Metadata))
	return err
}

func (s *Store) CreatePlatformUser(ctx context.Context, in CreatePlatformUserInput) (PlatformUser, error) {
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "active"
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO platform_users (email, status, created_at, updated_at)
		VALUES (?, ?, unixepoch(), unixepoch())
		RETURNING id, COALESCE(email, ''), status, created_at, updated_at
	`, nullIfEmpty(in.Email), status)

	var out PlatformUser
	if err := row.Scan(&out.ID, &out.Email, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return PlatformUser{}, err
	}
	return out, nil
}

func (s *Store) GetPlatformUserByID(ctx context.Context, userID int64) (PlatformUser, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(email, ''), status, created_at, updated_at
		FROM platform_users
		WHERE id = ?
	`, userID)

	var out PlatformUser
	if err := row.Scan(&out.ID, &out.Email, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PlatformUser{}, ErrNotFound
		}
		return PlatformUser{}, err
	}
	return out, nil
}

func (s *Store) GetPlatformUserByIdentity(ctx context.Context, provider, subject string) (PlatformUser, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, COALESCE(u.email, ''), u.status, u.created_at, u.updated_at
		FROM platform_users u
		INNER JOIN auth_identities ai ON ai.user_id = u.id
		WHERE ai.provider = ? AND ai.subject = ?
	`, strings.TrimSpace(provider), strings.TrimSpace(subject))

	var out PlatformUser
	if err := row.Scan(&out.ID, &out.Email, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PlatformUser{}, ErrNotFound
		}
		return PlatformUser{}, err
	}
	return out, nil
}

func (s *Store) UpsertAuthIdentity(ctx context.Context, in UpsertAuthIdentityInput) (AuthIdentity, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO auth_identities (user_id, provider, subject, email, created_at, updated_at)
		VALUES (?, ?, ?, ?, unixepoch(), unixepoch())
		ON CONFLICT(provider, subject) DO UPDATE SET
			user_id = excluded.user_id,
			email = excluded.email,
			updated_at = unixepoch()
		RETURNING id, user_id, provider, subject, COALESCE(email, ''), created_at, updated_at
	`, in.UserID, strings.TrimSpace(in.Provider), strings.TrimSpace(in.Subject), nullIfEmpty(in.Email))

	var out AuthIdentity
	if err := row.Scan(&out.ID, &out.UserID, &out.Provider, &out.Subject, &out.Email, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return AuthIdentity{}, err
	}
	return out, nil
}

func (s *Store) CreateAuthSession(ctx context.Context, in CreateAuthSessionInput) (AuthSession, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO auth_sessions (session_id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, unixepoch())
		RETURNING session_id, user_id, expires_at, created_at, COALESCE(revoked_at, 0)
	`, in.SessionID, in.UserID, in.ExpiresAt)

	var out AuthSession
	if err := row.Scan(&out.SessionID, &out.UserID, &out.ExpiresAt, &out.CreatedAt, &out.RevokedAt); err != nil {
		return AuthSession{}, err
	}
	return out, nil
}

func (s *Store) GetAuthSessionByID(ctx context.Context, sessionID string) (AuthSession, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT session_id, user_id, expires_at, created_at, COALESCE(revoked_at, 0)
		FROM auth_sessions
		WHERE session_id = ?
	`, strings.TrimSpace(sessionID))

	var out AuthSession
	if err := row.Scan(&out.SessionID, &out.UserID, &out.ExpiresAt, &out.CreatedAt, &out.RevokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthSession{}, ErrNotFound
		}
		return AuthSession{}, err
	}
	return out, nil
}

func (s *Store) RevokeAuthSession(ctx context.Context, sessionID string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = unixepoch()
		WHERE session_id = ? AND revoked_at IS NULL
	`, strings.TrimSpace(sessionID))
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func UnixNow() int64 {
	return time.Now().Unix()
}
