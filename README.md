# UniDB MCP Server

A shared MCP server that gives AI agents access to your databases. Run one instance, connect multiple projects.

## Features

- **Multi-database**: MySQL, PostgreSQL, SQL Server, MongoDB, Redis, SQLite
- **Web UI**: Add and manage database connections in your browser
- **MCP tools**: `connect`, `query`, `execute`, `schema` for AI agent use
- **SQLite Bridge**: Connect local SQLite files from Docker or remote machines via SSE
- **JWT authentication**: Secure API access with bearer tokens
- **Persistent config**: DSNs and bridge registrations stored in SQLite

## Quick Start

### 1. Configure environment

```bash
cp .env.example .env
```

For local development, enable `DEV_MODE` to skip JWT secret setup:

```env
DEV_MODE=true
ADDR=localhost:9093
DATA_PATH=data/config.db
```

### 2. Run the server

**Development (source):**

```bash
make dev
```

**Production binary:**

```bash
make build
./build/unidb-mcp-server -addr 0.0.0.0:9093 -data /path/to/config.db
```

**Docker Compose:**

```bash
docker-compose up -d
```

### 3. Add a database connection

Open **http://localhost:9093** in your browser and click **Add DSN**.

### 4. Connect Claude Code

Create `.mcp.json` in your project root:

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

All projects sharing this file use the same running UniDB instance.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | — | Required in production. Secret key for JWT tokens. |
| `DEV_MODE` | `false` | Use a built-in default secret for local development. |
| `ADDR` | `localhost:9093` | Server listen address (`host:port`). Use `0.0.0.0:9093` in Docker. |
| `DATA_PATH` | `/app/data/config.db` | Path to the SQLite config database. |
| `UI_PASSWORD` | auto-generated | Web UI password. Set to `false` to disable auth. |

## SQLite Bridge

The SQLite bridge lets you expose a local `.db` file to UniDB over HTTP/SSE — useful when the database lives on a different machine or inside a Docker container.

```bash
docker run \
  -v /path/to/your/database.db:/data/sqlite.db:ro \
  -e BRIDGE_NAME=mydb \
  -e BRIDGE_SECRET=your-secret \
  -e UNIDB_URL=http://unidb:9093 \
  unidb-sqlite-bridge
```

Or run the bridge binary directly:

```bash
./build/unidb-sqlite-bridge \
  -name mydb \
  -file /path/to/database.db \
  -unidb http://localhost:9093 \
  -secret your-secret
```

Register the bridge in the Web UI under **Bridges**, then use it like any other DSN.

## License

MIT
