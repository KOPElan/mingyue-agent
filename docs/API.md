# Mingyue Agent API Documentation

## Overview

Mingyue Agent provides RESTful HTTP APIs for managing home server resources. All APIs return JSON responses and support standard HTTP methods.

**Base URL:** `http://localhost:8080`

**Authentication:** Currently supports header-based user identification via `X-User` header (token authentication coming soon).

## Response Format

All API responses follow this structure:

```json
{
  "success": true,
  "data": { ... },
  "error": ""
}
```

- `success`: Boolean indicating if the request succeeded
- `data`: Response payload (varies by endpoint)
- `error`: Error message if `success` is false

## Health & Status APIs

### GET /healthz

Health check endpoint for monitoring system health.

**Response:**
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

**Status Codes:**
- `200 OK`: System is healthy
- `503 Service Unavailable`: System is degraded (memory >95% or disk >98%)

### GET /api/v1/status

Get agent status information.

**Response:**
```json
{
  "success": true,
  "data": {
    "hostname": "localhost",
    "uptime": 3600.5,
    "status": "running"
  }
}
```

### POST /api/v1/register

Register agent with WebUI control center.

**Response:**
```json
{
  "success": true,
  "data": {
    "agent_id": "agent-hostname-1234567890",
    "hostname": "hostname",
    "version": "1.0.0",
    "start_time": "2026-02-07T10:00:00Z",
    "api_urls": ["http://localhost:8080/api/v1"]
  }
}
```

## Monitoring APIs

### GET /api/v1/monitor/stats

Get comprehensive system resource statistics.

**Response:**
```json
{
  "success": true,
  "data": {
    "cpu": {
      "cores": 8,
      "usage_percent": 45.2,
      "load_avg_1": 2.5,
      "load_avg_5": 2.1,
      "load_avg_15": 1.8
    },
    "memory": {
      "total": 16777216000,
      "available": 8388608000,
      "used": 8388608000,
      "used_percent": 50.0,
      "swap_total": 4194304000,
      "swap_used": 0
    },
    "disk": {
      "total": 1099511627776,
      "free": 549755813888,
      "used": 549755813888,
      "used_percent": 50.0
    },
    "process": {
      "pid": 12345,
      "goroutines": 25,
      "mem_alloc": 12582912,
      "mem_sys": 75497472,
      "num_gc": 15,
      "open_files": 18
    },
    "uptime": 3600.5
  }
}
```

**Fields:**
- `cpu.cores`: Number of CPU cores
- `cpu.load_avg_*`: System load averages (1, 5, 15 minutes)
- `memory.*`: Memory usage in bytes
- `disk.*`: Root filesystem disk usage in bytes
- `process.*`: Current process statistics
- `uptime`: Agent uptime in seconds

### GET /api/v1/monitor/health

Get health status with detailed information.

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "healthy": true,
    "timestamp": "2026-02-07T10:00:00Z"
  }
}
```

**Health Status:**
- `healthy`: All resources within normal thresholds
- `unhealthy`: Memory >95% or disk >98%

## File Management APIs

### GET /api/v1/files/list

List files and directories in a path.

**Query Parameters:**
- `path` (required): Directory path to list

**Example:**
```bash
curl "http://localhost:8080/api/v1/files/list?path=/tmp"
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "name": "example.txt",
      "path": "/tmp/example.txt",
      "size": 1024,
      "mode": 420,
      "mod_time": "2026-02-07T10:00:00Z",
      "is_dir": false,
      "is_symlink": false,
      "owner": 1000,
      "group": 1000,
      "permissions": "-rw-r--r--"
    }
  ]
}
```

### GET /api/v1/files/info

Get detailed information about a file or directory.

**Query Parameters:**
- `path` (required): File or directory path

**Example:**
```bash
curl "http://localhost:8080/api/v1/files/info?path=/tmp/example.txt"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "name": "example.txt",
    "path": "/tmp/example.txt",
    "size": 1024,
    "mode": 420,
    "mod_time": "2026-02-07T10:00:00Z",
    "is_dir": false,
    "is_symlink": false,
    "owner": 1000,
    "group": 1000,
    "permissions": "-rw-r--r--"
  }
}
```

### POST /api/v1/files/mkdir

Create a new directory.

**Request Body:**
```json
{
  "path": "/tmp/newdir"
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"path":"/tmp/newdir"}' \
  http://localhost:8080/api/v1/files/mkdir
```

**Response:**
```json
{
  "success": true
}
```

### POST /api/v1/files/delete

Delete a file or directory.

**Request Body:**
```json
{
  "path": "/tmp/example.txt"
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"path":"/tmp/example.txt"}' \
  http://localhost:8080/api/v1/files/delete
```

**Response:**
```json
{
  "success": true
}
```

### POST /api/v1/files/rename

Rename or move a file or directory.

**Request Body:**
```json
{
  "old_path": "/tmp/old.txt",
  "new_path": "/tmp/new.txt"
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"old_path":"/tmp/old.txt","new_path":"/tmp/new.txt"}' \
  http://localhost:8080/api/v1/files/rename
```

**Response:**
```json
{
  "success": true
}
```

### POST /api/v1/files/copy

Copy a file.

**Request Body:**
```json
{
  "src_path": "/tmp/source.txt",
  "dst_path": "/tmp/destination.txt"
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"src_path":"/tmp/source.txt","dst_path":"/tmp/destination.txt"}' \
  http://localhost:8080/api/v1/files/copy
```

**Response:**
```json
{
  "success": true
}
```

### POST /api/v1/files/move

Move a file or directory.

**Request Body:**
```json
{
  "src_path": "/tmp/source.txt",
  "dst_path": "/tmp/destination.txt"
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"src_path":"/tmp/source.txt","dst_path":"/tmp/destination.txt"}' \
  http://localhost:8080/api/v1/files/move
```

**Response:**
```json
{
  "success": true
}
```

### POST /api/v1/files/upload

Upload a file to the server.

**Query Parameters:**
- `path` (required): Destination path for the uploaded file
- `max_size` (optional): Maximum file size in bytes

**Example:**
```bash
curl -X POST --data-binary @localfile.txt \
  "http://localhost:8080/api/v1/files/upload?path=/tmp/uploaded.txt"
```

**Response:**
```json
{
  "success": true
}
```

### GET /api/v1/files/download

Download a file from the server.

**Query Parameters:**
- `path` (required): File path to download

**Headers:**
- `Range`: Optional HTTP range header for partial/resumable downloads

**Example:**
```bash
curl "http://localhost:8080/api/v1/files/download?path=/tmp/file.txt" \
  -o downloaded.txt
```

**Example with range:**
```bash
curl -H "Range: bytes=0-1023" \
  "http://localhost:8080/api/v1/files/download?path=/tmp/file.txt" \
  -o partial.txt
```

**Response:**
- `200 OK`: Full file download
- `206 Partial Content`: Range download
- File content as binary stream

### POST /api/v1/files/symlink

Create a symbolic link.

**Request Body:**
```json
{
  "target": "/path/to/target",
  "link_path": "/path/to/link"
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"target":"/tmp/target","link_path":"/tmp/link"}' \
  http://localhost:8080/api/v1/files/symlink
```

**Response:**
```json
{
  "success": true
}
```

### POST /api/v1/files/hardlink

Create a hard link.

**Request Body:**
```json
{
  "target": "/path/to/target",
  "link_path": "/path/to/link"
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"target":"/tmp/target","link_path":"/tmp/link"}' \
  http://localhost:8080/api/v1/files/hardlink
```

**Response:**
```json
{
  "success": true
}
```

### GET /api/v1/files/checksum

Calculate MD5 checksum of a file.

**Query Parameters:**
- `path` (required): File path

**Example:**
```bash
curl "http://localhost:8080/api/v1/files/checksum?path=/tmp/file.txt"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "checksum": "5d41402abc4b2a76b9719d911017c592"
  }
}
```

## Security

### Path Validation

All file operations enforce strict path validation:
- Paths must be absolute
- No path traversal (`..`) allowed
- No null bytes in paths
- Paths must be within configured `allowed_paths` whitelist

### Audit Logging

All file operations and privileged actions are logged to the audit log with:
- Timestamp
- User identifier
- Action performed
- Resource accessed
- Operation result
- Additional context details

Audit logs are stored in JSON format at the configured `audit.log_path`.

## Error Handling

**Common Error Responses:**

```json
{
  "success": false,
  "error": "invalid path: path traversal detected"
}
```

**HTTP Status Codes:**
- `200 OK`: Request successful
- `400 Bad Request`: Invalid request parameters
- `405 Method Not Allowed`: Incorrect HTTP method
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service degraded

## Configuration

Configure API settings in `config.yaml`:

```yaml
server:
  listen_addr: "0.0.0.0"
  http_port: 8080
  grpc_port: 9090
  uds_path: "/var/run/mingyue-agent/agent.sock"

security:
  allowed_paths:
    - "/home"
    - "/data"
  max_upload_size: 10737418240  # 10GB
  rate_limit_per_min: 1000
```

## Rate Limiting

API requests are subject to rate limiting based on configuration:
- Default: 1000 requests per minute per client
- Configurable via `security.rate_limit_per_min`

## Disk Management APIs

### GET /api/v1/disk/list

Lists all physical disks with partitions and metadata.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "device": "/dev/sda",
      "model": "Samsung SSD 860",
      "size": 1000204886016,
      "partitions": [
        {
          "name": "sda1",
          "device": "/dev/sda1",
          "mount_point": "/",
          "filesystem": "ext4",
          "size": 1000204886016,
          "used": 450000000000,
          "available": 550000000000,
          "used_percent": 45.0,
          "uuid": "1234-5678-90AB-CDEF",
          "label": "root",
          "read_only": false
        }
      ]
    }
  ]
}
```

**Audit Log:** `disk.list`

---

### GET /api/v1/disk/partitions

Lists all mounted partitions with usage statistics.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "name": "sda1",
      "device": "/dev/sda1",
      "mount_point": "/",
      "filesystem": "ext4",
      "size": 1000204886016,
      "used": 450000000000,
      "available": 550000000000,
      "used_percent": 45.0,
      "uuid": "1234-5678-90AB-CDEF",
      "label": "root",
      "read_only": false
    }
  ]
}
```

**Field Descriptions:**
- `name`: Partition name (e.g., sda1)
- `device`: Full device path
- `mount_point`: Where the partition is mounted
- `filesystem`: Filesystem type (ext4, xfs, ntfs, etc.)
- `size`: Total size in bytes
- `used`: Used space in bytes
- `available`: Available space in bytes
- `used_percent`: Usage percentage
- `uuid`: Partition UUID
- `label`: Partition label/name
- `read_only`: Whether mounted as read-only

**Audit Log:** `disk.list_partitions`

---

### POST /api/v1/disk/mount

Mounts a device to a specified mount point.

**Request Body:**
```json
{
  "device": "/dev/sdb1",
  "mount_point": "/mnt/data",
  "filesystem": "ext4",
  "options": ["rw", "noexec"],
  "read_only": false
}
```

**Parameters:**
- `device` (required): Device path to mount
- `mount_point` (required): Target mount point
- `filesystem` (optional): Filesystem type (auto-detected if omitted)
- `options` (optional): Mount options array
- `read_only` (optional): Mount as read-only

**Security:**
- Mount point must be in `security.allowed_paths` whitelist
- Creates mount point directory if it doesn't exist
- Validates device exists

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "device mounted successfully"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Missing required parameters
- `403 Forbidden`: Mount point not in allowed list
- `500 Internal Server Error`: Mount operation failed

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"device":"/dev/sdb1","mount_point":"/mnt/backup","filesystem":"ext4"}' \
  http://localhost:8080/api/v1/disk/mount
```

**Audit Log:** `disk.mount` with device and mount point details

---

### POST /api/v1/disk/unmount

Unmounts a device or mount point.

**Request Body:**
```json
{
  "target": "/mnt/data",
  "force": false
}
```

**Parameters:**
- `target` (required): Device path or mount point to unmount
- `force` (optional): Force unmount even if busy

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "device unmounted successfully"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Missing target parameter
- `500 Internal Server Error`: Unmount operation failed

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"target":"/mnt/backup","force":false}' \
  http://localhost:8080/api/v1/disk/unmount
```

**Audit Log:** `disk.unmount` with target and force flag

---

### GET /api/v1/disk/smart

Retrieves SMART health information for a disk.

**Query Parameters:**
- `device` (required): Device path (e.g., /dev/sda)

**Response:**
```json
{
  "success": true,
  "data": {
    "healthy": true,
    "temperature": 35,
    "power_on_hours": 5000,
    "raw_data": "smartctl output..."
  }
}
```

**Field Descriptions:**
- `healthy`: Overall health status (true if PASSED)
- `temperature`: Current temperature in Celsius
- `power_on_hours`: Total hours disk has been powered on
- `raw_data`: Full smartctl output for advanced analysis

**Requirements:**
- Requires `smartctl` (smartmontools package) installed
- May require elevated permissions for some devices

**Example:**
```bash
curl "http://localhost:8080/api/v1/disk/smart?device=/dev/sda"
```

**Error Responses:**
- `400 Bad Request`: Missing device parameter
- `500 Internal Server Error`: smartctl command failed or not installed

**Audit Log:** `disk.smart` with device path

---

## Future APIs

Planned API additions:
- Network disk management (CIFS/NFS mounting)
- Network management (interface, IP configuration)
- Share management (Samba, NFS)
- File indexing and search
- Task scheduling

See `IMPLEMENTATION.md` for development roadmap.
