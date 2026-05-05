CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,

  email TEXT NOT NULL,
  password TEXT,
  salt TEXT,

  provider TEXT,
  provider_id TEXT,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMP,

  CONSTRAINT email_not_empty CHECK (TRIM(email) <> ''),

  CONSTRAINT check_auth_method CHECK (
    (
      password IS NOT NULL AND salt IS NOT NULL
      AND provider IS NULL AND provider_id IS NULL
    )
    OR
    (
      password IS NULL AND salt IS NULL
      AND provider IS NOT NULL AND provider_id IS NOT NULL
    )
  )
);

-- Active users view (exclude soft-deleted users)
CREATE VIEW active_users AS
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM users
WHERE deleted_at IS NULL;

-- case-insensitive email uniqueness (only active users)
CREATE UNIQUE INDEX unique_users_email_ci
ON users (LOWER(email))
WHERE deleted_at IS NULL;

-- enforce social login uniqueness (only active users)
CREATE UNIQUE INDEX unique_provider_identity
ON users (provider, provider_id)
WHERE provider IS NOT NULL
  AND provider_id IS NOT NULL
  AND deleted_at IS NULL;

-- active users index
CREATE INDEX idx_users_active
ON users (id)
WHERE deleted_at IS NULL;
