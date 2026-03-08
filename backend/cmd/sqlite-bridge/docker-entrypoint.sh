#!/bin/sh
set -e

SQLITE_FILE="/data/sqlite.db"

if [ ! -f "$SQLITE_FILE" ]; then
    echo ""
    echo "Error: No SQLite file found at $SQLITE_FILE"
    echo ""
    echo "Mount your SQLite database file to $SQLITE_FILE, for example:"
    echo ""
    echo "  docker run \\"
    echo "    -v /path/to/your/database.db:/data/sqlite.db:ro \\"
    echo "    -e BRIDGE_NAME=mydb \\"
    echo "    -e BRIDGE_SECRET=your-secret \\"
    echo "    -e UNIDB_URL=http://unidb:9093 \\"
    echo "    unidb-bridge"
    echo ""
    echo "Environment variables:"
    echo "  BRIDGE_NAME    Bridge name to register with UniDB (required)"
    echo "  BRIDGE_SECRET  Authentication secret (required)"
    echo "  UNIDB_URL      UniDB server URL (default: http://localhost:9093)"
    echo "  RECONNECT      Auto-reconnect on connection loss (default: true)"
    echo ""
    exit 1
fi

# Validate required env vars
if [ -z "$BRIDGE_NAME" ]; then
    echo "Error: BRIDGE_NAME environment variable is required"
    exit 1
fi

if [ -z "$BRIDGE_SECRET" ]; then
    echo "Error: BRIDGE_SECRET environment variable is required"
    exit 1
fi

exec ./unidb-sqlite-bridge \
    -name "$BRIDGE_NAME" \
    -file "$SQLITE_FILE" \
    -unidb "${UNIDB_URL:-http://localhost:9093}" \
    -secret "$BRIDGE_SECRET" \
    -reconnect "${RECONNECT:-true}"
