# Mingyue Agent CLI Reference

## Overview

The Mingyue Agent CLI provides a comprehensive command-line interface to interact with the Mingyue Agent API. It allows you to manage files, disks, monitor system resources, schedule tasks, and more, all from the command line.

## Installation

The CLI is part of the main `mingyue-agent` binary. After building or installing the agent, you can use all CLI commands.

```bash
# Build from source
make build

# Or download from releases
wget https://github.com/KOPElan/mingyue-agent/releases/latest/download/mingyue-agent-linux-amd64.tar.gz
tar -xzf mingyue-agent-linux-amd64.tar.gz
```

## Global Flags

These flags apply to all commands that interact with the API (all except `start` and `version`):

| Flag | Description | Default |
|------|-------------|---------|
| `--api-url` | API server URL | `http://localhost:8080` |
| `--api-key` | API authentication key | (empty) |
| `--user` | User identifier for audit logs | (empty) |

## Commands

### Daemon Management

#### start

Start the Mingyue Agent daemon process.

```bash
mingyue-agent start [--config CONFIG_FILE]
```

**Flags:**
- `-c, --config`: Path to configuration file (default: `/etc/mingyue-agent/config.yaml`)

**Examples:**
```bash
# Start with default config
mingyue-agent start

# Start with custom config
mingyue-agent start --config ./my-config.yaml
```

#### version

Print version information.

```bash
mingyue-agent version
```

### File Management

Manage files and directories on the system.

#### files list

List files and directories in a path.

```bash
mingyue-agent files list <path>
```

**Examples:**
```bash
mingyue-agent files list /home/user
mingyue-agent files list /data --api-url http://remote:8080
```

#### files info

Get detailed information about a file or directory.

```bash
mingyue-agent files info <path>
```

**Examples:**
```bash
mingyue-agent files info /etc/hosts
```

#### files mkdir

Create a new directory.

```bash
mingyue-agent files mkdir <path>
```

**Examples:**
```bash
mingyue-agent files mkdir /data/newdir
```

#### files delete

Delete a file or directory.

```bash
mingyue-agent files delete <path>
```

**Examples:**
```bash
mingyue-agent files delete /tmp/oldfile.txt
```

#### files copy

Copy a file from source to destination.

```bash
mingyue-agent files copy <source> <destination>
```

**Examples:**
```bash
mingyue-agent files copy /data/file.txt /backup/file.txt
```

#### files move

Move a file or directory.

```bash
mingyue-agent files move <source> <destination>
```

**Examples:**
```bash
mingyue-agent files move /tmp/data /archive/data
```

### Disk Management

Manage disks, partitions, and monitor disk health.

#### disk list

List all available disks.

```bash
mingyue-agent disk list
```

**Examples:**
```bash
mingyue-agent disk list
```

#### disk partitions

List all disk partitions with usage information.

```bash
mingyue-agent disk partitions
```

**Examples:**
```bash
mingyue-agent disk partitions
```

#### disk smart

Get SMART information for a disk.

```bash
mingyue-agent disk smart <device>
```

**Examples:**
```bash
mingyue-agent disk smart /dev/sda
```

#### disk mount

Mount a disk partition.

```bash
mingyue-agent disk mount <device> --mount-point <path> [--filesystem <type>]
```

**Flags:**
- `-m, --mount-point`: Mount point path (required)
- `-f, --filesystem`: Filesystem type (default: `ext4`)

**Examples:**
```bash
mingyue-agent disk mount /dev/sdb1 --mount-point /mnt/data
mingyue-agent disk mount /dev/sdc1 -m /mnt/backup -f ntfs
```

#### disk unmount

Unmount a disk partition.

```bash
mingyue-agent disk unmount <target> [--force]
```

**Flags:**
- `-f, --force`: Force unmount

**Examples:**
```bash
mingyue-agent disk unmount /mnt/data
mingyue-agent disk unmount /mnt/backup --force
```

### System Monitoring

Monitor system resources and health.

#### monitor stats

Get comprehensive system resource statistics.

```bash
mingyue-agent monitor stats
```

**Examples:**
```bash
mingyue-agent monitor stats
```

**Output includes:**
- CPU: cores, usage, load averages
- Memory: total, used, available, swap
- Disk: total, used, free
- Process: PID, goroutines, memory, GC stats
- Uptime

#### monitor health

Get system health status.

```bash
mingyue-agent monitor health
```

**Examples:**
```bash
mingyue-agent monitor health
```

### File Indexing

Index and search files on the system.

#### indexer scan

Scan paths for file indexing.

```bash
mingyue-agent indexer scan <paths...> [--recursive] [--incremental]
```

**Flags:**
- `-r, --recursive`: Scan directories recursively (default: `true`)
- `-i, --incremental`: Incremental scan - skip unchanged files (default: `true`)

**Examples:**
```bash
mingyue-agent indexer scan /data
mingyue-agent indexer scan /home /data --recursive
```

#### indexer search

Search indexed files.

```bash
mingyue-agent indexer search <query> [--limit N]
```

**Flags:**
- `-l, --limit`: Maximum number of results (default: `10`)

**Examples:**
```bash
mingyue-agent indexer search "photo"
mingyue-agent indexer search "vacation" --limit 50
```

#### indexer stats

Get indexer statistics.

```bash
mingyue-agent indexer stats
```

**Examples:**
```bash
mingyue-agent indexer stats
```

### Task Scheduling

Manage scheduled tasks.

#### scheduler list

List all scheduled tasks.

```bash
mingyue-agent scheduler list
```

**Examples:**
```bash
mingyue-agent scheduler list
```

#### scheduler add

Add a new scheduled task.

```bash
mingyue-agent scheduler add <name> [--type TYPE] [--schedule SCHEDULE] [--enabled]
```

**Flags:**
- `-t, --type`: Task type - `cleanup`, `backup`, or `indexing` (default: `cleanup`)
- `-s, --schedule`: Schedule - `daily`, `weekly`, `monthly`, or cron expression (default: `daily`)
- `-e, --enabled`: Enable task immediately (default: `true`)

**Examples:**
```bash
mingyue-agent scheduler add "Daily Cleanup"
mingyue-agent scheduler add "Weekly Backup" --type backup --schedule weekly
mingyue-agent scheduler add "Nightly Index" -t indexing -s "0 2 * * *"
```

#### scheduler remove

Remove a scheduled task.

```bash
mingyue-agent scheduler remove <task-id>
```

**Examples:**
```bash
mingyue-agent scheduler remove task-123
```

#### scheduler execute

Execute a task immediately.

```bash
mingyue-agent scheduler execute <task-id>
```

**Examples:**
```bash
mingyue-agent scheduler execute task-123
```

### Authentication

Manage API tokens and authentication.

#### auth token-create

Create a new API token.

```bash
mingyue-agent auth token-create <name> [--user USER] [--expires SECONDS]
```

**Flags:**
- `-u, --user`: User ID (default: `admin`)
- `-e, --expires`: Token expiration in seconds (default: `31536000` = 1 year)

**Examples:**
```bash
mingyue-agent auth token-create my-token
mingyue-agent auth token-create automation-token --user automation --expires 86400
```

**Important:** Save the token immediately - you won't be able to see it again!

#### auth token-list

List all API tokens.

```bash
mingyue-agent auth token-list
```

**Examples:**
```bash
mingyue-agent auth token-list
```

#### auth token-revoke

Revoke an API token.

```bash
mingyue-agent auth token-revoke <token-id>
```

**Examples:**
```bash
mingyue-agent auth token-revoke token-abc123
```

## Configuration

### API Connection

By default, the CLI connects to `http://localhost:8080`. You can override this with the `--api-url` flag or by setting environment variables:

```bash
# Using flag
mingyue-agent files list /data --api-url http://192.168.1.100:8080

# Using environment variable
export MINGYUE_API_URL=http://192.168.1.100:8080
mingyue-agent files list /data
```

### Authentication

If token authentication is enabled on the server, provide an API key:

```bash
# Create a token first
mingyue-agent auth token-create cli-access

# Use the token in subsequent commands
mingyue-agent files list /data --api-key YOUR_TOKEN_HERE
```

### User Identification

For audit logging, specify the user making requests:

```bash
mingyue-agent files delete /data/old.txt --user alice
```

## Examples

### Complete Workflow

```bash
# 1. Start the agent
mingyue-agent start --config /etc/mingyue-agent/config.yaml &

# 2. Check system health
mingyue-agent monitor health

# 3. View system stats
mingyue-agent monitor stats

# 4. List files in a directory
mingyue-agent files list /home/user

# 5. Create a backup directory
mingyue-agent files mkdir /backup/$(date +%Y%m%d)

# 6. Copy important files
mingyue-agent files copy /data/important.db /backup/$(date +%Y%m%d)/

# 7. Check disk usage
mingyue-agent disk partitions

# 8. Monitor disk health
mingyue-agent disk smart /dev/sda

# 9. Index files for search
mingyue-agent indexer scan /data --recursive

# 10. Search for files
mingyue-agent indexer search "report"

# 11. Schedule a daily cleanup task
mingyue-agent scheduler add "Daily Cleanup" --type cleanup --schedule daily
```

### Remote Server Management

```bash
# Manage a remote server
export REMOTE_URL="http://server.example.com:8080"
export API_KEY="your-api-key-here"

mingyue-agent monitor stats --api-url $REMOTE_URL --api-key $API_KEY
mingyue-agent disk list --api-url $REMOTE_URL --api-key $API_KEY
mingyue-agent files list /data --api-url $REMOTE_URL --api-key $API_KEY
```

## Troubleshooting

### Connection Refused

If you get "connection refused" errors, make sure the agent is running:

```bash
# Check if agent is running
ps aux | grep mingyue-agent

# Start the agent if needed
mingyue-agent start
```

### Authentication Errors

If you get authentication errors:

1. Check if token authentication is enabled in the config
2. Create a valid token: `mingyue-agent auth token-create my-token`
3. Use the token in your commands: `--api-key YOUR_TOKEN`

### Permission Denied

For disk operations and certain file operations, the agent needs appropriate permissions:

1. Run the agent with sufficient privileges
2. Check the allowed paths in the configuration
3. Ensure the user has access to the requested resources

## See Also

- [API Documentation](API.md) - Complete REST API reference
- [Architecture Guide](ARCHITECTURE.md) - Technical architecture
- [Deployment Guide](DEPLOYMENT.md) - Installation and deployment
