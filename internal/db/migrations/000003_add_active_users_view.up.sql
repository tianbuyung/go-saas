BEGIN;

CREATE VIEW active_users AS
SELECT id, email, password, salt, provider, provider_id, created_at, deleted_at
FROM users
WHERE deleted_at IS NULL;

COMMIT;
