.PHONY: dev dev-frontend build build-frontend docker-build-images docker-push-images clean

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
	CGO_ENABLED=0 go build -C backend $(LDFLAGS) -o ../$(BUILD_DIR)/unidb-boltdb-bridge ./cmd/boltdb-bridge
	@echo "Copying frontend dist..."
	mkdir -p $(BUILD_DIR)/frontend
	cp -r frontend/dist $(BUILD_DIR)/frontend/dist
	@echo "Creating data directory..."
	mkdir -p $(BUILD_DIR)/data
	@echo "Build complete. Output in $(BUILD_DIR)/"

# docker-build-images: build both Docker images with auto-versioned tags from docker/.tags
# Version is auto-incremented (patch) each run, or override with VERSION=vX.Y.Z
# Tags applied: latest, vX.Y, vX.Y.Z
# Usage: make docker-build-images
#        make docker-build-images VERSION=v2.0.0
build-docker-images:
	@VERSION=$(VERSION) utils/docker-build-images.sh

# docker-push-images: push all tags for both images to Docker Hub
# Reads version from docker/.tags (run 'make docker-build-images' first)
push-docker-images:
	@utils/docker-push.sh

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
	@echo "  docker-build-images - Build both Docker images (auto-increment patch, or VERSION=vX.Y.Z)"
	@echo "  docker-push-images  - Push all tags for both images to Docker Hub"
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy Go modules"
	@echo "  help           - Show this help message"
