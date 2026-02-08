# Deployment Guide

This guide provides comprehensive instructions for deploying Mingyue Agent on Linux systems.

## Prerequisites

- Linux operating system (Ubuntu 20.04+, Debian 11+, CentOS 8+, or similar)
- systemd (for service management)
- Root or sudo access
- Go 1.22+ (for building from source)

### Optional Tools

- `smartctl` - For SMART disk information (install `smartmontools`)
- `blkid` - For partition information (usually pre-installed)
- `lsblk` - For disk listing (usually pre-installed)

## Installation Methods

### Method 1: Automated Installation (Recommended)

1. Build the binary:

```bash
git clone https://github.com/KOPElan/mingyue-agent.git
cd mingyue-agent
make build
```

2. Run the installation script:

```bash
sudo ./scripts/install.sh
```

The script will:
- Create a dedicated system user (`mingyue-agent`)
- Set up directories (`/etc/mingyue-agent`, `/var/log/mingyue-agent`, `/var/run/mingyue-agent`, `/var/lib/mingyue-agent`)
- Install the binary to `/usr/local/bin`
- Create a systemd service with security hardening
- Enable the service for automatic startup

3. Configure the agent:

```bash
sudo vi /etc/mingyue-agent/config.yaml
```

Important settings to configure:
- `security.allowed_paths` - Directories the agent can access
- `api.enable_http`, `api.enable_grpc`, `api.enable_uds` - Enable/disable protocols
- `audit.enabled` - Enable audit logging

4. Start the service:

```bash
sudo systemctl start mingyue-agent
sudo systemctl status mingyue-agent
```

### Method 2: Manual Installation

1. Build the binary:

```bash
make build
```

2. Create system user:

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin mingyue-agent
```

3. Create directories:

```bash
sudo mkdir -p /etc/mingyue-agent
sudo mkdir -p /var/log/mingyue-agent
sudo mkdir -p /var/run/mingyue-agent
sudo mkdir -p /var/lib/mingyue-agent
sudo chown -R mingyue-agent:mingyue-agent /var/log/mingyue-agent
sudo chown -R mingyue-agent:mingyue-agent /var/run/mingyue-agent
sudo chown -R mingyue-agent:mingyue-agent /var/lib/mingyue-agent
```

4. Install binary:

```bash
sudo install -m 755 bin/mingyue-agent /usr/local/bin/
```

5. Install configuration:

```bash
sudo cp config.example.yaml /etc/mingyue-agent/config.yaml
sudo chmod 644 /etc/mingyue-agent/config.yaml
```

6. Create systemd service file at `/etc/systemd/system/mingyue-agent.service`:

```ini
[Unit]
Description=Mingyue Agent - Local management service for home servers
Documentation=https://github.com/KOPElan/mingyue-agent
After=network.target

[Service]
Type=simple
User=mingyue-agent
Group=mingyue-agent
ExecStart=/usr/local/bin/mingyue-agent start --config /etc/mingyue-agent/config.yaml
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mingyue-agent

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/mingyue-agent /var/run/mingyue-agent /var/lib/mingyue-agent
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictNamespaces=true
LockPersonality=true

# Resource limits
LimitNOFILE=65536
LimitNPROC=512

[Install]
WantedBy=multi-user.target
```

7. Enable and start service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable mingyue-agent
sudo systemctl start mingyue-agent
```

### Method 3: Docker Deployment

Create a `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN apk add --no-cache make git && \
    make build

FROM alpine:latest

RUN apk add --no-cache ca-certificates smartmontools util-linux blkid

WORKDIR /app
COPY --from=builder /app/bin/mingyue-agent /usr/local/bin/
COPY config.example.yaml /etc/mingyue-agent/config.yaml

RUN adduser -D -s /sbin/nologin mingyue-agent && \
    mkdir -p /var/log/mingyue-agent /var/run/mingyue-agent && \
    chown -R mingyue-agent:mingyue-agent /var/log/mingyue-agent /var/run/mingyue-agent

USER mingyue-agent

EXPOSE 8080 9090

CMD ["/usr/local/bin/mingyue-agent", "start", "--config", "/etc/mingyue-agent/config.yaml"]
```

Build and run:

```bash
docker build -t mingyue-agent .
docker run -d \
  --name mingyue-agent \
  -p 8080:8080 \
  -p 9090:9090 \
  -v /etc/mingyue-agent:/etc/mingyue-agent \
  -v /var/log/mingyue-agent:/var/log/mingyue-agent \
  mingyue-agent
```

## Configuration

### Main Configuration File

Location: `/etc/mingyue-agent/config.yaml`

```yaml
server:
  listen_addr: "0.0.0.0"      # Listen on all interfaces
  http_port: 8080              # HTTP API port
  grpc_port: 9090              # gRPC API port
  uds_path: "/var/run/mingyue-agent/agent.sock"  # Unix socket

api:
  enable_http: true            # Enable HTTP API
  enable_grpc: true            # Enable gRPC API
  enable_uds: true             # Enable Unix domain socket
  tls_cert: ""                 # TLS certificate path (optional)
  tls_key: ""                  # TLS key path (optional)

audit:
  enabled: true                # Enable audit logging
  log_path: "/var/log/mingyue-agent/audit.log"
  remote_push: false           # Push to remote server
  remote_url: ""               # Remote audit server URL

security:
  enable_mtls: false           # Enable mTLS (future)
  token_auth: true             # Enable token authentication
  allowed_paths:               # Whitelist of accessible paths
    - "/home"
    - "/data"
    - "/mnt"
  max_upload_size: 10737418240 # 10GB max upload
  rate_limit_per_min: 1000     # Rate limit
  require_confirm: true        # Require confirmation for dangerous ops
```

### Security Considerations

1. **Allowed Paths**: Only add paths that the agent should access. This prevents unauthorized file access.

2. **TLS**: For production, enable TLS:
   ```yaml
   api:
     tls_cert: "/etc/mingyue-agent/certs/server.crt"
     tls_key: "/etc/mingyue-agent/certs/server.key"
   ```

3. **Firewall**: Configure firewall rules:
   ```bash
   sudo ufw allow 8080/tcp  # HTTP API
   sudo ufw allow 9090/tcp  # gRPC API
   ```

4. **Audit Logs**: Regularly review audit logs for suspicious activity:
   ```bash
   tail -f /var/log/mingyue-agent/audit.log
   ```

## Verification

After installation, verify the service is running:

```bash
# Check service status
sudo systemctl status mingyue-agent

# Check logs
sudo journalctl -u mingyue-agent -f

# Test HTTP API
curl http://localhost:8080/healthz

# Test file listing (if /tmp is in allowed_paths)
curl "http://localhost:8080/api/v1/files/list?path=/tmp"

# Test disk management
curl http://localhost:8080/api/v1/disk/partitions
```

## Monitoring

### Service Health

Monitor with systemd:

```bash
# Status
sudo systemctl status mingyue-agent

# Logs (last 100 lines)
sudo journalctl -u mingyue-agent -n 100

# Follow logs
sudo journalctl -u mingyue-agent -f
```

### Application Health

Use the health endpoint:

```bash
curl http://localhost:8080/healthz
```

Expected response:
```json
{
  "success": true,
  "data": {
    "status": "ok",
    "timestamp": "2026-02-07T12:00:00Z",
    "version": "1.0.0"
  }
}
```

### Resource Monitoring

Monitor system resources:

```bash
curl http://localhost:8080/api/v1/monitor/stats
```

## Troubleshooting

### Service Won't Start

1. Check logs:
   ```bash
   sudo journalctl -u mingyue-agent -n 50
   ```

2. Verify configuration:
   ```bash
   /usr/local/bin/mingyue-agent start --config /etc/mingyue-agent/config.yaml
   ```

3. Check permissions:
   ```bash
   sudo chown -R mingyue-agent:mingyue-agent /var/log/mingyue-agent /var/run/mingyue-agent
   ```

### API Not Responding

1. Check if service is running:
   ```bash
   sudo systemctl status mingyue-agent
   ```

2. Verify port binding:
   ```bash
   sudo netstat -tlnp | grep mingyue-agent
   ```

3. Check firewall:
   ```bash
   sudo ufw status
   ```

### Permission Denied Errors

1. Add paths to allowed_paths in config.yaml
2. Ensure mingyue-agent user has necessary permissions
3. Check audit logs for details

### High Memory Usage

1. Check for file indexing operations
2. Review configured paths (large directories cause high memory use)
3. Adjust resource limits in systemd service file

## Upgrading

### Automated Upgrade

1. Stop the service:
   ```bash
   sudo systemctl stop mingyue-agent
   ```

2. Build new version:
   ```bash
   git pull
   make build
   ```

3. Install:
   ```bash
   sudo ./scripts/install.sh
   ```

4. Start service:
   ```bash
   sudo systemctl start mingyue-agent
   ```

### Manual Upgrade

1. Stop service and backup:
   ```bash
   sudo systemctl stop mingyue-agent
   sudo cp /usr/local/bin/mingyue-agent /usr/local/bin/mingyue-agent.bak
   sudo cp /etc/mingyue-agent/config.yaml /etc/mingyue-agent/config.yaml.bak
   ```

2. Install new binary:
   ```bash
   sudo install -m 755 bin/mingyue-agent /usr/local/bin/
   ```

3. Update configuration if needed:
   ```bash
   sudo vi /etc/mingyue-agent/config.yaml
   ```

4. Restart service:
   ```bash
   sudo systemctl restart mingyue-agent
   ```

## Uninstallation

### Automated Uninstallation

```bash
sudo ./scripts/uninstall.sh
```

The script will prompt for:
- Removing configuration files
- Removing log files
- Removing system user

### Manual Uninstallation

1. Stop and disable service:
   ```bash
   sudo systemctl stop mingyue-agent
   sudo systemctl disable mingyue-agent
   ```

2. Remove service file:
   ```bash
   sudo rm /etc/systemd/system/mingyue-agent.service
   sudo systemctl daemon-reload
   ```

3. Remove binary:
   ```bash
   sudo rm /usr/local/bin/mingyue-agent
   ```

4. Remove files (optional):
   ```bash
   sudo rm -rf /etc/mingyue-agent
   sudo rm -rf /var/log/mingyue-agent
   sudo rm -rf /var/run/mingyue-agent
   ```

5. Remove user (optional):
   ```bash
   sudo userdel mingyue-agent
   ```

## Production Deployment Checklist

- [ ] Configure TLS certificates
- [ ] Set strong allowed_paths whitelist
- [ ] Enable audit logging with remote push
- [ ] Configure firewall rules
- [ ] Set up monitoring and alerting
- [ ] Configure backup for configuration files
- [ ] Review and harden systemd service file
- [ ] Test disaster recovery procedures
- [ ] Document custom configurations
- [ ] Set up log rotation
- [ ] Configure resource limits
- [ ] Enable rate limiting
- [ ] Test all API endpoints
- [ ] Review security audit logs
- [ ] Plan upgrade schedule

## Support

- Issues: https://github.com/KOPElan/mingyue-agent/issues
- Documentation: https://github.com/KOPElan/mingyue-agent/tree/main/docs
- Security: Report via GitHub Security Advisories
