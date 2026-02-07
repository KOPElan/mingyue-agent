# Mingyue Agent

[![CI](https://github.com/KOPElan/mingyue-agent/workflows/CI/badge.svg)](https://github.com/KOPElan/mingyue-agent/actions)
[![Release](https://github.com/KOPElan/mingyue-agent/workflows/Release/badge.svg)](https://github.com/KOPElan/mingyue-agent/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/KOPElan/mingyue-agent)](https://goreportcard.com/report/github.com/KOPElan/mingyue-agent)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Mingyue Agent is the core local management service for the Mingyue Portal home server ecosystem, providing both remote collaboration agent and local privileged operations capabilities.

## ✨ Features

### 🚀 Core Infrastructure (Implemented)
- **Daemon Lifecycle**: CLI-based daemon with graceful shutdown and signal handling
- **Multi-Protocol APIs**: HTTP (8080), gRPC (9090), Unix domain socket
- **Configuration Management**: YAML-based with validation and defaults
- **Audit Logging**: Structured JSON logs with local storage and remote push

### 📁 Secure File Management (Implemented)
- **Path Validation**: Prevents traversal attacks, enforces whitelist, blocks null bytes
- **File Operations**: List, create, delete, rename, move, copy with full metadata
- **Transfer Support**: Upload with size limits, download with HTTP range/resumption
- **Link Operations**: Symlink and hardlink creation
- **Complete API**: 12 RESTful endpoints with comprehensive audit trail

### 💾 Disk Management (Implemented)
- **Partition Management**: Auto-detection, listing with detailed metadata (UUID, label, usage)
- **Mount Operations**: Secure mount/unmount with whitelist-based access control
- **SMART Monitoring**: Disk health status, temperature, power-on hours via smartctl
- **Disk Information**: Physical disk detection, partition mapping, filesystem details
- **Safety Features**: Allowed mount points whitelist, comprehensive audit logging

### 📊 Resource Monitoring (Implemented)
- **System Metrics**: CPU (cores, load avg), memory (RAM/swap), disk usage, process stats
- **Health Checks**: `/healthz` with degraded status on resource thresholds
- **Monitoring APIs**: Detailed stats and health status endpoints

### 🔮 Implemented Features (v1.0 Complete!)
- ✅ **Network Disk Management**: CIFS/NFS mounting, credential encryption, auto-recovery
- ✅ **Network Management**: Interface monitoring, IP configuration, traffic stats
- ✅ **Share Management**: Samba/NFS share configuration and management
- ✅ **Indexing & Thumbnails**: Media file indexing and thumbnail generation
- ✅ **Task Scheduling**: Distributed task orchestration with offline tolerance
- ✅ **Security Hardening**: Token-based authentication, session management, audit logging
- ✅ **OpenAPI Documentation**: Interactive Swagger UI for API exploration

### 🔮 Planned Future Features (v1.1+)
- **v1.1**: Enhanced mTLS authentication, privilege separation
- **v1.2**: Client library, advanced CLI features
- **v1.3**: Full-text search, video transcoding, distributed execution
- **v2.0**: Plugin system, metrics export, advanced security features

## 🚀 Quick Start

### Prerequisites

- Go 1.22 or higher
- Linux operating system (primary target)
- Make (optional, for build automation)

### Installation

#### From Binary Release

Download the latest binary from [Releases](https://github.com/KOPElan/mingyue-agent/releases):

```bash
# Linux AMD64
wget https://github.com/KOPElan/mingyue-agent/releases/download/v1.0.0/mingyue-agent-v1.0.0-linux-amd64.tar.gz
tar -xzf mingyue-agent-v1.0.0-linux-amd64.tar.gz
```

#### From Source

```bash
# Clone repository
git clone https://github.com/KOPElan/mingyue-agent.git
cd mingyue-agent

# Build
make build

# Or with Go directly
go build -o bin/mingyue-agent ./cmd/agent
```

### Configuration

1. Copy the example configuration:

```bash
cp config.example.yaml config.yaml
```

2. Edit `config.yaml` to customize settings:

```yaml
server:
  listen_addr: "0.0.0.0"
  http_port: 8080
  grpc_port: 9090

security:
  allowed_paths:
    - "/home"
    - "/data"
  max_upload_size: 10737418240  # 10GB

audit:
  enabled: true
  log_path: "/var/log/mingyue-agent/audit.log"
```

### Running

```bash
# Start the agent
./bin/mingyue-agent start --config config.yaml

# Or use the default config location
./bin/mingyue-agent start

# Check version
./bin/mingyue-agent version

# Get help
./bin/mingyue-agent --help
```

## 📖 Documentation

- **[API Documentation](docs/API.md)**: Complete API reference with examples
- **[OpenAPI/Swagger](http://localhost:8080/swagger/)**: Interactive API documentation (when agent is running)
- **[Architecture Guide](docs/ARCHITECTURE.md)**: Technical architecture and design
- **[Implementation Progress](IMPLEMENTATION.md)**: Current status and roadmap
- **[Deployment Guide](docs/DEPLOYMENT.md)**: Installation and deployment instructions

## 🔌 API Usage Examples

### Interactive API Documentation

Once the agent is running, visit **http://localhost:8080/swagger/** for interactive API documentation powered by Swagger UI. You can explore all endpoints, view request/response schemas, and test API calls directly from your browser.

### Health Check

```bash
curl http://localhost:8080/healthz
```

Response:
```json
{
  "success": true,
  "data": {
    "status": "ok",
    "timestamp": "2026-02-07T10:00:00Z",
    "version": "1.0.0"
  }
}
```

### System Monitoring

```bash
curl http://localhost:8080/api/v1/monitor/stats
```

Response:
```json
{
  "success": true,
  "data": {
    "cpu": {
      "cores": 8,
      "load_avg_1": 2.5
    },
    "memory": {
      "total": 16777216000,
      "used_percent": 50.0
    },
    "disk": {
      "total": 1099511627776,
      "used_percent": 50.0
    }
  }
}
```

### File Operations

```bash
# List files
curl "http://localhost:8080/api/v1/files/list?path=/tmp"

# Create directory
curl -X POST -H "Content-Type: application/json" \
  -d '{"path":"/tmp/newdir"}' \
  http://localhost:8080/api/v1/files/mkdir

# Upload file
curl -X POST --data-binary @file.txt \
  "http://localhost:8080/api/v1/files/upload?path=/tmp/uploaded.txt"

# Download file
curl "http://localhost:8080/api/v1/files/download?path=/tmp/file.txt" \
  -o downloaded.txt
```

### Disk Management

```bash
# List all disks
curl http://localhost:8080/api/v1/disk/list

# List partitions
curl http://localhost:8080/api/v1/disk/partitions

# Get SMART info
curl "http://localhost:8080/api/v1/disk/smart?device=/dev/sda"

# Mount a device
curl -X POST -H "Content-Type: application/json" \
  -d '{"device":"/dev/sdb1","mount_point":"/mnt/data","filesystem":"ext4"}' \
  http://localhost:8080/api/v1/disk/mount

# Unmount a device
curl -X POST -H "Content-Type: application/json" \
  -d '{"target":"/mnt/data","force":false}' \
  http://localhost:8080/api/v1/disk/unmount
```

### Task Scheduling

```bash
# Add a new scheduled task
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"Daily Cleanup","type":"cleanup","schedule":"daily","enabled":true}' \
  http://localhost:8080/api/v1/scheduler/tasks/add

# List all tasks
curl http://localhost:8080/api/v1/scheduler/tasks

# Execute a task manually
curl -X POST "http://localhost:8080/api/v1/scheduler/tasks/execute?id=task-123"
```

### File Indexing and Thumbnails

```bash
# Scan files for indexing
curl -X POST -H "Content-Type: application/json" \
  -d '{"paths":["/data"],"recursive":true,"incremental":true}' \
  http://localhost:8080/api/v1/indexer/scan

# Search indexed files
curl "http://localhost:8080/api/v1/indexer/search?q=photo&limit=10"

# Generate thumbnail
curl -X POST "http://localhost:8080/api/v1/thumbnail/generate?path=/data/image.jpg"
```

### Authentication

```bash
# Create API token
curl -X POST -H "Content-Type: application/json" \
  -d '{"user_id":"admin","name":"my-token","expires_in":31536000}' \
  http://localhost:8080/api/v1/auth/tokens/create

# Use token in requests
curl -H "X-API-Key: your-token-here" \
  http://localhost:8080/api/v1/files/list?path=/data
```

See [API Documentation](docs/API.md) for complete endpoint reference.

## 🏗️ Project Structure

```
.
├── cmd/
│   └── agent/              # Main application entry point
├── internal/
│   ├── api/                # HTTP/gRPC API handlers
│   ├── audit/              # Audit logging system
│   ├── config/             # Configuration management
│   ├── daemon/             # Daemon lifecycle management
│   ├── filemanager/        # File operations with security
│   ├── monitor/            # System resource monitoring
│   └── server/             # Multi-protocol server framework
├── docs/                   # Documentation
│   ├── API.md             # API reference
│   └── ARCHITECTURE.md    # Technical architecture
├── config.example.yaml     # Example configuration
├── Makefile               # Build automation
└── README.md              # This file
```

## 🛠️ Development

### Build

```bash
make build

# Generate OpenAPI/Swagger documentation
make swagger
```

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Lint

```bash
make lint
```

### Clean Build Artifacts

```bash
make clean
```

## 🔒 Security

### Security Features

- **Path Validation**: All file operations validate paths to prevent traversal attacks
- **Whitelist Access**: File operations restricted to configured allowed directories
- **Audit Trail**: Complete logging of all privileged operations
- **Input Validation**: Strict validation at all API boundaries
- **No Command Injection**: Type-safe operations, no shell command execution

### Security Principles

1. **Least Privilege**: Main process runs as non-root (privilege elevation planned)
2. **Whitelist Approach**: Deny by default, explicit allow lists
3. **Comprehensive Auditing**: All sensitive operations logged with context
4. **Input Sanitization**: Strict validation and sanitization of all inputs

### Reporting Security Issues

Please report security vulnerabilities via GitHub Security Advisories or by emailing the maintainers directly. Do not open public issues for security concerns.

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines and code of conduct before submitting pull requests.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🗺️ Roadmap

### Current Status (v1.0 - 100% Complete ✅)

- ✅ Multi-protocol API support (HTTP, gRPC, UDS)
- ✅ Secure file management with validation
- ✅ System resource monitoring
- ✅ Disk management with SMART support
- ✅ Network disk management (CIFS/NFS)
- ✅ Network interface management
- ✅ Share management (Samba/NFS)
- ✅ File indexing and thumbnail generation
- ✅ Task scheduling and orchestration
- ✅ Token-based authentication and sessions
- ✅ Audit logging system
- ✅ OpenAPI/Swagger documentation
- ✅ Deployment automation scripts

### Next Milestones

- **v1.1**: Enhanced mTLS authentication, privilege separation
- **v1.2**: Go client library, advanced CLI features
- **v1.3**: Full-text search, video transcoding
- **v1.4**: Distributed task execution
- **v1.5**: Plugin system for extensibility
- **v2.0**: Advanced security features, metrics export

See [IMPLEMENTATION.md](IMPLEMENTATION.md) for detailed progress tracking.

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/KOPElan/mingyue-agent/issues)
- **Discussions**: [GitHub Discussions](https://github.com/KOPElan/mingyue-agent/discussions)
- **Documentation**: [docs/](docs/)

## 🙏 Acknowledgments

Built with:
- [Go](https://golang.org/) - Programming language
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [gRPC](https://grpc.io/) - RPC framework
- [YAML](https://yaml.org/) - Configuration format

## 📊 Status

This project is under active development. The current focus is on implementing core management features as outlined in the [PRD](prd.md) and [Requirements](docs/agent-需求说明书.md).
