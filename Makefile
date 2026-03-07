.PHONY: dev build build-image clean

# Binary name
BINARY := unidb-mcp-server
BUILD_DIR := build

# Go build flags
LDFLAGS := -ldflags "-s -w"

# Default target
all: build

# dev: run the server in development mode
dev:
	@echo "Starting in development mode..."
	@mkdir -p data
	@echo "Data directory created."
	@DEV_MODE=true DATA_PATH=data/config.db go run ./cmd/server

# build: compile binary and copy required files to build folder
build: clean
	@echo "Building $(BINARY)..."
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/server
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/unidb-sqlite-bridge ./cmd/sqlite-bridge
	@echo "Copying web assets..."
	cp -r web $(BUILD_DIR)/
	@echo "Creating data directory..."
	mkdir -p $(BUILD_DIR)/data
	@echo "Build complete. Output in $(BUILD_DIR)/"

# build-image: build Docker image with latest tag
# Usage: make build-image
#        make build-image EXTRA_TAGS="v1.0.0 v1.0"
build-image:
	@echo "Building Docker image with latest tag..."
	docker build -t $(BINARY):latest .
	@if [ -n "$(EXTRA_TAGS)" ]; then \
		for tag in $(EXTRA_TAGS); do \
			echo "Adding tag: $$tag"; \
			docker tag $(BINARY):latest $(BINARY):$$tag; \
		done; \
	fi

# clean: remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint the code
lint:
	golangci-lint run ./...

# Format the code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Help target
help:
	@echo "Available targets:"
	@echo "  dev            - Run server in development mode"
	@echo "  build          - Build binary and copy files to $(BUILD_DIR)/"
	@echo "  build-image    - Build Docker image with latest tag"
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy Go modules"
	@echo "  help           - Show this help message"
