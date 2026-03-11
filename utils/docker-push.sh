#!/usr/bin/env sh
# Push all tags for both Docker images to Docker Hub.
# Reads the current version from docker/.tags.
# Run 'make docker-images' (or utils/docker-images.sh) first.

set -e

BINARY="unidb-mcp-server"
TAGS_FILE="docker/.tags"

VERSION_TO_USE=$(cat "${TAGS_FILE}" 2>/dev/null) || {
    echo "Error: ${TAGS_FILE} not found. Run 'make docker-images' first." >&2
    exit 1
}

MAJ_MIN=$(echo "${VERSION_TO_USE}" | sed 's/^v//' | cut -d. -f1-2)

echo "Pushing mikespook/${BINARY} and mikespook/unidb-sqlite-bridge at ${VERSION_TO_USE}"

for img in "mikespook/${BINARY}" "mikespook/unidb-sqlite-bridge"; do
    docker push "${img}:latest"
    docker push "${img}:v${MAJ_MIN}"
    docker push "${img}:${VERSION_TO_USE}"
done

echo "Push complete."
