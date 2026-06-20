# Syntaxes like cache mounts require BuildKit enabled
# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Install ca-certificates in case your app makes outbound HTTPS requests
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy dependency files first to leverage Docker layer caching
COPY go.mod go.sum ./

# Use BuildKit cache mount to speed up downloading Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source code
COPY . .

# Build the binary:
# - CGO_ENABLED=0: Statically links libraries (creates a self-contained binary)
# - -ldflags="-s -w": Strips debugging information and symbols, reducing binary size by ~30-40%
# - Cache mounts: Keeps the Go build cache intact across Docker rebuilds
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o server ./cmd/server


# Stage 2: Final minimal runner image
FROM alpine:3.19 AS runner

# Create a non-root user for security (containers should not run as root)
RUN adduser -D -u 10001 appuser

WORKDIR /app

# Copy CA certificates from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/server .

# Run as the non-root user
USER appuser

EXPOSE 8080

# Use ENTRYPOINT for the primary executable behavior
ENTRYPOINT ["./server"]
