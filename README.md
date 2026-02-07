# Mingyue Agent

Mingyue Agent is the core local management service for the Mingyue Portal home server ecosystem, providing both remote collaboration agent and local privileged operations capabilities.

## Features

- **Startup & Registration**: CLI-based daemon with automatic WebUI registration
- **File Management**: Secure file operations with audit logging
- **Disk Management**: Partition detection, mounting, SMART monitoring
- **Network Disk Management**: CIFS/NFS share management
- **System Network Management**: Network interface monitoring and configuration
- **Resource Monitoring**: CPU, memory, disk, and process monitoring
- **Share Management**: Samba/NFS share configuration
- **Indexing & Thumbnails**: Media file indexing and thumbnail generation
- **Scheduled Tasks**: Task orchestration and execution
- **Security & Audit**: Comprehensive logging and authentication

## Quick Start

### Build

```bash
make build
```

### Run

```bash
./bin/mingyue-agent start
```

Or use the Makefile:

```bash
make run
```

### Configuration

Copy the example configuration:

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` to customize settings.

### Install

```bash
make install
```

## Development

### Prerequisites

- Go 1.22 or higher
- Make

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Formatting

```bash
make fmt
```

### Linting

```bash
make lint
```

## Project Structure

```
.
├── cmd/
│   └── agent/          # Main application entry point
├── internal/
│   ├── api/            # API handlers
│   ├── audit/          # Audit logging
│   ├── config/         # Configuration management
│   ├── daemon/         # Daemon process management
│   └── server/         # HTTP/gRPC server
├── pkg/
│   └── client/         # Client library (future)
├── docs/               # Documentation
└── Makefile            # Build automation
```

## API Endpoints

### Health Check

```
GET /healthz
```

### Registration

```
POST /api/v1/register
```

### Status

```
GET /api/v1/status
```

## License

See LICENSE file for details.
