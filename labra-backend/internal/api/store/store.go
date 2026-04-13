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

func (s *Store) GetLatestDeploymentByAppForUser(ctx context.Context, appID, userID int64) (Deployment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(correlation_id, ''), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE app_id = ? AND user_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, appID, userID)

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

func (s *Store) GetLastSuccessfulDeploymentByAppForUser(ctx context.Context, appID, userID int64) (Deployment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(correlation_id, ''), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE app_id = ? AND user_id = ? AND status = 'succeeded'
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, appID, userID)

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

func (s *Store) ListAppEnvVarsByAppForUser(ctx context.Context, appID, userID int64) ([]AppEnvVar, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.app_id, e.key, e.value, e.is_secret, e.created_at, e.updated_at
		FROM app_env_vars e
		JOIN apps a ON a.id = e.app_id
		WHERE e.app_id = ? AND a.user_id = ?
		ORDER BY e.key ASC
	`, appID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AppEnvVar, 0)
	for rows.Next() {
		var envVar AppEnvVar
		var isSecretInt int
		if err := rows.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
			return nil, err
		}
		envVar.IsSecret = isSecretInt == 1
		out = append(out, envVar)
	}
	return out, rows.Err()
}

func (s *Store) ListAppEnvVarsForApp(ctx context.Context, appID int64) ([]AppEnvVar, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, app_id, key, value, is_secret, created_at, updated_at
		FROM app_env_vars
		WHERE app_id = ?
		ORDER BY key ASC
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AppEnvVar, 0)
	for rows.Next() {
		var envVar AppEnvVar
		var isSecretInt int
		if err := rows.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
			return nil, err
		}
		envVar.IsSecret = isSecretInt == 1
		out = append(out, envVar)
	}
	return out, rows.Err()
}

func (s *Store) CreateAppEnvVar(ctx context.Context, appID, userID int64, in CreateAppEnvVarInput) (AppEnvVar, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO app_env_vars (app_id, key, value, is_secret, created_at, updated_at)
		SELECT a.id, ?, ?, ?, unixepoch(), unixepoch()
		FROM apps a
		WHERE a.id = ? AND a.user_id = ?
		RETURNING id, app_id, key, value, is_secret, created_at, updated_at
	`, in.Key, in.Value, boolToInt(in.IsSecret), appID, userID)

	var envVar AppEnvVar
	var isSecretInt int
	if err := row.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppEnvVar{}, ErrNotFound
		}
		return AppEnvVar{}, err
	}
	envVar.IsSecret = isSecretInt == 1
	return envVar, nil
}

func (s *Store) GetAppEnvVarByIDForUser(ctx context.Context, appID, envVarID, userID int64) (AppEnvVar, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT e.id, e.app_id, e.key, e.value, e.is_secret, e.created_at, e.updated_at
		FROM app_env_vars e
		JOIN apps a ON a.id = e.app_id
		WHERE e.id = ? AND e.app_id = ? AND a.user_id = ?
	`, envVarID, appID, userID)

	var envVar AppEnvVar
	var isSecretInt int
	if err := row.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppEnvVar{}, ErrNotFound
		}
		return AppEnvVar{}, err
	}
	envVar.IsSecret = isSecretInt == 1
	return envVar, nil
}

func (s *Store) UpdateAppEnvVarForUser(ctx context.Context, appID, envVarID, userID int64, in UpdateAppEnvVarInput) (AppEnvVar, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE app_env_vars
		SET key = ?, value = ?, is_secret = ?, updated_at = unixepoch()
		WHERE id = ?
			AND app_id = ?
			AND EXISTS (
				SELECT 1
				FROM apps a
				WHERE a.id = ? AND a.user_id = ?
			)
		RETURNING id, app_id, key, value, is_secret, created_at, updated_at
	`, in.Key, in.Value, boolToInt(in.IsSecret), envVarID, appID, appID, userID)

	var envVar AppEnvVar
	var isSecretInt int
	if err := row.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppEnvVar{}, ErrNotFound
		}
		return AppEnvVar{}, err
	}
	envVar.IsSecret = isSecretInt == 1
	return envVar, nil
}

func (s *Store) DeleteAppEnvVarForUser(ctx context.Context, appID, envVarID, userID int64) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM app_env_vars
		WHERE id = ?
			AND app_id = ?
			AND EXISTS (
				SELECT 1
				FROM apps a
				WHERE a.id = ? AND a.user_id = ?
			)
	`, envVarID, appID, appID, userID)
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

func (s *Store) RecordAppDeploymentOutcome(ctx context.Context, appID int64, status string, finishedAt int64) error {
	succInc := int64(0)
	failInc := int64(0)
	lastSuccessAt := int64(0)
	lastFailureAt := int64(0)

	switch strings.TrimSpace(strings.ToLower(status)) {
	case "succeeded":
		succInc = 1
		lastSuccessAt = finishedAt
	case "failed":
		failInc = 1
		lastFailureAt = finishedAt
	default:
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO app_health_metrics (
			app_id, success_count, failure_count, last_success_at, last_failure_at, updated_at
		) VALUES (?, ?, ?, ?, ?, unixepoch())
		ON CONFLICT(app_id) DO UPDATE SET
			success_count = app_health_metrics.success_count + excluded.success_count,
			failure_count = app_health_metrics.failure_count + excluded.failure_count,
			last_success_at = CASE
				WHEN excluded.last_success_at > 0 THEN excluded.last_success_at
				ELSE app_health_metrics.last_success_at
			END,
			last_failure_at = CASE
				WHEN excluded.last_failure_at > 0 THEN excluded.last_failure_at
				ELSE app_health_metrics.last_failure_at
			END,
			updated_at = unixepoch()
	`, appID, succInc, failInc, lastSuccessAt, lastFailureAt)
	return err
}

func (s *Store) GetAppHealthMetricsForUser(ctx context.Context, appID, userID int64) (AppHealthMetrics, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id,
			COALESCE(m.success_count, 0),
			COALESCE(m.failure_count, 0),
			COALESCE(m.last_success_at, 0),
			COALESCE(m.last_failure_at, 0),
			COALESCE(m.updated_at, a.updated_at)
		FROM apps a
		LEFT JOIN app_health_metrics m ON m.app_id = a.id
		WHERE a.id = ? AND a.user_id = ?
	`, appID, userID)

	var metrics AppHealthMetrics
	if err := row.Scan(&metrics.AppID, &metrics.SuccessCount, &metrics.FailureCount, &metrics.LastSuccessAt, &metrics.LastFailureAt, &metrics.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppHealthMetrics{}, ErrNotFound
		}
		return AppHealthMetrics{}, err
	}
	return metrics, nil
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
