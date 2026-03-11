#!/usr/bin/env sh
# Build the MCP server Docker image with latest tag plus any extra tags.
# Usage: ./utils/docker-build-server.sh [EXTRA_TAGS...]
#   e.g. ./utils/docker-build-server.sh v1.0.0 v1.0

set -e

BINARY="unidb-mcp-server"

echo "Building MCP server Docker image with latest tag..."
docker build -f docker/Dockerfile -t "mikespook/${BINARY}:latest" .

for tag in "$@"; do
    echo "Adding tag: ${tag}"
    docker tag "mikespook/${BINARY}:latest" "mikespook/${BINARY}:${tag}"
done
