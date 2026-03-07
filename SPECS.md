# UniDB — Technical Specifications

Reference for developers implementing or extending UniDB.

---

## Architecture

```
HTTP Layer (cmd/mcp-server/main.go)
  ├── Public routes: /, /health, /app.js, /login, /logout, /api/ui/me, /sse
  └── Protected routes: /api/* (JWT middleware via possum.Chain)
        ├── Handlers (internal/handlers/)
        │     ├── ui.go       — DSN CRUD, session auth, UI password
        │     ├── bridge.go   — Bridge registration and management
        │     ├── mcp.go      — MCP HTTP handler
        │     └── health.go   — Health check
        ├── MCP Layer (internal/mcp/)
        │     ├── tools.go    — Tool definitions
        │     └── handlers.go — JSON-RPC dispatch
        ├── Database Layer (internal/database/)
        │     ├── manager.go  — Connection pool
        │     ├── drivers.go  — Driver interface & registry
        │     ├── mysql.go    — MySQL
        │     ├── postgres.go — PostgreSQL
        │     ├── mssql.go    — SQL Server
        │     ├── mongodb.go  — MongoDB
        │     ├── redis.go    — Redis
        │     └── sqlite.go   — SQLite
        └── Store Layer (internal/store/sqlite.go)
              — Persistent config: DSNs, bridges, settings
```

The bridge layer (`internal/bridge/`) is the **client** — it runs as a separate binary that connects to the server over SSE.

---

## Project Structure

```
unidb/
├── cmd/
│   ├── mcp-server/
│   │   └── main.go                 # Entry point, routing, JWT middleware
│   └── sqlite-bridge/
│       ├── main.go                 # SQLite bridge client binary
│       └── docker-entrypoint.sh   # Docker entry script
├── internal/
│   ├── config/
│   │   ├── jwt.go                  # JWT config loader (auto-generate + persist)
│   │   ├── ui_password.go          # UI password config (bcrypt, session)
│   │   └── errors.go
│   ├── database/
│   │   ├── drivers.go              # Driver interface & registry
│   │   ├── manager.go              # Connection pool manager
│   │   ├── mysql.go
│   │   ├── postgres.go
│   │   ├── mssql.go
│   │   ├── mongodb.go
│   │   ├── redis.go
│   │   └── sqlite.go
│   ├── handlers/
│   │   ├── bridge.go               # BridgeManager + BridgeHandler
│   │   ├── mcp.go
│   │   ├── ui.go                   # UIHandler (DSN CRUD, login, password)
│   │   └── health.go
│   ├── mcp/
│   │   ├── tools.go                # MCP tool schemas
│   │   └── handlers.go             # JSON-RPC 2.0 dispatch
│   ├── bridge/
│   │   ├── client.go               # Bridge SSE client
│   │   └── sqlite.go               # SQLite query execution
│   └── store/
│       └── sqlite.go               # DSN & bridge persistence
├── web/
│   ├── index.html                  # Web UI
│   └── app.js                      # Frontend logic
├── .env.example
├── .mcp.example.json
├── docker-compose.yml
├── Dockerfile                      # Main server image
├── Dockerfile.bridge               # SQLite bridge image
└── Makefile
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | auto-generated | HMAC-SHA256 secret for JWT signing. Auto-generated on first start and persisted in SQLite `settings` table. |
| `DEV_MODE` | `false` | When `true`, uses `data/config.db` as default data path. |
| `ADDR` | `localhost:9093` | Server listen address (`host:port`). Use `0.0.0.0:9093` in Docker. |
| `DATA_PATH` | `/app/data/config.db` | Path to SQLite config database. |
| `UI_PASSWORD` | auto-generated | Web UI password. Set to `false` to disable. Auto-generated and stored as bcrypt hash in `settings` on first start. |

---

## API Reference

### Public (no auth)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Web UI |
| `GET` | `/health` | Health check (`{"status":"ok"}`) |
| `GET` | `/app.js` | Frontend JS |
| `POST` | `/login` | Web UI login (session cookie) |
| `POST` | `/logout` | Web UI logout |
| `GET` | `/api/ui/me` | Returns 200 if session valid, 401 otherwise |
| `GET/POST` | `/sse` | Bridge SSE connection (auth via query params) |

### Protected — JWT Bearer token required

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/dsns` | List all DSNs |
| `POST` | `/api/dsns` | Create DSN |
| `PUT` | `/api/dsns` | Update DSN |
| `DELETE` | `/api/dsns` | Delete DSN |
| `POST` | `/api/dsns/{id}/test` | Test a DSN connection |
| `GET/POST` | `/api/mcp` | MCP Streamable HTTP transport |
| `GET` | `/api/bridges` | List registered bridges |
| `PUT` | `/api/bridges` | Update bridge |
| `DELETE` | `/api/bridges` | Delete bridge |
| `POST` | `/api/bridges/register` | Register/reconnect a bridge (public path, self-authenticated via secret) |
| `POST` | `/api/ui/password` | Change UI password (session required) |

---

## MCP Tools Reference

All tools use JSON-RPC 2.0 over `POST /api/mcp`.

### `connect`

Establish a connection from a stored DSN name.

```json
{"name": "connect", "arguments": {"dsn_name": "mydb"}}
```

Returns `{"connection_id": "<uuid>"}`.

### `disconnect`

```json
{"name": "disconnect", "arguments": {"connection_id": "<uuid>"}}
```

### `list_connections`

```json
{"name": "list_connections"}
```

### `list_dsns`

```json
{"name": "list_dsns"}
```

### `query`

```json
{"name": "query", "arguments": {"connection_id": "<uuid>", "sql": "SELECT * FROM users LIMIT 10"}}
```

### `execute`

```json
{"name": "execute", "arguments": {"connection_id": "<uuid>", "sql": "UPDATE users SET active = true WHERE id = 1"}}
```

### `schema`

```json
{"name": "schema", "arguments": {"connection_id": "<uuid>", "table": "users"}}
```

Omit `table` to get schema for all tables.

---

## DSN Format Examples

**MySQL**
```
user:password@tcp(host:3306)/database
user:password@tcp(host:3306)/database?parseTime=true&charset=utf8mb4
```

**PostgreSQL**
```
postgres://user:password@host:5432/database?sslmode=disable
host=localhost port=5432 user=postgres dbname=mydb sslmode=disable
```

**SQL Server**
```
sqlserver://user:password@host:1433?database=mydb
server=host;user id=user;password=password;port=1433;database=mydb
```

**MongoDB**
```
mongodb://host:27017/database
mongodb://user:password@host:27017/database?authSource=admin
```

**Redis**
```
redis://host:6379/0
redis://:password@host:6379/0
```

**SQLite**
```
/path/to/database.db
/app/data/mydb.db
```

---

## JWT Authentication

The server uses HMAC-SHA256 (`HS256`) JWT tokens.

On first start with no `JWT_SECRET` env var, a secret is auto-generated and stored in the `settings` table under key `jwt_secret`. Subsequent restarts reuse the stored secret.

**Dev mode** (`DEV_MODE=true`) uses the default secret: `your-secure-secret-key-here`.

**Generate a token** (Python):

```python
import base64, json, hmac, hashlib, time
secret = b'your-secure-secret-key-here'
header = base64.urlsafe_b64encode(json.dumps({'alg':'HS256','typ':'JWT'}).encode()).rstrip(b'=')
payload = base64.urlsafe_b64encode(json.dumps({'sub':'claude','iat':int(time.time())}).encode()).rstrip(b'=')
sig_input = header + b'.' + payload
sig = base64.urlsafe_b64encode(hmac.new(secret, sig_input, hashlib.sha256).digest()).rstrip(b'=')
print((sig_input + b'.' + sig).decode())
```

---

## SQLite Bridge Protocol

The SQLite bridge runs as a separate process/container and communicates with UniDB via HTTP.

### Flow

```
Bridge binary start
  └─► POST /api/bridges/register   (name, secret, type="sqlite")
        ├── New bridge: 201 Created
        └── Existing name+secret: 200 OK (reconnect)
  └─► GET /sse?name=<name>&secret=<secret>
        ├── Sends keep-alive pings every 30s
        └── Forwards MCP requests as SSE events (event: mcp)
  └─► POST /api/bridges/response?name=<name>&secret=<secret>
        └── Bridge posts JSON-RPC response
```

### Registration idempotency

- Same `name` + correct `secret` → 200 OK (reconnect, no error)
- Same `name` + wrong `secret` → 401 Unauthorized
- New `name` → 201 Created

### Bridge client flags

| Flag | Default | Description |
|------|---------|-------------|
| `-name` | required | Bridge name |
| `-file` | required | Path to SQLite file |
| `-unidb` | `http://localhost:9093` | UniDB server URL |
| `-secret` | auto-generated | Authentication secret |
| `-reconnect` | `true` | Auto-reconnect on disconnect |
| `-reconnect-delay` | `5s` | Delay between reconnect attempts |

### Docker usage

```bash
docker run \
  -v /path/to/your/database.db:/data/sqlite.db:ro \
  -e BRIDGE_NAME=mydb \
  -e BRIDGE_SECRET=your-secret \
  -e UNIDB_URL=http://unidb:9093 \
  -e RECONNECT=true \
  unidb-sqlite-bridge
```

---

## Store Schema

Database: SQLite at `DATA_PATH`.

```sql
CREATE TABLE dsns (
  id         TEXT PRIMARY KEY,      -- UUID
  name       TEXT UNIQUE NOT NULL,
  driver     TEXT NOT NULL,         -- "mysql" | "postgres" | "sqlite"
  dsn        TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE bridges (
  id           TEXT PRIMARY KEY,    -- UUID
  name         TEXT UNIQUE NOT NULL,
  secret       TEXT NOT NULL,       -- plaintext (restrict file access)
  type         TEXT NOT NULL,       -- "sqlite"
  connected    INTEGER DEFAULT 0,   -- boolean
  connected_at DATETIME,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
-- Used for: jwt_secret, ui_password_hash, ui_session_<token>
```

---

## Middleware

Routes are protected by a JWT middleware chain built with `possum.Chain`:

```go
apiHandler := possum.Chain(
    apiMux.ServeHTTP,
    possum.Log,
    possum.Cors(nil),
    jwtMiddleware(jwtCfg),
)
```

`possum.Chain` applies middleware in reverse order (last = outermost). The JWT middleware skips paths in `isPublicPath`:

```go
publicPaths := []string{"/", "/health", "/app.js", "/sse", "/mcp", "/ui/", "/bridges/register"}
```

---

## Build

```bash
# Run in dev mode
make dev

# Build binaries to build/
make build

# Build Docker image
make build-image

# Build with extra tags
make build-image EXTRA_TAGS="v1.0.0 v1.0"

# Run tests
make test

# Lint
make lint
```

Binaries produced:
- `build/unidb-mcp` — main server
- `build/unidb-sqlite-bridge` — SQLite bridge client

---

## Docker Compose

`docker-compose.yml` runs the main server. Mount your SQLite files via a volume override:

```yaml
# docker-compose.override.yml
services:
  unidb-mcp:
    volumes:
      - /path/to/sqlite/files:/host-sqlite:ro
```

Then use `/host-sqlite/your-database.db` as the DSN.

---

## Security Notes

- `JWT_SECRET` and `Bridge.Secret` are stored in plaintext SQLite — restrict access to `DATA_PATH`.
- Never use `DEV_MODE=true` in production (uses a hardcoded secret).
- Run behind a TLS reverse proxy in production.
- `.mcp.json` contains the JWT token — do not commit to public repositories.
- `Bridge.Secret` is tagged `json:"-"` on the store struct but exposed via a `bridgeResponse` wrapper in the List API (needed for Setup Tips in the Web UI).
