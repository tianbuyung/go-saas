CREATE VIEW active_users AS
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM users
WHERE deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (email, password, salt, provider, provider_id)
VALUES (LOWER($1), $2, $3, $4, $5)
RETURNING *;

-- name: ListUsers :many
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM active_users
ORDER BY LOWER(email);

-- name: GetUserByEmail :one
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM active_users
WHERE LOWER(email) = LOWER($1)
LIMIT 1;

-- name: GetUserByProvider :one
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM active_users
WHERE provider = $1
AND provider_id = $2
LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM active_users
WHERE id = $1
LIMIT 1;

-- name: UpdateUserEmail :one
UPDATE users
SET email = LOWER($2)
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password = $2,
    salt = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteUser :one
UPDATE users
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id;
