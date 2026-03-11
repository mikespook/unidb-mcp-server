.PHONY: dev dev-frontend build build-frontend build-image-server build-image-sqlite-bridge docker-images docker-push clean

# Binary name
BINARY := unidb-mcp-server
BUILD_DIR := build

# Go build flags
LDFLAGS := -ldflags "-s -w"

# Default target
all: build

# dev: run the backend server in development mode
dev:
	@echo "Starting in development mode..."
	@mkdir -p backend/data
	@echo "Data directory created."
	@DEV_MODE=true DATA_PATH=data/config.db go run -C backend ./cmd/mcp-server

# dev-frontend: run the Vite dev server (proxy to backend at :9093)
dev-frontend:
	@echo "Starting Vite dev server..."
	npm --prefix frontend run dev

# build-frontend: install deps and build Vue/Vite frontend
build-frontend:
	@echo "Building frontend..."
	npm --prefix frontend install
	npm --prefix frontend run build

# build: compile binaries and assemble output in build folder
build: clean build-frontend
	@echo "Building $(BINARY)..."
	CGO_ENABLED=1 go build -C backend $(LDFLAGS) -o ../$(BUILD_DIR)/$(BINARY) ./cmd/mcp-server
	CGO_ENABLED=1 go build -C backend $(LDFLAGS) -o ../$(BUILD_DIR)/unidb-sqlite-bridge ./cmd/sqlite-bridge
	@echo "Copying frontend dist..."
	mkdir -p $(BUILD_DIR)/frontend
	cp -r frontend/dist $(BUILD_DIR)/frontend/dist
	@echo "Creating data directory..."
	mkdir -p $(BUILD_DIR)/data
	@echo "Build complete. Output in $(BUILD_DIR)/"

# build-image-server: build Docker image for the MCP server with latest tag
# Usage: make build-image-server
#        make build-image-server EXTRA_TAGS="v1.0.0 v1.0"
build-image-server:
	@echo "Building MCP server Docker image with latest tag..."
	docker build -f docker/Dockerfile -t mikespook/$(BINARY):latest .
	@if [ -n "$(EXTRA_TAGS)" ]; then \
		for tag in $(EXTRA_TAGS); do \
			echo "Adding tag: $$tag"; \
			docker tag mikespook/$(BINARY):latest mikespook/$(BINARY):$$tag; \
		done; \
	fi

# build-image-sqlite-bridge: build Docker image for the SQLite bridge with latest tag
# Usage: make build-image-sqlite-bridge
#        make build-image-sqlite-bridge EXTRA_TAGS="v1.0.0 v1.0"
build-image-sqlite-bridge:
	@echo "Building SQLite bridge Docker image with latest tag..."
	docker build -f docker/Dockerfile.bridge -t mikespook/unidb-sqlite-bridge:latest .
	@if [ -n "$(EXTRA_TAGS)" ]; then \
		for tag in $(EXTRA_TAGS); do \
			echo "Adding tag: $$tag"; \
			docker tag mikespook/unidb-sqlite-bridge:latest mikespook/unidb-sqlite-bridge:$$tag; \
		done; \
	fi

# docker-images: build both Docker images with auto-versioned tags from docker/.tags
# Version is auto-incremented (patch) each run, or override with VERSION=vX.Y.Z
# Tags applied: latest, vX.Y, vX.Y.Z
# Usage: make docker-images
#        make docker-images VERSION=v2.0.0
docker-images:
	@VERSION_TO_USE=$$(if [ -n "$(VERSION)" ]; then \
	    echo "$(VERSION)"; \
	  else \
	    CURRENT=$$(cat docker/.tags 2>/dev/null || echo "v0.0.0"); \
	    MAJ=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f1); \
	    MIN=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f2); \
	    PAT=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f3); \
	    echo "v$$MAJ.$$MIN.$$(( PAT + 1 ))"; \
	  fi); \
	  MAJ_MIN=$$(echo $$VERSION_TO_USE | sed 's/^v//' | cut -d. -f1-2); \
	  echo "Building images: $$VERSION_TO_USE  (tags: latest, v$$MAJ_MIN, $$VERSION_TO_USE)"; \
	  docker build -f docker/Dockerfile \
	    -t mikespook/$(BINARY):latest \
	    -t mikespook/$(BINARY):v$$MAJ_MIN \
	    -t mikespook/$(BINARY):$$VERSION_TO_USE .; \
	  docker build -f docker/Dockerfile.bridge \
	    -t mikespook/unidb-sqlite-bridge:latest \
	    -t mikespook/unidb-sqlite-bridge:v$$MAJ_MIN \
	    -t mikespook/unidb-sqlite-bridge:$$VERSION_TO_USE .; \
	  echo $$VERSION_TO_USE > docker/.tags; \
	  echo "Done. Version $$VERSION_TO_USE saved to docker/.tags"

# docker-push: push all tags for both images to Docker Hub
# Reads version from docker/.tags (run 'make docker-images' first)
docker-push:
	@VERSION_TO_USE=$$(cat docker/.tags 2>/dev/null || { echo "Error: docker/.tags not found. Run 'make docker-images' first." >&2; exit 1; }); \
	  MAJ_MIN=$$(echo $$VERSION_TO_USE | sed 's/^v//' | cut -d. -f1-2); \
	  echo "Pushing mikespook/$(BINARY) and mikespook/unidb-sqlite-bridge at $$VERSION_TO_USE"; \
	  for img in mikespook/$(BINARY) mikespook/unidb-sqlite-bridge; do \
	    docker push $$img:latest; \
	    docker push $$img:v$$MAJ_MIN; \
	    docker push $$img:$$VERSION_TO_USE; \
	  done; \
	  echo "Push complete."

# clean: remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY)

# Run tests
test:
	go test -C backend -v ./...

# Run tests with coverage
test-coverage:
	go test -C backend -v -coverprofile=../coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint the code
lint:
	golangci-lint run ./backend/...

# Format the code
fmt:
	go fmt -C backend ./...

# Tidy dependencies
tidy:
	go mod -C backend tidy

# Help target
help:
	@echo "Available targets:"
	@echo "  dev            - Run backend server in development mode"
	@echo "  dev-frontend   - Run Vite dev server (proxy to :9093)"
	@echo "  build-frontend - Build Vue/Vite frontend"
	@echo "  build          - Build binary and copy files to $(BUILD_DIR)/"
	@echo "  build-image-server        - Build MCP server Docker image with latest tag"
	@echo "  build-image-sqlite-bridge - Build SQLite bridge Docker image with latest tag"
	@echo "  docker-images  - Build both Docker images (auto-increment patch, or VERSION=vX.Y.Z)"
	@echo "  docker-push    - Push all tags for both images to Docker Hub"
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy Go modules"
	@echo "  help           - Show this help message"
