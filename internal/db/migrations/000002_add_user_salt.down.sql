BEGIN;

-- 1. Drop new indexes (safe)
DROP INDEX IF EXISTS unique_users_email_ci;
DROP INDEX IF EXISTS unique_provider_identity;
DROP INDEX IF EXISTS idx_users_active;

-- 2. Restore old indexes (no soft-delete logic)
CREATE UNIQUE INDEX unique_users_email_ci
ON users (LOWER(email));

CREATE UNIQUE INDEX unique_provider_identity
ON users (provider, provider_id)
WHERE provider IS NOT NULL AND provider_id IS NOT NULL;

CREATE INDEX idx_users_email
ON users (LOWER(email));

-- 3. Restore old constraint (no salt logic)
ALTER TABLE users
DROP CONSTRAINT IF EXISTS check_auth_method;

ALTER TABLE users
ADD CONSTRAINT check_auth_method CHECK (
  (
    password IS NOT NULL
    AND provider IS NULL
    AND provider_id IS NULL
  )
  OR
  (
    password IS NULL
    AND provider IS NOT NULL
    AND provider_id IS NOT NULL
  )
);

-- 4. Drop new columns
ALTER TABLE users
DROP COLUMN IF EXISTS salt;

ALTER TABLE users
DROP COLUMN IF EXISTS deleted_at;

COMMIT;
