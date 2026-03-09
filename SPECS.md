# UniDB — Technical Specifications

Reference for developers implementing or extending UniDB.

---

## Architecture

```
HTTP Layer (cmd/mcp-server/main.go)
  ├── Public routes (no auth): /, /health, /assets/*, /login, /logout, /init,
  │                             /api/ui/me, /api/ui/init-status, /sse, /mcp
  └── Protected routes /api/* (JWT middleware — per-user HS256 bearer token)
        ├── Handlers (internal/handlers/)
        │     ├── ui.go       — DSN CRUD, session auth, password change
        │     ├── user.go     — User management (CRUD, JWT secret)
        │     ├── team.go     — Team management (CRUD, membership, DSN assignment)
        │     ├── bridge.go   — BridgeManager + SSEHandler
        │     ├── mcp.go      — MCP HTTP handler
        │     └── health.go   — Health check
        ├── MCP Layer (internal/mcp/)
        │     ├── tools.go    — Tool definitions / schemas
        │     └── handlers.go — JSON-RPC 2.0 dispatch
        ├── Database Layer (internal/database/)
        │     ├── manager.go  — Connection pool
        │     ├── drivers.go  — Driver interface & registry
        │     ├── mysql.go, postgres.go, mssql.go, mongodb.go,
        │     │   redis.go, etcd.go, sqlite.go
        │     └── bridge.go   — sqlite-bridge driver (proxied via SSE)
        ├── RBAC Layer (internal/rbac/)
        │     └── rbac.go     — Role definitions and permission checks
        └── Store Layer (internal/store/sqlite.go)
              — Persistent config: users, teams, DSNs, sessions, settings
```

The bridge layer (`internal/bridge/`) is the **client binary** — it runs as a separate process and connects to the server over SSE.

---

## Project Structure

```
unidb/
├── backend/
│   ├── cmd/
│   │   ├── mcp-server/
│   │   │   └── main.go                 # Entry point, routing, middleware
│   │   └── sqlite-bridge/
│   │       ├── main.go                 # SQLite bridge client binary
│   │       └── docker-entrypoint.sh    # Docker entry script
│   └── internal/
│       ├── database/
│       │   ├── drivers.go              # Driver interface & registry
│       │   ├── manager.go              # Connection pool manager
│       │   ├── mysql.go, postgres.go, mssql.go
│       │   ├── mongodb.go, redis.go, etcd.go, sqlite.go
│       │   └── bridge.go               # sqlite-bridge driver
│       ├── handlers/
│       │   ├── bridge.go               # BridgeManager + SSEHandler
│       │   ├── mcp.go                  # MCP HTTP handler wrapper
│       │   ├── ui.go                   # UIHandler (DSN CRUD, login, session)
│       │   ├── user.go                 # UserHandler
│       │   ├── team.go                 # TeamHandler
│       │   └── health.go
│       ├── mcp/
│       │   ├── tools.go                # MCP tool schemas
│       │   └── handlers.go             # JSON-RPC 2.0 dispatch
│       ├── bridge/
│       │   ├── client.go               # Bridge SSE client
│       │   └── sqlite.go               # SQLite query execution
│       ├── rbac/
│       │   └── rbac.go                 # RBAC roles and permissions
│       └── store/
│           └── sqlite.go               # All persistence
├── frontend/
│   └── src/
│       ├── App.vue
│       ├── api.ts
│       ├── types.ts
│       ├── components/
│       └── composables/
├── docker/
│   ├── Dockerfile
│   ├── Dockerfile.bridge
│   ├── docker-compose.yml
│   └── .env.example
└── Makefile
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDR` | `localhost:9093` | Server listen address. Use `0.0.0.0:9093` in Docker. |
| `DATA_PATH` | `/app/data/config.db` (dev: `data/config.db`) | Path to SQLite config database. |
| `FRONTEND_PATH` | `frontend/dist` (dev: `../frontend/dist`) | Path to compiled frontend assets. |
| `DEV_MODE` | `false` | When `true`, uses local default paths for dev convenience. |

No global `JWT_SECRET` env var — JWT secrets are **per-user**, generated at user creation and stored in the `users` table. There is no `UI_PASSWORD` env var.

---

## Server Flags

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `-addr` | `ADDR` | `localhost:9093` | Listen address |
| `-data` | `DATA_PATH` | `/app/data/config.db` | Config database path |
| `-frontend` | `FRONTEND_PATH` | `frontend/dist` | Frontend dist path |

---

## Authentication Model

### Web UI (session-based)

- Cookie name: `ui_session`
- Duration: 24 hours
- Session token: 32-byte random, base64url-encoded
- Stored in `settings` table as:
  - `ui_session_{token}` → expiry (RFC3339)
  - `ui_session_user_{token}` → user ID
- Cookie flags: `HttpOnly`, `SameSiteLax`

### MCP / API (JWT bearer)

- Header: `Authorization: Bearer <token>`
- Algorithm: HMAC-SHA256 (`HS256`)
- Each user has their own `jwt_secret` (stored in `users.jwt_secret`)
- Server validates the token against **all** user JWT secrets via `ListAllJWTSecrets()`
- JWT secret can be refreshed via `POST /api/users/{id}/jwt-secret`

### RBAC

Roles are determined at request time by checking team membership:

```go
isAdmin, _ := store.IsUserAdmin(userID)
// IsUserAdmin: SELECT COUNT(*) FROM user_teams JOIN teams WHERE name='admin' AND user_id=?
role := "member" // or "admin"
```

| Role | How assigned | Permissions |
|------|-------------|-------------|
| `admin` | Member of the `admin` team | All permissions |
| `member` | Any other user | `dsn:read`, `dsn:test`, `team:read` |

Permission constants:

```go
PermDSNRead    = "dsn:read"
PermDSNWrite   = "dsn:write"
PermDSNDelete  = "dsn:delete"
PermDSNTest    = "dsn:test"
PermUserRead   = "user:read"
PermUserWrite  = "user:write"
PermUserDelete = "user:delete"
PermTeamRead   = "team:read"
PermTeamWrite  = "team:write"
PermTeamDelete = "team:delete"
```

### DSN Visibility

- **Admins**: see all DSNs (`store.List()`)
- **Members**: see only DSNs in their teams (`store.ListDSNsForUser(userID)`)
  - Query: `SELECT DISTINCT dsns ... JOIN dsn_teams ... JOIN user_teams WHERE user_id=?`

---

## API Reference

### Public (no auth required)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Web UI (serves frontend) |
| `GET` | `/health` | Health check → `{"status":"ok"}` |
| `GET` | `/assets/*` | Static frontend assets |
| `POST` | `/login` | Session login. Body: `{username, password}`. Sets `ui_session` cookie. |
| `POST` | `/logout` | Session logout. Clears cookie. |
| `POST` | `/init` | One-time admin setup. Body: `{username, password, jwt_secret?}`. |
| `GET` | `/api/ui/me` | Returns session user info or 401. |
| `GET` | `/api/ui/init-status` | Returns `{initialized: bool}`. |
| `GET` | `/sse` | Bridge SSE endpoint. Query: `?name=&secret=`. |
| `GET` | `/mcp` | MCP SSE stream (reserved). |
| `POST` | `/mcp` | MCP JSON-RPC handler (no JWT required). |

**`/api/ui/me` response:**
```json
{
  "authenticated": true,
  "username": "alice",
  "init_admin_id": "<uuid>",
  "is_admin": true
}
```

### Protected — JWT Bearer required

**DSN management** (RBAC: read/write/delete/test per permission):

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| `GET` | `/api/dsns` | `dsn:read` | List DSNs (filtered by team for members) |
| `POST` | `/api/dsns` | `dsn:write` | Create DSN. Body: `{name, driver, dsn}`. |
| `PUT` | `/api/dsns?id=<id>` | `dsn:write` | Update DSN. Body: `{name, driver, dsn}`. |
| `DELETE` | `/api/dsns?id=<id>` | `dsn:delete` | Delete DSN. |
| `POST` | `/api/dsns/{id}/test` | `dsn:test` | Test connection. Returns `{success, duration?, error?}`. |

**Session-protected (Web UI only):**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/ui/password` | Change password. Body: `{current, new}`. Session required. |

**User management** (admin only):

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| `GET` | `/api/users` | `user:read` | List users. |
| `POST` | `/api/users` | `user:write` | Create user. Body: `{username, password}`. Returns `{user, jwt_secret}`. |
| `PUT` | `/api/users?id=<id>` | `user:write` | Update password. Body: `{password}`. |
| `DELETE` | `/api/users?id=<id>` | `user:delete` | Delete user. Cannot delete self or init admin. |
| `GET` | `/api/users/{id}/jwt-secret` | `user:read` | Get user's JWT secret. |
| `POST` | `/api/users/{id}/jwt-secret` | `user:write` | Rotate JWT secret. Returns `{jwt_secret}`. |

**Team management** (admin for write; members can read):

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| `GET` | `/api/teams` | `team:read` | List teams. |
| `POST` | `/api/teams` | `team:write` | Create team. Body: `{name}`. |
| `DELETE` | `/api/teams?id=<id>` | `team:delete` | Delete team. Cannot delete `admin` team. |
| `GET` | `/api/teams/{id}/users` | `team:read` | List team members. |
| `POST` | `/api/teams/{id}/users` | `team:write` | Add user to team. Body: `{user_id}`. Init admin cannot be removed from admin team. |
| `DELETE` | `/api/teams/{id}/users` | `team:write` | Remove user from team. Body: `{user_id}`. |
| `GET` | `/api/teams/{id}/dsns` | `team:read` | List DSNs assigned to team. |
| `POST` | `/api/teams/{id}/dsns` | `team:write` | Assign DSN to team. Body: `{dsn_id}`. Cannot assign to admin team. |
| `DELETE` | `/api/teams/{id}/dsns` | `team:write` | Remove DSN from team. Body: `{dsn_id}`. Cannot modify admin team. |

---

## MCP Protocol

Transport: JSON-RPC 2.0 over `POST /mcp` (no JWT required — bearer token not needed for MCP).

**Methods handled:**

| Method | Description |
|--------|-------------|
| `initialize` | Returns `{protocolVersion: "2025-03-26", serverInfo: {name: "unidb-mcp", version: "1.0.0"}}` |
| `notifications/initialized` | No-op notification |
| `tools/list` | Returns tool definitions |
| `tools/call` | Dispatches to tool handler |

### MCP Tools

#### `connect`
Establish connection using stored DSN name.
```json
{"name": "connect", "arguments": {"dsn_name": "mydb"}}
```
Returns: `{"connection_id": "<uuid>"}`

#### `disconnect`
```json
{"name": "disconnect", "arguments": {"connection_id": "<uuid>"}}
```

#### `list_dsns`
```json
{"name": "list_dsns"}
```
Returns DSNs visible to the requesting JWT user, with `name`, `driver`, `created_at`.

#### `list_connections`
```json
{"name": "list_connections"}
```
Returns active connections with `id`, `dsn_name`, `driver`, `connected_at`.

#### `query`
```json
{"name": "query", "arguments": {"connection_id": "<uuid>", "sql": "SELECT * FROM users LIMIT 10"}}
```
Returns: `{"columns": [...], "rows": [...], "row_count": N}`

#### `execute`
```json
{"name": "execute", "arguments": {"connection_id": "<uuid>", "sql": "UPDATE users SET active = true WHERE id = 1"}}
```
Returns: `{"rows_affected": N}`

#### `schema`
```json
{"name": "schema", "arguments": {"connection_id": "<uuid>", "table": "users"}}
```
Omit `table` for all tables. Returns: `{"tables": {"tablename": [...columns...]}}`

---

## Store Schema

All persistence in a single SQLite file at `DATA_PATH`.

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,         -- UUID
    username      TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,            -- bcrypt
    jwt_secret    TEXT NOT NULL,            -- per-user HMAC secret
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS teams (
    id         TEXT PRIMARY KEY,
    name       TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_teams (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, team_id)
);

CREATE TABLE IF NOT EXISTS dsns (
    id         TEXT PRIMARY KEY,
    name       TEXT UNIQUE NOT NULL,
    driver     TEXT NOT NULL,               -- see supported drivers
    dsn        TEXT NOT NULL,               -- connection string or bridge secret
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- For driver='sqlite-bridge': dsn field stores the bridge secret (plaintext)

CREATE TABLE IF NOT EXISTS dsn_teams (
    dsn_id  TEXT NOT NULL REFERENCES dsns(id) ON DELETE CASCADE,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    PRIMARY KEY (dsn_id, team_id)
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
-- Keys used:
--   init_admin_user_id          — ID of the initial admin user (set once)
--   ui_session_{token}          — session expiry (RFC3339)
--   ui_session_user_{token}     — user ID for session

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_teams_name ON teams(name);
CREATE INDEX IF NOT EXISTS idx_dsns_name ON dsns(name);
```

**Error sentinels:**
```go
ErrDSNNotFound  // "DSN not found"
ErrDSNExists    // "DSN name already exists"
ErrUserNotFound // "user not found"
ErrUserExists   // "username already exists"
ErrTeamNotFound // "team not found"
ErrTeamExists   // "team already exists"
```

---

## Go Type Definitions

```go
// store/sqlite.go
type User struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    PasswordHash string    `json:"-"`
    JWTSecret    string    `json:"-"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type Team struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

type DSN struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Driver    string    `json:"driver"`
    DSN       string    `json:"dsn"`
    Connected bool      `json:"connected,omitempty"` // in-memory only, not stored
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

---

## SQLite Bridge

The bridge is a separate binary (`unidb-sqlite-bridge`) that runs alongside the SQLite file and connects to UniDB over HTTP/SSE.

### Setup flow

1. Admin creates a `sqlite-bridge` DSN in the UI: choose a name and a secret. The secret is stored in `dsns.dsn`.
2. Run the bridge binary with matching `-name` and `-secret`.
3. Bridge connects to `GET /sse?name=&secret=` — server authenticates by looking up the DSN and comparing `dsn` field.
4. On success, bridge appears as 🟩 (connected) in the UI. Connection state is in-memory only; `dsns.updated_at` is bumped on connect/disconnect.

**There is no auto-registration.** The DSN must exist before the bridge can connect. Unknown names or wrong secrets return 401.

### Bridge SSE protocol

```
Bridge → GET /sse?name={name}&secret={secret}
Server sends:
  event: connected   data: {"status":"connected","name":"mydb"}
  event: ping        data: {"type":"ping"}          (every 30s)
  event: mcp         data: <JSON-RPC request>       (when AI calls a tool)

Bridge → POST /api/bridges/response?name={name}&secret={secret}
  body: <JSON-RPC response>
```

### Bridge binary flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `-name` | yes | — | Bridge name (must match DSN name in UI) |
| `-file` | yes | — | Path to SQLite `.db` file |
| `-unidb` | no | `http://localhost:9093` | UniDB server URL |
| `-secret` | no | auto-generated UUID | Auth secret (must match DSN secret in UI) |
| `-reconnect` | no | `true` | Auto-reconnect on disconnect |
| `-reconnect-delay` | no | `5s` | Delay between reconnect attempts |

### Docker usage

```bash
docker run \
  -v /path/to/your/database.db:/data/sqlite.db:ro \
  -e BRIDGE_NAME=mydb \
  -e BRIDGE_SECRET=your-secret \
  -e UNIDB_URL=http://unidb-host:9093 \
  mikespook/unidb-sqlite-bridge
```

Docker environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BRIDGE_NAME` | yes | — | Bridge name |
| `BRIDGE_SECRET` | yes | — | Auth secret |
| `UNIDB_URL` | no | `http://localhost:9093` | UniDB server URL |
| `RECONNECT` | no | `true` | Auto-reconnect |

The SQLite file must be mounted at `/data/sqlite.db` inside the container.

---

## DSN Format Reference

| Driver | DSN format |
|--------|-----------|
| `mysql` | `user:password@tcp(host:3306)/database` |
| `postgres` / `postgresql` | `postgres://user:password@host:5432/database?sslmode=disable` |
| `mssql` / `sqlserver` | `sqlserver://user:password@host:1433?database=mydb` |
| `mongodb` | `mongodb://user:password@host:27017/database` |
| `redis` | `redis://:password@host:6379/0` |
| `etcd` | `http://host:2379` or `etcd://user:password@host:2379` |
| `sqlite` / `sqlite3` | `/path/to/database.db` |
| `sqlite-bridge` | *(dsn field = auth secret; name must match bridge)* |

---

## Middleware

JWT middleware is applied to all `/api/*` routes. Public paths bypass it:

```go
publicPaths = []string{"/", "/health", "/assets/", "/sse", "/mcp",
                       "/ui/", "/ui/init-status"}
```

The JWT middleware validates the Bearer token against all per-user secrets via `store.ListAllJWTSecrets()`. If the token matches any user's secret, the request proceeds.

Session middleware (`UIHandler.SessionMiddleware`) is applied only to the password-change route — it requires a valid `ui_session` cookie in addition to (or instead of) JWT.

---

## Build

```bash
# Dev
make dev           # backend (DEV_MODE=true, localhost:9093)
make dev-frontend  # Vite dev server (proxy to :9093)

# Production build
make build         # build/unidb-mcp-server + build/unidb-sqlite-bridge + frontend

# Docker
make build-image-server          # mikespook/unidb-mcp-server:latest
make build-image-sqlite-bridge   # mikespook/unidb-sqlite-bridge:latest
make build-image-server EXTRA_TAGS="v1.0.1 v1.0"

# Quality
make test
make test-coverage    # generates coverage.html
make lint
make fmt
make tidy
```

Build flags: `CGO_ENABLED=1`, `-ldflags "-s -w"`.

---

## Security Notes

- Per-user `jwt_secret` values are stored **plaintext** in SQLite — restrict filesystem access to `DATA_PATH`.
- `sqlite-bridge` auth secrets are stored plaintext in `dsns.dsn` — same restriction applies.
- Never use `DEV_MODE=true` in production.
- Run behind a TLS reverse proxy in production.
- `.mcp.json` contains JWT tokens — do not commit to public repositories.
- The initial admin user (`init_admin_user_id` in settings) cannot be removed from the `admin` team (enforced in both the frontend and backend).
- Users cannot delete themselves (`DELETE /api/users` checks caller ID).
- DSNs cannot be assigned to the `admin` team (enforced in `TeamHandler.AddDSN`).
