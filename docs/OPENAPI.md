# OpenAPI Documentation

Mingyue Agent v1.0 provides a comprehensive RESTful API for managing home server operations. This document supplements the interactive Swagger UI available at http://localhost:8080/swagger/ when the agent is running.

## Interactive API Explorer

Visit **http://localhost:8080/swagger/** for an interactive API documentation interface where you can:
- Browse all available endpoints
- View request/response schemas
- Test API calls directly from your browser
- Download the OpenAPI specification

## API Overview

The Mingyue Agent API is organized into the following modules:

### Core Services
- **Health & Status**: System health checks and agent status
- **Registration**: Agent registration with WebUI

### File Management
- **File Operations**: List, create, delete, rename, move, copy files
- **File Transfer**: Upload and download with resumption support
- **Link Operations**: Create symlinks and hardlinks

### Storage Management
- **Disk Management**: Physical disk operations, partitions, SMART monitoring
- **Network Disks**: CIFS/NFS share management
- **Share Management**: Samba/NFS share configuration

### System Management
- **Resource Monitoring**: CPU, memory, disk usage metrics
- **Network Management**: Interface configuration and monitoring

### Advanced Features
- **File Indexing**: Metadata indexing and search
- **Thumbnails**: Automatic thumbnail generation for media files
- **Task Scheduling**: Scheduled task management and execution
- **Authentication**: API tokens and session management

## Authentication

The API supports two authentication methods:

### 1. API Token Authentication
Include the token in the `X-API-Key` header:
```bash
curl -H "X-API-Key: your-token-here" \
  http://localhost:8080/api/v1/files/list?path=/data
```

### 2. User Authentication
Include the user identifier in the `X-User` header:
```bash
curl -H "X-User: admin" \
  http://localhost:8080/api/v1/files/list?path=/data
```

## API Endpoints Summary

### Health & Status
- `GET /healthz` - Health check endpoint
- `GET /api/v1/status` - Agent status
- `POST /api/v1/register` - Register with WebUI

### Resource Monitoring
- `GET /api/v1/monitor/stats` - System resource statistics
- `GET /api/v1/monitor/health` - Health status with thresholds

### File Management (12 endpoints)
- `GET /api/v1/files/list` - List files and directories
- `POST /api/v1/files/mkdir` - Create directory
- `DELETE /api/v1/files/delete` - Delete file or directory
- `POST /api/v1/files/rename` - Rename file or directory
- `POST /api/v1/files/move` - Move file or directory
- `POST /api/v1/files/copy` - Copy file or directory
- `POST /api/v1/files/upload` - Upload file
- `GET /api/v1/files/download` - Download file
- `POST /api/v1/files/symlink` - Create symbolic link
- `POST /api/v1/files/hardlink` - Create hard link
- `GET /api/v1/files/info` - Get file information
- `GET /api/v1/files/checksum` - Calculate MD5 checksum

### Disk Management (5 endpoints)
- `GET /api/v1/disk/list` - List all physical disks
- `GET /api/v1/disk/partitions` - List all partitions
- `POST /api/v1/disk/mount` - Mount a device
- `POST /api/v1/disk/unmount` - Unmount a device
- `GET /api/v1/disk/smart` - Get SMART information

### Network Disk Management (6 endpoints)
- `GET /api/v1/netdisk/shares` - List network shares
- `POST /api/v1/netdisk/shares/add` - Add network share
- `DELETE /api/v1/netdisk/shares/remove` - Remove network share
- `POST /api/v1/netdisk/mount` - Mount network share
- `POST /api/v1/netdisk/unmount` - Unmount network share
- `GET /api/v1/netdisk/status` - Get share health status

### Network Management (8 endpoints)
- `GET /api/v1/network/interfaces` - List network interfaces
- `GET /api/v1/network/interface` - Get interface details
- `POST /api/v1/network/config` - Set IP configuration
- `POST /api/v1/network/rollback` - Rollback configuration
- `GET /api/v1/network/history` - Get configuration history
- `POST /api/v1/network/enable` - Enable interface
- `POST /api/v1/network/disable` - Disable interface
- `GET /api/v1/network/ports` - List listening ports
- `GET /api/v1/network/traffic` - Get traffic statistics

### Share Management (7 endpoints)
- `GET /api/v1/shares` - List all shares
- `GET /api/v1/shares/get` - Get share details
- `POST /api/v1/shares/add` - Add new share
- `PUT /api/v1/shares/update` - Update share
- `DELETE /api/v1/shares/remove` - Remove share
- `POST /api/v1/shares/enable` - Enable share
- `POST /api/v1/shares/disable` - Disable share
- `POST /api/v1/shares/rollback` - Rollback configuration

### File Indexing (4 endpoints)
- `POST /api/v1/indexer/scan` - Scan files for indexing
- `GET /api/v1/indexer/search` - Search indexed files
- `POST /api/v1/thumbnail/generate` - Generate thumbnail
- `POST /api/v1/thumbnail/cleanup` - Cleanup thumbnail cache

### Task Scheduling (7 endpoints)
- `GET /api/v1/scheduler/tasks` - List all tasks
- `GET /api/v1/scheduler/tasks/get` - Get task details
- `POST /api/v1/scheduler/tasks/add` - Add new task
- `PUT /api/v1/scheduler/tasks/update` - Update task
- `DELETE /api/v1/scheduler/tasks/delete` - Delete task
- `POST /api/v1/scheduler/tasks/execute` - Execute task manually
- `GET /api/v1/scheduler/history` - Get execution history

### Authentication (5 endpoints)
- `POST /api/v1/auth/tokens/create` - Create API token
- `GET /api/v1/auth/tokens` - List API tokens
- `DELETE /api/v1/auth/tokens/revoke` - Revoke token
- `POST /api/v1/auth/sessions/create` - Create session
- `DELETE /api/v1/auth/sessions/revoke` - Revoke session

## Response Format

All API endpoints return JSON responses in the following format:

### Success Response
```json
{
  "success": true,
  "data": {
    // Response data here
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": "Error message here"
}
```

## Rate Limiting

Currently, there are no rate limits enforced. Rate limiting will be added in future versions.

## Versioning

The API is versioned through the URL path (`/api/v1/`). Future versions will be released as `/api/v2/`, etc., with backwards compatibility maintained for at least one major version.

## WebSocket Support

WebSocket support for real-time updates is planned for v1.2.

## Documentation Generation

To regenerate the OpenAPI documentation:

```bash
make swagger
```

This will update the Swagger spec files in the `docs/` directory.

## Further Reading

- See [API.md](API.md) for detailed endpoint documentation with examples
- See [ARCHITECTURE.md](ARCHITECTURE.md) for system design information
- See [README.md](../README.md) for general usage and setup instructions
