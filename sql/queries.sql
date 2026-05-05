-- ============================================================
-- users
-- ============================================================

-- name: CreateUser :one
INSERT INTO users (name, email, image)
VALUES (sqlc.arg(name), LOWER(sqlc.arg(email)), sqlc.arg(image))
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM active_users WHERE id = sqlc.arg(id) LIMIT 1;

-- name: GetUserByPublicID :one
SELECT * FROM active_users WHERE public_id = sqlc.arg(public_id) LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM active_users WHERE LOWER(email) = LOWER(sqlc.arg(email)) LIMIT 1;

-- name: UpdateUserEmail :one
UPDATE users
SET email = LOWER(sqlc.arg(email)), updated_at = NOW()
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserProfile :one
UPDATE users
SET name = sqlc.arg(name), image = sqlc.arg(image), updated_at = NOW()
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING *;

-- name: MarkEmailVerified :one
UPDATE users
SET email_verified = TRUE, updated_at = NOW()
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteUser :one
UPDATE users
SET deleted_at = NOW()
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id;

-- ============================================================
-- accounts
-- ============================================================

-- name: CreateAccount :one
INSERT INTO accounts (user_id, account_id, provider_id, password, salt)
VALUES (
  sqlc.arg(user_id),
  sqlc.arg(account_id),
  sqlc.arg(provider_id),
  sqlc.arg(password),
  sqlc.arg(salt)
)
RETURNING *;

-- name: GetAccountByProvider :one
SELECT * FROM accounts
WHERE provider_id = sqlc.arg(provider_id) AND account_id = sqlc.arg(account_id)
LIMIT 1;

-- name: GetCredentialAccountByUserID :one
SELECT * FROM accounts
WHERE user_id = sqlc.arg(user_id) AND provider_id = 'credential'
LIMIT 1;

-- name: UpdateAccountPassword :one
UPDATE accounts
SET password = sqlc.arg(password), salt = sqlc.arg(salt), updated_at = NOW()
WHERE user_id = sqlc.arg(user_id) AND provider_id = 'credential'
RETURNING *;

-- ============================================================
-- sessions
-- ============================================================

-- name: CreateSession :one
INSERT INTO sessions (user_id, token, expires_at, ip_address, user_agent)
VALUES (
  sqlc.arg(user_id),
  sqlc.arg(token),
  sqlc.arg(expires_at),
  sqlc.arg(ip_address),
  sqlc.arg(user_agent)
)
RETURNING *;

-- name: GetSessionByToken :one
SELECT * FROM sessions
WHERE token = sqlc.arg(token) AND expires_at > NOW()
LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = sqlc.arg(token);

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = sqlc.arg(user_id);

-- ============================================================
-- verifications
-- ============================================================

-- name: UpsertVerification :one
INSERT INTO verifications (identifier, value, type, expires_at)
VALUES (sqlc.arg(identifier), sqlc.arg(value), sqlc.arg(type), sqlc.arg(expires_at))
ON CONFLICT (identifier, type) DO UPDATE
  SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at, updated_at = NOW()
RETURNING *;

-- name: GetVerification :one
SELECT * FROM verifications
WHERE identifier = sqlc.arg(identifier)
  AND type = sqlc.arg(type)
  AND expires_at > NOW()
LIMIT 1;

-- name: DeleteVerification :exec
DELETE FROM verifications
WHERE identifier = sqlc.arg(identifier) AND type = sqlc.arg(type);
