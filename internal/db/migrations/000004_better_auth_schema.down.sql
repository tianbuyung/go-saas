BEGIN;

DROP VIEW IF EXISTS active_users;

-- restore credential/oauth columns to users
ALTER TABLE users
  ADD COLUMN password    text,
  ADD COLUMN salt        text,
  ADD COLUMN provider    text,
  ADD COLUMN provider_id text;

-- restore credential data from accounts
UPDATE users u
SET password = a.password, salt = a.salt
FROM accounts a
WHERE a.user_id = u.id AND a.provider_id = 'credential';

-- restore OAuth data from accounts
UPDATE users u
SET provider = a.provider_id, provider_id = a.account_id
FROM accounts a
WHERE a.user_id = u.id AND a.provider_id <> 'credential';

-- restore check constraint
ALTER TABLE users
  ADD CONSTRAINT check_auth_method CHECK (
    (password IS NOT NULL AND salt IS NOT NULL AND provider IS NULL AND provider_id IS NULL)
    OR
    (password IS NULL AND salt IS NULL AND provider IS NOT NULL AND provider_id IS NOT NULL)
  );

-- restore provider uniqueness index
CREATE UNIQUE INDEX unique_provider_identity
ON users (provider, provider_id)
WHERE provider IS NOT NULL AND provider_id IS NOT NULL AND deleted_at IS NULL;

-- drop columns added in up
ALTER TABLE users
  DROP COLUMN IF EXISTS public_id,
  DROP COLUMN IF EXISTS name,
  DROP COLUMN IF EXISTS email_verified,
  DROP COLUMN IF EXISTS image,
  DROP COLUMN IF EXISTS updated_at;

-- drop new tables
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS accounts;

-- recreate original view
CREATE VIEW active_users AS
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM users
WHERE deleted_at IS NULL;

COMMIT;
