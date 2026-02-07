# Changelog

All notable changes to the Mingyue Agent project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-07

### Added - Core v1.0 Features Complete! 🎉

#### OpenAPI & Documentation
- Interactive Swagger UI at `/swagger/` endpoint
- Complete OpenAPI 2.0 specification with all endpoints
- Automatic API documentation generation via `make swagger`
- Comprehensive OPENAPI.md guide
- Updated README and IMPLEMENTATION docs for v1.0

#### File Indexing & Thumbnails (GH-008)
- SQLite-based file metadata indexing with full-text search
- Incremental and full file scanning support
- MD5 hash calculation and MIME type detection
- Automatic thumbnail generation for images, videos, and PDFs
- LRU cache management with configurable size limits
- Automatic orphan cleanup for deleted files
- Search API with pagination support
- API endpoints:
  - POST /api/v1/indexer/scan - Scan files for indexing
  - GET /api/v1/indexer/search - Search indexed files
  - POST /api/v1/thumbnail/generate - Generate thumbnail
  - POST /api/v1/thumbnail/cleanup - Cleanup cache

#### Task Scheduling (GH-009)
- Cron-like task scheduling with multiple formats support
- SQLite-based task persistence and execution history
- Automatic task execution based on schedule
- Manual task execution support
- Offline tolerance with local task persistence
- Extensible task handler system
- Concurrent task execution with proper cancellation
- Comprehensive task execution history and status tracking
- API endpoints:
  - GET /api/v1/scheduler/tasks - List all tasks
  - GET /api/v1/scheduler/tasks/get - Get task details
  - POST /api/v1/scheduler/tasks/add - Add new task
  - PUT /api/v1/scheduler/tasks/update - Update task
  - DELETE /api/v1/scheduler/tasks/delete - Delete task
  - POST /api/v1/scheduler/tasks/execute - Execute task manually
  - GET /api/v1/scheduler/history - Get execution history

#### Authentication & Security (GH-010)
- Token-based API authentication with bcrypt hashing
- Session management with IP and User-Agent tracking
- Secure random token generation (32 bytes, base64-encoded)
- Configurable token and session expiration
- Token revocation support
- Constant-time string comparison for security
- SQLite-based persistent token and session storage
- API endpoints:
  - POST /api/v1/auth/tokens/create - Create API token
  - GET /api/v1/auth/tokens - List API tokens
  - DELETE /api/v1/auth/tokens/revoke - Revoke token
  - POST /api/v1/auth/sessions/create - Create session
  - DELETE /api/v1/auth/sessions/revoke - Revoke session

#### Network Disk Management (GH-004)
- CIFS/NFS share discovery and mounting
- Credential encryption using AES-256-GCM
- Whitelist-based host and mount point configuration
- Automatic health monitoring and auto-recovery
- State persistence for share configurations
- 6 API endpoints for network disk operations

#### Network Management (GH-005)
- Network interface monitoring (status, MAC, IP, traffic stats)
- IP configuration management (static/DHCP)
- Interface enable/disable operations
- Configuration history and rollback
- Listening ports monitoring
- Management interface protection
- 8 API endpoints for network operations

#### Share Management (GH-007)
- Samba and NFS share creation/modification/deletion
- User and group permission configuration
- Configuration generation with templates
- Configuration hot reload with testparm validation
- Atomic configuration rollback
- Health monitoring for shares
- 7 API endpoints for share operations

### Changed
- Updated README.md to reflect v1.0 completion (100%)
- Enhanced IMPLEMENTATION.md with all completed features
- Improved project structure documentation
- Updated Makefile with swagger target
- Reorganized documentation files

### Security
- Implemented token-based authentication system
- Added session management with security tracking
- Bcrypt password hashing for all tokens
- Secure random number generation for tokens
- Comprehensive audit logging for all auth operations
- Constant-time comparisons to prevent timing attacks

### Dependencies
- Added github.com/mattn/go-sqlite3 v1.14.33
- Added github.com/swaggo/swag v1.16.6
- Added github.com/swaggo/http-swagger v1.3.4
- Added golang.org/x/crypto v0.47.0

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
