# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o unidb-mcp-server ./backend/cmd/mcp-server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -D appuser

# Copy binary from builder
COPY --from=builder /app/unidb-mcp-server .

# Copy web assets
COPY --from=builder /app/frontend ./frontend

# Create data directory with proper ownership
RUN mkdir -p /app/data && \
    chown -R appuser:appgroup /app/data

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 9093

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
	CMD wget -q --spider http://localhost:9093/health || exit 1

# Run the application
CMD ["./unidb-mcp-server"]
