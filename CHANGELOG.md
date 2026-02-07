# Changelog

All notable changes to the Mingyue Agent project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Disk management module with partition detection and listing
- Mount/unmount operations with whitelist-based security
- SMART disk health monitoring via smartctl
- Deployment automation scripts (install.sh and uninstall.sh)
- Comprehensive deployment guide (docs/DEPLOYMENT.md)
- Docker support with Dockerfile and docker-compose.yml
- Disk management API documentation
- Security hardening in systemd service file
- API endpoints for disk operations:
  - GET /api/v1/disk/list - List all physical disks
  - GET /api/v1/disk/partitions - List all partitions
  - POST /api/v1/disk/mount - Mount a device
  - POST /api/v1/disk/unmount - Unmount a device
  - GET /api/v1/disk/smart - Get SMART information

### Changed
- Updated README.md with disk management features and examples
- Enhanced API.md with detailed disk management endpoints
- Updated IMPLEMENTATION.md with completed features
- Improved project documentation structure

### Security
- Implemented whitelist-based mount point access control
- Added comprehensive audit logging for disk operations
- Systemd service hardening with NoNewPrivileges, ProtectSystem, etc.
- Non-root execution with proper privilege separation

## [0.0.3] - 2026-02-07

### Added
- File management API with 12 endpoints
- System resource monitoring (CPU, memory, disk, process stats)
- Health check endpoints with degraded status detection
- Audit logging system with structured JSON logs
- Path validation and security checks
- HTTP, gRPC, and Unix domain socket support
- Configuration management with YAML
- CLI with start and version commands
- Daemon lifecycle management with graceful shutdown

### Security
- Path traversal prevention
- Null byte injection blocking
- Whitelist-based directory access
- Comprehensive audit trail for file operations

## [0.0.2] - Earlier

### Added
- Initial project structure
- Basic daemon framework
- Configuration system

## [0.0.1] - Initial

### Added
- Project initialization
- Go module setup
- Basic README and documentation
