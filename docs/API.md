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

## Network Disk Management APIs

### GET /api/v1/netdisk/shares

Lists all configured network shares (CIFS/NFS).

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "cifs-192.168.1.100-1707312000",
      "name": "backup-share",
      "protocol": "cifs",
      "host": "192.168.1.100",
      "path": "/backup",
      "mount_point": "/mnt/backup",
      "username": "user",
      "options": {},
      "auto_mount": true,
      "mounted": true,
      "last_checked": "2026-02-07T14:30:00Z",
      "healthy": true
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/netdisk/shares
```

---

### POST /api/v1/netdisk/shares/add

Adds a new network share configuration.

**Request Body:**
```json
{
  "name": "media-share",
  "protocol": "nfs",
  "host": "192.168.1.200",
  "path": "/export/media",
  "mount_point": "/mnt/media",
  "options": {
    "vers": "4"
  },
  "auto_mount": true
}
```

**For CIFS:**
```json
{
  "name": "documents",
  "protocol": "cifs",
  "host": "192.168.1.150",
  "path": "/documents",
  "mount_point": "/mnt/documents",
  "username": "user",
  "password": "password",
  "options": {},
  "auto_mount": false
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "share_id": "nfs-192.168.1.200-1707312100"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"media","protocol":"nfs","host":"192.168.1.200","path":"/export/media","mount_point":"/mnt/media"}' \
  http://localhost:8080/api/v1/netdisk/shares/add
```

**Security:** Passwords are encrypted using AES-256-GCM before storage.

---

### DELETE /api/v1/netdisk/shares/remove

Removes a network share (unmounts if mounted).

**Query Parameters:**
- `id` (required): Share ID

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "share removed"
  }
}
```

**Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/netdisk/shares/remove?id=nfs-192.168.1.200-1707312100"
```

---

### POST /api/v1/netdisk/mount

Mounts a configured network share.

**Request Body:**
```json
{
  "id": "cifs-192.168.1.100-1707312000"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "share mounted"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"id":"cifs-192.168.1.100-1707312000"}' \
  http://localhost:8080/api/v1/netdisk/mount
```

---

### POST /api/v1/netdisk/unmount

Unmounts a network share.

**Request Body:**
```json
{
  "id": "cifs-192.168.1.100-1707312000"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "share unmounted"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"id":"cifs-192.168.1.100-1707312000"}' \
  http://localhost:8080/api/v1/netdisk/unmount
```

---

### GET /api/v1/netdisk/status

Gets the status of a specific share.

**Query Parameters:**
- `id` (required): Share ID

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "cifs-192.168.1.100-1707312000",
    "name": "backup-share",
    "mounted": true,
    "healthy": true,
    "last_checked": "2026-02-07T14:35:00Z"
  }
}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/netdisk/status?id=cifs-192.168.1.100-1707312000"
```

---

## Network Management APIs

### GET /api/v1/network/interfaces

Lists all network interfaces with statistics.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "name": "eth0",
      "mac": "00:0c:29:12:34:56",
      "ip_addresses": ["192.168.1.10", "fe80::20c:29ff:fe12:3456"],
      "state": "up",
      "speed": 1000,
      "mtu": 1500,
      "rx_bytes": 123456789,
      "tx_bytes": 987654321,
      "rx_packets": 654321,
      "tx_packets": 123456,
      "rx_errors": 0,
      "tx_errors": 0,
      "flags": ["UP"],
      "last_updated": "2026-02-07T14:40:00Z"
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/network/interfaces
```

---

### GET /api/v1/network/interface

Gets detailed information about a specific interface.

**Query Parameters:**
- `name` (required): Interface name

**Response:**
```json
{
  "success": true,
  "data": {
    "name": "eth0",
    "mac": "00:0c:29:12:34:56",
    "ip_addresses": ["192.168.1.10"],
    "state": "up",
    "speed": 1000,
    "mtu": 1500
  }
}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/network/interface?name=eth0"
```

---

### POST /api/v1/network/config

Sets IP configuration for an interface.

**Request Body:**
```json
{
  "config": {
    "interface": "eth1",
    "method": "static",
    "address": "192.168.2.10",
    "netmask": "24",
    "gateway": "192.168.2.1",
    "dns_servers": ["8.8.8.8", "8.8.4.4"]
  },
  "reason": "Changing to static IP for server"
}
```

**For DHCP:**
```json
{
  "config": {
    "interface": "eth1",
    "method": "dhcp"
  },
  "reason": "Switching to DHCP"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "IP config updated"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"config":{"interface":"eth1","method":"static","address":"192.168.2.10","netmask":"24","gateway":"192.168.2.1"},"reason":"Configuration update"}' \
  http://localhost:8080/api/v1/network/config
```

**Security:** Cannot configure management interface to prevent self-disconnection.

---

### POST /api/v1/network/rollback

Rolls back to a previous network configuration.

**Request Body:**
```json
{
  "history_id": "eth1-1707312000"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "config rolled back"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"history_id":"eth1-1707312000"}' \
  http://localhost:8080/api/v1/network/rollback
```

---

### GET /api/v1/network/history

Gets configuration history for an interface.

**Query Parameters:**
- `interface` (optional): Filter by interface name

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "eth1-1707312000",
      "timestamp": "2026-02-07T14:00:00Z",
      "interface": "eth1",
      "config": {
        "method": "static",
        "address": "192.168.2.10",
        "netmask": "24"
      },
      "user": "admin",
      "reason": "Initial configuration"
    }
  ]
}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/network/history?interface=eth1"
```

---

### POST /api/v1/network/enable

Enables a network interface.

**Request Body:**
```json
{
  "interface": "eth1"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "interface enabled"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"interface":"eth1"}' \
  http://localhost:8080/api/v1/network/enable
```

---

### POST /api/v1/network/disable

Disables a network interface.

**Request Body:**
```json
{
  "interface": "eth1"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "interface disabled"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"interface":"eth1"}' \
  http://localhost:8080/api/v1/network/disable
```

**Security:** Cannot disable management interface.

---

### GET /api/v1/network/ports

Lists all listening ports and processes.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "port": 8080,
      "protocol": "tcp",
      "address": "0.0.0.0",
      "state": "LISTEN",
      "process": "mingyue-agent"
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/network/ports
```

---

### GET /api/v1/network/traffic

Gets real-time traffic statistics for all interfaces.

**Response:**
```json
{
  "success": true,
  "data": {
    "eth0": {
      "name": "eth0",
      "rx_bytes": 123456789,
      "tx_bytes": 987654321,
      "rx_packets": 654321,
      "tx_packets": 123456
    }
  }
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/network/traffic
```

---

## Share Management APIs

### GET /api/v1/shares

Lists all configured shares (Samba and NFS).

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "share-photos-1707312000",
      "name": "photos",
      "type": "samba",
      "path": "/data/photos",
      "description": "Family photos",
      "users": ["user1", "user2"],
      "groups": [],
      "access_mode": "ro",
      "options": {},
      "enabled": true,
      "healthy": true,
      "last_checked": "2026-02-07T14:45:00Z",
      "created_at": "2026-02-07T10:00:00Z",
      "updated_at": "2026-02-07T14:00:00Z"
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/shares
```

---

### GET /api/v1/shares/get

Gets details of a specific share.

**Query Parameters:**
- `id` (required): Share ID

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "share-photos-1707312000",
    "name": "photos",
    "type": "samba",
    "path": "/data/photos",
    "enabled": true
  }
}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/shares/get?id=share-photos-1707312000"
```

---

### POST /api/v1/shares/add

Creates a new share.

**Request Body (Samba):**
```json
{
  "name": "documents",
  "type": "samba",
  "path": "/data/documents",
  "description": "Shared documents",
  "users": ["user1"],
  "access_mode": "rw",
  "options": {
    "browseable": "yes"
  }
}
```

**Request Body (NFS):**
```json
{
  "name": "media",
  "type": "nfs",
  "path": "/data/media",
  "access_mode": "ro",
  "options": {
    "no_subtree_check": ""
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "share_id": "share-documents-1707312100"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"documents","type":"samba","path":"/data/documents","access_mode":"rw"}' \
  http://localhost:8080/api/v1/shares/add
```

---

### PUT /api/v1/shares/update

Updates an existing share.

**Query Parameters:**
- `id` (required): Share ID

**Request Body:**
```json
{
  "description": "Updated description",
  "access_mode": "ro",
  "users": ["user1", "user2"]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "share updated"
  }
}
```

**Example:**
```bash
curl -X PUT -H "Content-Type: application/json" \
  -d '{"access_mode":"ro"}' \
  "http://localhost:8080/api/v1/shares/update?id=share-documents-1707312100"
```

---

### DELETE /api/v1/shares/remove

Removes a share.

**Query Parameters:**
- `id` (required): Share ID

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "share removed"
  }
}
```

**Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/shares/remove?id=share-documents-1707312100"
```

---

### POST /api/v1/shares/enable

Enables a share.

**Query Parameters:**
- `id` (required): Share ID

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "share enabled"
  }
}
```

**Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/shares/enable?id=share-documents-1707312100"
```

---

### POST /api/v1/shares/disable

Disables a share.

**Query Parameters:**
- `id` (required): Share ID

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "share disabled"
  }
}
```

**Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/shares/disable?id=share-documents-1707312100"
```

---

### POST /api/v1/shares/rollback

Rolls back to a previous share configuration.

**Request Body:**
```json
{
  "timestamp": 1707312000
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "config rolled back"
  }
}
```

**Example:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"timestamp":1707312000}' \
  http://localhost:8080/api/v1/shares/rollback
```

---

## Future APIs

Planned API additions:
- File indexing and search
- Task scheduling

See `IMPLEMENTATION.md` for development roadmap.
