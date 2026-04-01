-- name: CreateUser :one
INSERT INTO users (id, github_id, username, created_at)
VALUES (?,?,?,CURRENT_TIMESTAMP)
RETURNING id, github_id, username, created_at;

-- name: GetUserByID :one
SELECT id, github_id, username, created_at
FROM users
WHERE id = ?;

-- name: GetUserByGitHubID :one
SELECT id, github_id, username, created_at
FROM users
WHERE github_id = ?;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;

-- name: ListUsers :many
SELECT id, github_id, username, created_at
FROM users
ORDER BY id;
