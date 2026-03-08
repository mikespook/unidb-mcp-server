.PHONY: dev dev-frontend build build-frontend build-image-server build-image-sqlite-bridge clean

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
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy Go modules"
	@echo "  help           - Show this help message"
