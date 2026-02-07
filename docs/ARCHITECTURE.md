# Mingyue Agent Architecture Documentation

## System Overview

Mingyue Agent is a single-process daemon that provides secure management capabilities for home servers. It combines API services, automation, privileged operations, and multimedia processing in a unified architecture.

## Architecture Principles

### Design Goals

1. **Security First**: All operations validate inputs, enforce whitelists, and maintain comprehensive audit trails
2. **Minimal Privilege**: Main process runs as non-root; privileged operations use temporary elevation
3. **Separation of Concerns**: Clean module boundaries with single-responsibility packages
4. **Production Ready**: Graceful shutdown, proper error handling, structured logging
5. **Extensible**: Plugin-ready architecture for future enhancements

### Core Principles

- **No arbitrary command execution**: All operations use type-safe APIs
- **Whitelist-based access**: File and network operations restricted to configured paths
- **Comprehensive auditing**: All privileged operations logged with full context
- **Fail-safe defaults**: Conservative security settings out of the box

## Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     External Clients                         │
│          (WebUI, CLI tools, Monitoring systems)              │
└────────┬──────────────────────────────────────┬─────────────┘
         │                                       │
         ├─ HTTP API (port 8080)               │
         ├─ gRPC API (port 9090)               │
         └─ Unix Domain Socket                  │
         │                                       │
┌────────▼───────────────────────────────────────▼─────────────┐
│                      API Layer                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Health  │  │   File   │  │ Monitor  │  │  (Future) │    │
│  │ Handlers │  │ Handlers │  │ Handlers │  │ Handlers │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
└────────┬───────────────────────────────────────┬─────────────┘
         │                                       │
┌────────▼───────────────────────────────────────▼─────────────┐
│                    Business Logic Layer                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │   File   │  │ Resource │  │   Disk   │  │   Task   │    │
│  │ Manager  │  │ Monitor  │  │ Manager  │  │Scheduler │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
│       │             │              │              │           │
│  ┌────▼─────────────▼──────────────▼──────────────▼─────┐   │
│  │             Path Validator & Security                 │   │
│  └────────────────────────────────────────────────────────┘  │
└────────┬───────────────────────────────────────┬─────────────┘
         │                                       │
┌────────▼───────────────────────────────────────▼─────────────┐
│                   Infrastructure Layer                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Audit   │  │  Config  │  │  Server  │  │  Daemon  │    │
│  │  Logger  │  │ Manager  │  │Framework │  │Lifecycle │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
└───────────────────────────────────────────────────────────────┘
         │
┌────────▼───────────────────────────────────────────────────┐
│                    System Resources                         │
│     (Filesystem, Network, Processes, Hardware)              │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

### cmd/agent

**Purpose**: Application entry point
**Responsibilities**:
- CLI command parsing (cobra)
- Version information
- Signal handling
- Bootstrap daemon

**Key Files**:
- `main.go`: Entry point with command definitions

### internal/api

**Purpose**: HTTP/gRPC API handlers
**Responsibilities**:
- Request routing
- Input validation
- Response formatting
- User identification

**Key Files**:
- `handlers.go`: Core endpoints (health, status, registration)
- `file_handlers.go`: File management API (12 endpoints)
- `monitor_handlers.go`: Resource monitoring API

**Design Patterns**:
- Handler functions wrapped in API structs for dependency injection
- Consistent response format across all endpoints
- Per-handler user context extraction

### internal/audit

**Purpose**: Structured audit logging
**Responsibilities**:
- Log all privileged operations
- JSON-formatted entries
- Remote log shipping (optional)
- Asynchronous log processing

**Key Types**:
```go
type Entry struct {
    Timestamp time.Time
    User      string
    Action    string
    Resource  string
    Result    string
    SourceIP  string
    Details   map[string]interface{}
}
```

**Usage Pattern**:
```go
audit.Log(ctx, &audit.Entry{
    User:     user,
    Action:   "delete",
    Resource: path,
    Result:   "success",
})
```

### internal/config

**Purpose**: Configuration management
**Responsibilities**:
- YAML parsing and validation
- Default values
- Type safety
- Configuration hot-reload (planned)

**Configuration Structure**:
```yaml
server:      # Server binding configuration
api:         # API protocol settings
audit:       # Audit log configuration
security:    # Security policies
```

### internal/daemon

**Purpose**: Daemon lifecycle management
**Responsibilities**:
- Process initialization
- Graceful shutdown
- Log directory setup
- Component orchestration

**Lifecycle**:
1. Load configuration
2. Initialize audit logger
3. Create server instance
4. Start server (non-blocking)
5. Wait for shutdown signal
6. Graceful cleanup

### internal/server

**Purpose**: Multi-protocol server framework
**Responsibilities**:
- HTTP server management
- gRPC server management
- Unix domain socket handling
- Concurrent goroutine orchestration

**Key Features**:
- Shared handler registration across protocols
- Independent protocol lifecycle
- Context-based cancellation
- WaitGroup synchronization

### internal/filemanager

**Purpose**: Secure file operations
**Responsibilities**:
- File/directory CRUD operations
- Path validation and security
- Upload/download with resumption
- Symlink/hardlink creation

**Security Components**:

1. **PathValidator**: Prevents traversal attacks
   ```go
   // Validates paths before any operation
   - Absolute path requirement
   - No .. components
   - No null bytes
   - Whitelist enforcement
   ```

2. **Manager**: Orchestrates operations with audit
   ```go
   // Every operation pattern:
   1. Validate path(s)
   2. Perform operation
   3. Log to audit
   4. Return result
   ```

3. **Transfer**: Handle uploads/downloads
   ```go
   // Features:
   - Size limits
   - HTTP range support
   - Resumable transfers
   - MD5 checksums
   ```

### internal/monitor

**Purpose**: System resource monitoring
**Responsibilities**:
- CPU statistics (cores, load)
- Memory usage (RAM, swap)
- Disk space monitoring
- Process metrics (goroutines, GC, file handles)

**Data Collection**:
- Uses `syscall.Sysinfo` for memory
- Uses `syscall.Statfs` for disk
- Uses `runtime.ReadMemStats` for process
- Reads `/proc` for file handles

## Data Flow

### File Operation Example

```
1. Client Request
   ↓
2. HTTP Handler (file_handlers.go)
   - Extract parameters
   - Get user from header
   ↓
3. File Manager (manager.go)
   - Validate path (validation.go)
   - Check whitelist
   - Perform operation
   ↓
4. Audit Logger (audit.go)
   - Create log entry
   - Write to file
   - Queue for remote push (if enabled)
   ↓
5. Return Response
   - Success/error status
   - Operation result
```

### Health Check Flow

```
1. Client Request → /healthz
   ↓
2. Monitor Handler
   ↓
3. Resource Monitor
   - Check memory usage
   - Check disk usage
   - Evaluate thresholds
   ↓
4. Return Health Status
   - 200 OK: healthy
   - 503 Service Unavailable: degraded
```

## Security Architecture

### Defense Layers

1. **Input Validation**: All external input validated at API boundary
2. **Path Whitelisting**: File operations restricted to allowed directories
3. **Audit Logging**: Complete trail of all privileged operations
4. **Process Isolation**: Non-root execution with temporary privilege elevation
5. **Type Safety**: Go's strong typing prevents many common vulnerabilities

### Attack Surface Mitigation

**Path Traversal**:
- Blocked via `PathValidator`
- Rejects `..` components
- Requires absolute paths
- Enforces whitelist

**Command Injection**:
- No shell command execution
- All operations use Go stdlib APIs
- Type-safe interfaces only

**Resource Exhaustion**:
- Upload size limits
- Rate limiting (configured)
- Health monitoring
- Graceful degradation

**Unauthorized Access**:
- Path whitelist enforcement
- User identification tracking
- Audit trail for accountability
- (Future: mTLS, token auth)

## Configuration Management

### Configuration Layers

1. **Defaults**: Hardcoded safe defaults in code
2. **File**: User-provided YAML configuration
3. **Environment**: (Future) Environment variable overrides
4. **Runtime**: (Future) Dynamic updates via API

### Configuration Validation

```go
func (c *Config) Validate() error {
    // Port range checks
    // File existence verification
    // Security policy validation
    // Mutual exclusion rules
}
```

## Deployment Architecture

### Standalone Mode

```
┌─────────────────────┐
│   Mingyue Agent     │
│   (Single Process)  │
│                     │
│  ┌───────────────┐  │
│  │ HTTP Server   │  │ :8080
│  ├───────────────┤  │
│  │ gRPC Server   │  │ :9090
│  ├───────────────┤  │
│  │ UDS Server    │  │ /var/run/...
│  └───────────────┘  │
└─────────────────────┘
```

### Multi-Instance Mode (Future)

```
┌─────────────────┐      ┌─────────────────┐
│  Agent Node 1   │      │  Agent Node 2   │
│  192.168.1.10   │      │  192.168.1.11   │
└────────┬────────┘      └────────┬────────┘
         │                        │
         └────────┬───────────────┘
                  │
         ┌────────▼────────┐
         │   WebUI Control │
         │     Center      │
         └─────────────────┘
```

## Concurrency Model

### Goroutine Usage

1. **HTTP Server**: One goroutine per request (managed by net/http)
2. **gRPC Server**: One goroutine per request (managed by gRPC)
3. **UDS Server**: One goroutine per connection
4. **Audit Logger**: Background goroutine for remote push
5. **Main Loop**: Goroutine for signal handling

### Synchronization

- `sync.WaitGroup` for server shutdown coordination
- `sync.Mutex` in audit logger for file access
- Channel-based communication for log push
- Context for cancellation propagation

## Error Handling Strategy

### Error Wrapping

```go
if err != nil {
    return fmt.Errorf("operation context: %w", err)
}
```

### Error Logging

- Critical errors: Log and return
- Operational errors: Audit log + return
- Validation errors: Return without logging

### Error Response

```go
Response{
    Success: false,
    Error:   err.Error(), // User-facing message
}
```

## Performance Considerations

### Hot Paths

1. **Health Check**: Minimal computation, cached where possible
2. **File Listing**: Direct syscall, no buffering
3. **Resource Stats**: Syscall-based, sub-millisecond latency

### Optimization Strategies

- Preallocate slices where size is known
- Reuse buffers for I/O operations (future)
- Lazy initialization of expensive resources
- Minimal allocations in monitoring code

## Future Architecture

### Planned Enhancements

1. **Authentication Layer**:
   - mTLS for node-to-node
   - JWT tokens for API access
   - Session management

2. **Privilege Separation**:
   - Separate privileged subprocess
   - Capability-based permissions
   - Setuid helper binary

3. **Task Scheduler**:
   - Distributed task queue
   - Persistent task storage
   - Progress tracking

4. **Plugin System**:
   - Dynamic module loading
   - Extension points
   - Third-party integrations

## Testing Strategy

### Unit Tests

- Package-level tests for business logic
- Mock audit logger for testing
- Table-driven tests for validators

### Integration Tests

- End-to-end API tests
- Multi-protocol consistency
- Error path validation

### Security Tests

- Path traversal attempts
- Malformed input handling
- Resource exhaustion scenarios

## Monitoring & Observability

### Metrics (Current)

- System resources via `/api/v1/monitor/stats`
- Health status via `/healthz`
- Process statistics

### Logs

- Structured JSON audit logs
- Application logs (stdout/file)
- Separate security event log (planned)

### Future Observability

- Prometheus metrics export
- OpenTelemetry tracing
- Grafana dashboards
- Alert integration

## References

- [Go Best Practices](https://go.dev/doc/effective_go)
- [Project PRD](../prd.md)
- [Implementation Progress](../IMPLEMENTATION.md)
- [API Documentation](API.md)
