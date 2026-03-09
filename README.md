# UniDB MCP Server

A shared MCP server that gives AI agents access to your databases. Run one instance, connect multiple projects and multiple users.

## Features

- **Multi-database**: MySQL, PostgreSQL, SQL Server, MongoDB, Redis, etcd, SQLite
- **Web UI**: Add and manage database connections in your browser
- **MCP tools**: `connect`, `query`, `execute`, `schema` for AI agent use
- **SQLite Bridge**: Connect local SQLite files from Docker or remote machines via SSE
- **JWT authentication**: Per-user bearer tokens for secure MCP API access
- **Team-based access control**: Assign DSNs to teams; users see only their team's connections
- **Persistent config**: All data stored in a local SQLite database

## Supported Drivers

| Driver | DSN format |
|--------|-----------|
| `mysql` | `user:password@tcp(host:3306)/database` |
| `postgres` | `postgres://user:password@host:5432/database?sslmode=disable` |
| `mssql` | `sqlserver://user:password@host:1433?database=mydb` |
| `mongodb` | `mongodb://user:password@host:27017/database` |
| `redis` | `redis://:password@host:6379/0` |
| `etcd` | `http://host:2379` or `etcd://user:password@host:2379` |
| `sqlite` | `/path/to/database.db` |
| `sqlite-bridge` | *(configured via Web UI — see SQLite Bridge section)* |

## Quick Start

### 1. Run the server

**Docker Compose (recommended):**

```bash
cp docker/.env.example .env
# Edit .env as needed
docker compose -f docker/docker-compose.yml up -d
```

**Development (from source):**

```bash
make dev
```

**Production binary:**

```bash
make build
./build/unidb-mcp-server -addr 0.0.0.0:9093 -data /path/to/config.db
```

### 2. First-time setup

Open **http://localhost:9093** in your browser. On first run, an **Init Wizard** will appear — create your admin account and optionally set a JWT secret. Once submitted, you'll be redirected to the login page.

> The init wizard runs only once. The initial admin account cannot be removed from the admin team.

### 3. Add a database connection (admin)

Log in as admin, click **Add DSN**, fill in the name, driver, and connection string. Save it.

### 4. Assign the DSN to a team (admin)

Open **Access Management** → **Teams**, select a team, and assign the DSN to it. Users in that team will see the DSN in their connection list.

### 5. Connect Claude Code

Get your JWT token from **Access Management** → **Users** → **Show JWT Secret**. Create `.mcp.json` in your project root:

```json
{
  "mcpServers": {
    "unidb": {
      "type": "http",
      "url": "http://localhost:9093/api/mcp",
      "headers": {
        "Authorization": "Bearer <your-jwt-token>"
      }
    }
  }
}
```

All projects sharing this config use the same running UniDB instance.

## Configuration

Server configuration is done via environment variables (or a `.env` file):

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDR` | `localhost:9093` | Listen address. Use `0.0.0.0:9093` in Docker. |
| `DATA_PATH` | `/app/data/config.db` | Path to the SQLite config database. |
| `DEV_MODE` | `false` | Skip JWT validation for local development. |

Server flags (override env vars):

| Flag | Env var | Description |
|------|---------|-------------|
| `-addr` | `ADDR` | Listen address |
| `-data` | `DATA_PATH` | Config database path |
| `-frontend` | `FRONTEND_PATH` | Path to compiled frontend assets |

## Access Control

UniDB uses a **role + team** model:

| Role | How assigned | Permissions |
|------|-------------|-------------|
| **Admin** | Member of the `admin` team | Full access: manage users, teams, all DSNs |
| **Member** | Any other user | Read/test DSNs assigned to their teams |

**DSN visibility**: Members only see DSNs that have been assigned to at least one team they belong to. Admins see all DSNs.

**Teams**: Admins create teams, add users, and assign DSNs to teams. The `default` team is created automatically on first run.

## SQLite Bridge

The SQLite bridge connects a local `.db` file to UniDB over HTTP/SSE — useful when the database lives on a remote machine or inside a Docker container.

### Setup

**1. Create the bridge DSN in the Web UI (admin only)**

Click **Add DSN**, choose driver `sqlite-bridge`, enter a name and a secret. Save it. The secret is what the bridge client will use to authenticate.

**2. Run the bridge client**

Docker:

```bash
docker run \
  -v /path/to/your/database.db:/data/sqlite.db:ro \
  -e BRIDGE_NAME=mydb \
  -e BRIDGE_SECRET=your-secret \
  -e UNIDB_URL=http://unidb-host:9093 \
  mikespook/unidb-sqlite-bridge
```

Binary:

```bash
./build/unidb-sqlite-bridge \
  -name mydb \
  -file /path/to/database.db \
  -unidb http://localhost:9093 \
  -secret your-secret
```

The bridge connects to UniDB and shows as 🟩 (connected) or 🟥 (disconnected) in the connection list.

### Bridge flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `-name` | yes | — | Bridge name (must match the DSN name created in the UI) |
| `-file` | yes | — | Path to the SQLite `.db` file |
| `-unidb` | no | `http://localhost:9093` | UniDB server URL |
| `-secret` | no | auto-generated | Auth secret (must match the one set in the UI) |
| `-reconnect` | no | `true` | Auto-reconnect on connection loss |
| `-reconnect-delay` | no | `5s` | Delay between reconnect attempts |

### Bridge Docker environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BRIDGE_NAME` | yes | — | Bridge name |
| `BRIDGE_SECRET` | yes | — | Auth secret |
| `UNIDB_URL` | no | `http://localhost:9093` | UniDB server URL |
| `RECONNECT` | no | `true` | Auto-reconnect |

The SQLite file must be mounted at `/data/sqlite.db` inside the container.

## Development

```bash
# Run backend + frontend dev servers
make dev           # backend (localhost:9093, DEV_MODE=true)
make dev-frontend  # frontend (Vite, proxied to :9093)

# Build
make build          # compile binaries + frontend into build/
make build-frontend # frontend only

# Tests & lint
make test
make test-coverage
make lint

# Docker images
make build-image-server         # mikespook/unidb-mcp-server:latest
make build-image-sqlite-bridge  # mikespook/unidb-sqlite-bridge:latest

# Add extra tags
make build-image-server EXTRA_TAGS="v1.0.1 v1.0"
```

## License

GPL-3.0
