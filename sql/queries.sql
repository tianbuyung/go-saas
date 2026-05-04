-- name: CreateUser :one
INSERT INTO users (email, password, provider, provider_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY email;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByProvider :one
SELECT * FROM users
WHERE provider = $1 AND provider_id = $2;

-- name: UpdateUser :one
UPDATE users
SET 
  email = COALESCE($2, email),
  password = COALESCE($3, password)
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
