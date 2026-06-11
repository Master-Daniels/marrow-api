# syntax=docker/dockerfile:1

# ------------------------------------------------------------------
# Stage 1: Builder
# ------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Download dependencies first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source
COPY . .

# Build the binary
# -ldflags="-w -s" strips debug info for a smaller binary
# -trimpath removes host file system paths from stack traces
#
# NOTE: If you use a pure-Go SQLite driver, change to CGO_ENABLED=0
# and you can switch the runtime stage to gcr.io/distroless/static.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o marrow-api \
    ./cmd/marrow-api/main.go

# ------------------------------------------------------------------
# Stage 2: Production Runtime
# ------------------------------------------------------------------
FROM alpine:3.22.4



WORKDIR /app

# Install runtime libs: CA certs (HTTPS to RSSHub), SQLite libs (CGO), tzdata
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 1000 appgroup && \
    adduser -S -u 1000 -G appgroup appuser

# Create the SQLite data directory and give ownership to the non-root user.
# The named volume in docker-compose will inherit these permissions on init.
RUN mkdir -p /app/data && chown -R appuser:appgroup /app/data

# Bring in the compiled binary
COPY --from=builder --chown=appuser:appgroup /app/marrow-api .

USER appuser:appgroup

ENV APP_PORT=8080

EXPOSE 8080

# Lightweight healthcheck leveraging busybox wget
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:${APP_PORT}/health > /dev/null || exit 1

ENTRYPOINT ["./marrow-api"]