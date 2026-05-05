BEGIN;

-- accounts: stores credentials and OAuth tokens per provider
CREATE TABLE accounts (
  id                       BIGSERIAL PRIMARY KEY,
  user_id                  BIGINT NOT NULL,
  account_id               text NOT NULL,
  provider_id              text NOT NULL,
  access_token             text,
  refresh_token            text,
  access_token_expires_at  TIMESTAMPTZ,
  refresh_token_expires_at TIMESTAMPTZ,
  scope                    text,
  id_token                 text,
  password                 text,
  salt                     text,
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT accounts_provider_account_unique UNIQUE (provider_id, account_id)
);

-- sessions: active login sessions
CREATE TABLE sessions (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT NOT NULL,
  token      text NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  ip_address text,
  user_agent text,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- verifications: email-verification and password-reset tokens
CREATE TABLE verifications (
  id         BIGSERIAL PRIMARY KEY,
  identifier text NOT NULL,
  value      text NOT NULL,
  type       text NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT verifications_identifier_type_unique UNIQUE (identifier, type)
);

-- migrate existing credential users → accounts
INSERT INTO accounts (user_id, account_id, provider_id, password, salt)
SELECT id, LOWER(email), 'credential', password, salt
FROM users
WHERE password IS NOT NULL AND salt IS NOT NULL;

-- migrate existing OAuth users → accounts
INSERT INTO accounts (user_id, account_id, provider_id)
SELECT id, provider_id, provider
FROM users
WHERE provider IS NOT NULL AND provider_id IS NOT NULL;

-- extend users: add public_id, identity fields, drop credential columns
ALTER TABLE users
  ADD COLUMN public_id     text NOT NULL DEFAULT gen_random_uuid()::text,
  ADD COLUMN name          text NOT NULL DEFAULT '',
  ADD COLUMN email_verified boolean NOT NULL DEFAULT FALSE,
  ADD COLUMN image         text,
  ADD COLUMN updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- enforce public_id uniqueness
ALTER TABLE users ADD CONSTRAINT users_public_id_unique UNIQUE (public_id);

-- drop old auth constraints before dropping columns
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_auth_method;
DROP INDEX IF EXISTS unique_provider_identity;

-- drop active_users view before altering users columns
DROP VIEW IF EXISTS active_users;

-- drop credential/oauth columns now in accounts
ALTER TABLE users
  DROP COLUMN password,
  DROP COLUMN salt,
  DROP COLUMN provider,
  DROP COLUMN provider_id;

-- change created_at to TIMESTAMPTZ if still TIMESTAMP
ALTER TABLE users ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE users ALTER COLUMN deleted_at TYPE TIMESTAMPTZ USING deleted_at AT TIME ZONE 'UTC';

-- rebuild view
CREATE VIEW active_users AS
SELECT id, public_id, name, email, email_verified, image, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL;

-- indexes
CREATE INDEX idx_users_public_id ON users(public_id);
CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_verifications_identifier ON verifications(identifier);

COMMIT;
