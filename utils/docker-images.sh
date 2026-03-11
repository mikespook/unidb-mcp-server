#!/usr/bin/env sh
# Build both Docker images with auto-versioned tags.
# Reads current version from docker/.tags, auto-increments patch, then writes back.
# Override version with: VERSION=v2.0.0 ./utils/docker-images.sh
#
# Tags applied to each image: latest, vMAJOR.MINOR, vMAJOR.MINOR.PATCH

set -e

BINARY="unidb-mcp-server"
TAGS_FILE="docker/.tags"

# Determine version to use
if [ -n "${VERSION}" ]; then
    VERSION_TO_USE="${VERSION}"
else
    CURRENT=$(cat "${TAGS_FILE}" 2>/dev/null || echo "v0.0.0")
    MAJ=$(echo "${CURRENT}" | sed 's/^v//' | cut -d. -f1)
    MIN=$(echo "${CURRENT}" | sed 's/^v//' | cut -d. -f2)
    PAT=$(echo "${CURRENT}" | sed 's/^v//' | cut -d. -f3)
    VERSION_TO_USE="v${MAJ}.${MIN}.$((PAT + 1))"
fi

MAJ_MIN=$(echo "${VERSION_TO_USE}" | sed 's/^v//' | cut -d. -f1-2)

echo "Building images: ${VERSION_TO_USE}  (tags: latest, v${MAJ_MIN}, ${VERSION_TO_USE})"

docker build -f docker/Dockerfile \
    -t "mikespook/${BINARY}:latest" \
    -t "mikespook/${BINARY}:v${MAJ_MIN}" \
    -t "mikespook/${BINARY}:${VERSION_TO_USE}" .

docker build -f docker/Dockerfile.bridge \
    -t "mikespook/unidb-sqlite-bridge:latest" \
    -t "mikespook/unidb-sqlite-bridge:v${MAJ_MIN}" \
    -t "mikespook/unidb-sqlite-bridge:${VERSION_TO_USE}" .

echo "${VERSION_TO_USE}" > "${TAGS_FILE}"
echo "Done. Version ${VERSION_TO_USE} saved to ${TAGS_FILE}"
