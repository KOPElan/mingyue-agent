FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache make git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN make build

# Runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    smartmontools \
    util-linux \
    blkid \
    e2fsprogs \
    xfsprogs \
    nfs-utils \
    cifs-utils

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/mingyue-agent /usr/local/bin/

# Copy default config
COPY config.example.yaml /etc/mingyue-agent/config.yaml

# Create non-root user
RUN adduser -D -s /sbin/nologin mingyue-agent && \
    mkdir -p /var/log/mingyue-agent /var/run/mingyue-agent /var/lib/mingyue-agent /mnt/data && \
    chown -R mingyue-agent:mingyue-agent /var/log/mingyue-agent /var/run/mingyue-agent /var/lib/mingyue-agent

# Switch to non-root user
USER mingyue-agent

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run agent
CMD ["/usr/local/bin/mingyue-agent", "start", "--config", "/etc/mingyue-agent/config.yaml"]
