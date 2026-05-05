-- users: identity only (no credentials)
CREATE TABLE users (
  id            BIGSERIAL PRIMARY KEY,
  public_id     text NOT NULL UNIQUE DEFAULT gen_random_uuid()::text,
  name          text NOT NULL DEFAULT '',
  email         text NOT NULL,
  email_verified boolean NOT NULL DEFAULT FALSE,
  image         text,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at    TIMESTAMPTZ,

  CONSTRAINT email_not_empty CHECK (TRIM(email) <> '')
);

CREATE UNIQUE INDEX unique_users_email_ci ON users (LOWER(email)) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_active ON users(id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_public_id ON users(public_id);

-- accounts: one row per provider per user (credential, google, github, …)
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

CREATE INDEX idx_accounts_user_id ON accounts(user_id);

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

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token);

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

CREATE INDEX idx_verifications_identifier ON verifications(identifier);

-- active_users view (excludes soft-deleted users)
CREATE VIEW active_users AS
SELECT id, public_id, name, email, email_verified, image, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL;
