-- name: CreateSession :one
INSERT INTO sessions (uuid, user_id, expires_at)
VALUES (?, ?, ?)
RETURNING uuid, user_id, expires_at;

-- name: GetSessionByID :one
SELECT uuid, user_id, expires_at
FROM sessions
WHERE uuid = ?;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE uuid = ?;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions
WHERE user_id = ?;

-- name: ListSessions :many
SELECT uuid, user_id, expires_at
FROM sessions
ORDER BY expires_at;
