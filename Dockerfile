# Multi-stage Dockerfile for Ditto
# Builds tiny production binary

# ─────────────────────────────────────────────────────────────
# Stage 1: Build
# ─────────────────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Build optimized binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w \
      -X main.version=${VERSION} \
      -X main.commit=${COMMIT} \
      -X main.date=${DATE}" \
    -o /build/Ditto \
    ./cmd/ditto

# ─────────────────────────────────────────────────────────────
# Stage 2: Runtime (minimal image)
# ─────────────────────────────────────────────────────────────
FROM scratch AS runtime

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /build/Ditto /Ditto

# Set working directory
WORKDIR /data

# Default command
ENTRYPOINT ["/Ditto"]
CMD ["help"]

# ─────────────────────────────────────────────────────────────
# Stage 3: Development (with examples)
# ─────────────────────────────────────────────────────────────
FROM golang:1.21-alpine AS development

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build with debug symbols
RUN go build -race -o /app/Ditto ./cmd/ditto

WORKDIR /workspace

ENTRYPOINT ["/app/Ditto"]
