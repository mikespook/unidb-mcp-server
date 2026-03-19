#!/bin/sh
set -e

BOLTDB_FILE="/data/boltdb.db"

if [ ! -f "$BOLTDB_FILE" ]; then
    echo ""
    echo "Error: No BoltDB file found at $BOLTDB_FILE"
    echo ""
    echo "Mount your BoltDB database file to $BOLTDB_FILE, for example:"
    echo ""
    echo "  docker run \\"
    echo "    -v /path/to/your/database.db:/data/boltdb.db:ro \\"
    echo "    -e BRIDGE_NAME=mydb \\"
    echo "    -e BRIDGE_SECRET=your-secret \\"
    echo "    -e UNIDB_URL=http://unidb:9093 \\"
    echo "    unidb-boltdb-bridge"
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

exec ./unidb-boltdb-bridge \
    -name "$BRIDGE_NAME" \
    -file "$BOLTDB_FILE" \
    -unidb "${UNIDB_URL:-http://unidb-mcp:9093}" \
    -secret "$BRIDGE_SECRET" \
    -reconnect "${RECONNECT:-true}"
