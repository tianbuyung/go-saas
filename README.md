# SaaS Backend

A production-ready Go HTTP API with layered architecture, JWT authentication, rolling refresh tokens, and instant token revocation.

## Stack

- **Go** + **Gin** (behind a router abstraction — swap to Echo/Fiber without touching handlers)
- **PostgreSQL** via **pgx/v5** + **pgBouncer** (transaction mode)
- **Redis** for session cache and JWT blocklist
- **sqlc** for type-safe query generation
- **golang-migrate** for schema migrations
- **zap** for structured logging
- **argon2id** for password hashing
- **HS256/384/512** JWT with `iss`, `aud`, `jti` claims

## Repository layout

```
service/
  cmd/api/          — HTTP server entrypoint
  cmd/migrate/      — migration runner (up/down/force/drop)
  internal/
    bootstrap/      — dependency wiring and route registration
    config/         — .env loader (viper)
    db/             — sqlc-generated code + pgxpool connection
    domain/         — plain Go structs (no framework deps)
    handler/        — HTTP handlers (accept router.Context, never *gin.Context)
    iam/            — JWT, argon2id, SessionStore interface + Redis/Hybrid impls, Blocklist interface + Redis/Noop impls, opaque token generator
    module/         — feature modules: auth, user
    repository/     — UserRepository, AccountRepository, DBSessionStore interfaces + pgx impls
    router/         — Router/Context interfaces, GinEngine adapter, ZapLogger middleware, context key constants
    service/        — business logic: IamService, UserService
  pkg/
    cache/          — Redis client wrapper (Cacher interface + *Redis implementation)
    logger/         — zap production logger
  sql/
    schema.sql      — source of truth for the DB schema
    queries.sql     — sqlc-annotated queries
  internal/db/migrations/  — golang-migrate migration files
```

## Setup

```bash
# 1. Copy env file
cp service/.env.example service/.env
# Fill in values — see Configuration section below

# 2. Start infrastructure (Postgres + pgBouncer + Redis)
cd service && docker compose up -d

# 3. Run migrations
go run ./cmd/migrate up

# 4. Start the server
go run ./cmd/api
```

## Configuration

All settings live in `service/.env`.

| Variable | Description |
|---|---|
| `ENVIRONMENT` | `development` or `production` (production enables Gin release mode, blocks drop/down migrations) |
| `PORT` | HTTP listen port |
| `DATABASE_URL` | Direct Postgres URL — migrations only |
| `DATABASE_POOL_URL` | pgBouncer URL — application runtime |
| `JWT_SECRET` | HMAC signing key |
| `JWT_EXPIRE` | Access token TTL as Go duration: `5m`, `15m`, `1h` |
| `JWT_TOKEN_ISSUER` | `iss` claim — validated on every request; leave empty to skip |
| `JWT_TOKEN_AUDIENCE` | `aud` claim — validated on every request; leave empty to skip |
| `JWT_ALGORITHM` | `HS256` (default) \| `HS384` \| `HS512` |
| `SESSION_STORE` | `db` (default) \| `redis` \| `hybrid` — see Session store modes |
| `REFRESH_EXPIRE_HOURS` | Refresh token TTL in hours (default `168` = 7 days) |
| `REDIS_URL` | Redis connection URL — required for `redis`/`hybrid` session store and JWT blocklist |

## Session store modes

| `SESSION_STORE` | Behaviour |
|---|---|
| `db` | Refresh token sessions persisted in Postgres `sessions` table |
| `redis` | Sessions stored in Redis only — faster, but lost on Redis restart |
| `hybrid` | Redis as write-through cache in front of DB; miss → DB fallback → re-populate Redis |

> The JWT blocklist always uses Redis when `REDIS_URL` is set, regardless of `SESSION_STORE`.
> If `REDIS_URL` is unset, a no-op blocklist is used (fail-open: logout still invalidates the refresh token, but the access token lives until its natural expiry).

## API endpoints

### Auth

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/register` | — | Create account |
| `POST` | `/auth/login` | — | Login, returns `access_token` + `refresh_token` |
| `POST` | `/auth/refresh` | — | Rotate refresh token, returns new pair |
| `POST` | `/auth/logout` | Bearer JWT | Revoke session + blocklist JWT immediately |

### User

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/me` | Bearer JWT | Get current user profile |

### Health

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness check |

## Request / Response examples

### Register
```http
POST /auth/register
Content-Type: application/json

{
  "name": "Alice",
  "email": "alice@example.com",
  "password": "strongpassword"
}
```

### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "alice@example.com",
  "password": "strongpassword"
}
```
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<opaque>"
}
```

### Refresh (rolling — always returns a new pair)
```http
POST /auth/refresh
Content-Type: application/json

{ "refresh_token": "<opaque>" }
```
```json
{
  "access_token": "<new-jwt>",
  "refresh_token": "<new-opaque>"
}
```

### Logout
```http
POST /auth/logout
Authorization: Bearer <jwt>
Content-Type: application/json

{ "refresh_token": "<opaque>" }
```

## Auth design

```
Login  → JWT (short-lived, stateless) + opaque refresh token (long-lived, stored in sessions)
Request → validate JWT signature + iss/aud claims + Redis blocklist check (no DB hit)
Refresh → delete old session, create new session (fresh TTL), return new JWT + new refresh token
Logout  → delete session + add JWT jti to Redis blocklist (instant revocation within JWT TTL)
```

**Access token** — never stored. Validated via HMAC signature. Claims: `sub` (public_id), `jti`, `iat`, `exp`, `iss`, `aud`.

**Refresh token** — 32-byte cryptographically random opaque string. Stored in `sessions` table (or Redis). Single-use: rotated on every `/auth/refresh` call. Replayed tokens are rejected with 401.

**JWT blocklist** — on logout, the token's `jti` is stored in Redis with TTL matching the token's remaining lifetime. Every authenticated request checks the blocklist before processing.

## Security

| Concern | Implementation |
|---|---|
| Internal IDs | `domain` struct int64 IDs tagged `json:"-"` — never appear in API responses |
| Sensitive fields | `Password` and `Salt` fields tagged `json:"-"` |
| Input validation | Empty `email` or `password` rejected at the handler layer with 400 before reaching the service |
| Error messages | Service errors never forwarded verbatim to clients — all responses use sanitised messages |
| JWT algorithm | `ParseToken` pins the exact configured HMAC variant — a token signed with HS512 is rejected by an HS256-configured server |
| JWT revocation | Logout adds `jti` to Redis blocklist with TTL = remaining token lifetime; every authenticated request checks the blocklist |
| Refresh token | Single-use rolling rotation — replaying a consumed refresh token returns 401 immediately |
| Password hashing | argon2id with a 16-byte random salt per password |

## Testing

Run all commands from `service/`.

### Unit tests

No infrastructure required:

```bash
go test ./...
go test -race ./...
go test -v ./internal/iam/...
go test -run TestIamService_Login
```

### Integration tests

Require a running Postgres instance. Set `TEST_DATABASE_URL` or ensure `DATABASE_URL` is in `.env`:

```bash
docker compose up -d
go test -tags integration ./internal/repository/...
```

### E2E tests

Require the full stack (Postgres + Redis + live server):

```bash
docker compose up -d
go run ./cmd/migrate up
go run ./cmd/api &
go test -tags e2e ./internal/e2e/...
# or override the base URL:
E2E_BASE_URL=http://localhost:3002 go test -tags e2e ./internal/e2e/...
```

### Coverage

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

| Layer | Target |
|---|---|
| `iam/` | 100% |
| `service/` | 90% |
| `handler/` | 80% |
| `repository/` | 70% (integration tests) |

## Development commands

Run all commands from `service/`:

```bash
docker compose up -d          # start Postgres + pgBouncer + Redis
go run ./cmd/api              # run the API server
go run ./cmd/migrate up       # apply migrations
go run ./cmd/migrate down     # rollback one migration
go run ./cmd/migrate force 3  # force migration version
sqlc generate                 # regenerate db/ after editing sql/
go build ./...                # build check
go vet ./...                  # vet
go test ./...                        # unit tests (no infrastructure needed)
go test -race ./...                  # unit tests with race detector
go test -tags integration ./internal/repository/...  # integration tests (requires Postgres)
go test -tags e2e ./internal/e2e/... # E2E tests (requires full stack running)
```

## Database schema

Four tables following the better-auth pattern:

| Table | Purpose |
|---|---|
| `users` | Identity — `public_id` (UUID) is the external identifier; `id` (BIGSERIAL) is internal FK only |
| `accounts` | Credentials per provider — `provider_id='credential'` for password auth; OAuth provider tokens for future OAuth flows |
| `sessions` | Active refresh token sessions |
| `verifications` | Email verification and password reset tokens |

All queries target the `active_users` view which excludes soft-deleted rows.
