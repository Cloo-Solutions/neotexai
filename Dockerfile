# Build stage
FROM golang:1.25-alpine3.23 AS builder

# Install git for go mod download (some deps may need it)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /neotexd ./cmd/neotexd

# Runtime stage
FROM alpine:3.23

# Install ca-certificates for HTTPS and tzdata for timezones
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' neotex
USER neotex

WORKDIR /app

# Copy binary from builder
COPY --from=builder /neotexd /app/neotexd

# Copy migrations (needed for auto-migrate)
COPY --from=builder /app/migrations /app/migrations

# Expose default port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["/app/neotexd"]
CMD ["serve"]
