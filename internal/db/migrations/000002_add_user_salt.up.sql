BEGIN;

ALTER TABLE users
ADD COLUMN IF NOT EXISTS salt TEXT;

UPDATE users
SET password = NULL
WHERE password IS NOT NULL AND salt IS NULL;

ALTER TABLE users
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

ALTER TABLE users
DROP CONSTRAINT IF EXISTS check_auth_method;

ALTER TABLE users
ADD CONSTRAINT check_auth_method CHECK (
  (
    password IS NOT NULL AND salt IS NOT NULL
    AND provider IS NULL AND provider_id IS NULL
  )
  OR
  (
    password IS NULL AND salt IS NULL
    AND provider IS NOT NULL AND provider_id IS NOT NULL
  )
);

DROP INDEX IF EXISTS unique_users_email_ci;
CREATE UNIQUE INDEX unique_users_email_ci
ON users (LOWER(email))
WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_users_email;

DROP INDEX IF EXISTS unique_provider_identity;
CREATE UNIQUE INDEX unique_provider_identity
ON users (provider, provider_id)
WHERE provider IS NOT NULL
  AND provider_id IS NOT NULL
  AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_active
ON users (id)
WHERE deleted_at IS NULL;

COMMIT;
