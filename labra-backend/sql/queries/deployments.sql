-- name: CreateDeployment :one
INSERT INTO deployments (
  app_id,
  user_id,
  status,
  trigger_type,
  commit_sha,
  commit_message,
  commit_author,
  branch,
  site_url,
  failure_reason,
  correlation_id,
  created_at,
  updated_at,
  started_at,
  finished_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch(), unixepoch(), ?, ?)
RETURNING id, app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id, created_at, updated_at, started_at, finished_at;

-- name: GetDeploymentByIDForUser :one
SELECT id, app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id, created_at, updated_at, started_at, finished_at
FROM deployments
WHERE id = ? AND user_id = ?;

-- name: ListDeploymentsByAppForUser :many
SELECT id, app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id, created_at, updated_at, started_at, finished_at
FROM deployments
WHERE app_id = ? AND user_id = ?
ORDER BY created_at DESC;

-- name: UpdateDeploymentStatus :one
UPDATE deployments
SET status = ?, failure_reason = ?, updated_at = unixepoch(), started_at = ?, finished_at = ?, site_url = ?
WHERE id = ?
RETURNING id, app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id, created_at, updated_at, started_at, finished_at;

-- name: CreateDeploymentLog :one
INSERT INTO deployment_logs (deployment_id, log_level, message, created_at)
VALUES (?, ?, ?, unixepoch())
RETURNING id, deployment_id, log_level, message, created_at;

-- name: ListDeploymentLogs :many
SELECT id, deployment_id, log_level, message, created_at
FROM deployment_logs
WHERE deployment_id = ?
ORDER BY created_at ASC;

-- name: InsertWebhookDelivery :exec
INSERT INTO webhook_deliveries (app_id, delivery_id, event_type, commit_sha, received_at)
VALUES (?, ?, ?, ?, unixepoch());
