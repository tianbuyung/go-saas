-- name: CreateUser :one
INSERT INTO users (email, password, salt, provider, provider_id)
VALUES (
  LOWER(sqlc.arg(email)),
  sqlc.arg(password),
  sqlc.arg(salt),
  sqlc.arg(provider),
  sqlc.arg(provider_id)
)
RETURNING *;

-- name: ListUsers :many
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM active_users
ORDER BY LOWER(email);

-- name: GetUserByEmail :one
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM active_users
WHERE LOWER(email) = LOWER(sqlc.arg(email))
LIMIT 1;

-- name: GetUserByProvider :one
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM active_users
WHERE provider = sqlc.arg(provider)
AND provider_id = sqlc.arg(provider_id)
LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM active_users
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: UpdateUserEmail :one
UPDATE users
SET email = LOWER(sqlc.arg(email))
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password = sqlc.arg(password),
    salt = sqlc.arg(salt)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteUser :one
UPDATE users
SET deleted_at = NOW()
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id;
