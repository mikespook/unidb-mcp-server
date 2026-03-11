#!/usr/bin/env sh
# Build the SQLite bridge Docker image with latest tag plus any extra tags.
# Usage: ./utils/docker-build-bridge.sh [EXTRA_TAGS...]
#   e.g. ./utils/docker-build-bridge.sh v1.0.0 v1.0

set -e

IMAGE="unidb-sqlite-bridge"

echo "Building SQLite bridge Docker image with latest tag..."
docker build -f docker/Dockerfile.bridge -t "mikespook/${IMAGE}:latest" .

for tag in "$@"; do
    echo "Adding tag: ${tag}"
    docker tag "mikespook/${IMAGE}:latest" "mikespook/${IMAGE}:${tag}"
done
