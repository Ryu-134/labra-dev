-- name: CreateApp :one
INSERT INTO apps (
  user_id,
  name,
  repo_full_name,
  branch,
  build_type,
  output_dir,
  root_dir,
  site_url,
  auto_deploy_enabled,
  created_at,
  updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch(), unixepoch())
RETURNING id, user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at;

-- name: ListAppsByUser :many
SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at
FROM apps
WHERE user_id = ?
ORDER BY updated_at DESC;

-- name: GetAppByIDForUser :one
SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at
FROM apps
WHERE id = ? AND user_id = ?;

-- name: UpdateApp :one
UPDATE apps
SET name = ?, branch = ?, build_type = ?, output_dir = ?, root_dir = ?, site_url = ?, auto_deploy_enabled = ?, updated_at = unixepoch()
WHERE id = ? AND user_id = ?
RETURNING id, user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at;

-- name: ListAppsByRepo :many
SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at
FROM apps
WHERE lower(repo_full_name) = lower(?) AND auto_deploy_enabled = 1;
