CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,

  email TEXT NOT NULL,
  password TEXT,

  provider TEXT,
  provider_id TEXT,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),

  CONSTRAINT check_auth_method CHECK (
    (password IS NOT NULL AND provider IS NULL AND provider_id IS NULL)
    OR
    (password IS NULL AND provider IS NOT NULL AND provider_id IS NOT NULL)
  ),

  CONSTRAINT email_not_empty CHECK (TRIM(email) <> '')
);

-- case-insensitive email uniqueness
CREATE UNIQUE INDEX unique_users_email_ci
ON users (LOWER(email));

-- enforce social login uniqueness
CREATE UNIQUE INDEX unique_provider_identity
ON users (provider, provider_id)
WHERE provider IS NOT NULL AND provider_id IS NOT NULL;

-- performance index
CREATE INDEX idx_users_email
ON users (LOWER(email));
