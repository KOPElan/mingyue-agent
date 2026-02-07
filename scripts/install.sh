#!/bin/bash
# Mingyue Agent Installation Script
# This script installs and configures Mingyue Agent on Linux systems

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/mingyue-agent"
LOG_DIR="/var/log/mingyue-agent"
RUN_DIR="/var/run/mingyue-agent"
SYSTEMD_DIR="/etc/systemd/system"
USER="mingyue-agent"
GROUP="mingyue-agent"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_system() {
    log_info "Checking system requirements..."

    # Check if Linux
    if [[ "$(uname -s)" != "Linux" ]]; then
        log_error "This script only supports Linux systems"
        exit 1
    fi

    # Check if systemd is available
    if ! command -v systemctl &> /dev/null; then
        log_warn "systemd not found, service management will not be available"
    fi

    log_info "System check passed"
}

create_user() {
    if id "$USER" &>/dev/null; then
        log_info "User $USER already exists"
    else
        log_info "Creating user $USER..."
        useradd --system --no-create-home --shell /usr/sbin/nologin "$USER"
    fi
}

create_directories() {
    log_info "Creating directories..."

    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$RUN_DIR"

    chown -R "$USER:$GROUP" "$LOG_DIR"
    chown -R "$USER:$GROUP" "$RUN_DIR"
    chmod 755 "$CONFIG_DIR"
    chmod 755 "$LOG_DIR"
    chmod 755 "$RUN_DIR"
}

install_binary() {
    log_info "Installing binary..."

    if [[ ! -f "bin/mingyue-agent" ]]; then
        log_error "Binary not found. Please run 'make build' first"
        exit 1
    fi

    install -m 755 bin/mingyue-agent "$INSTALL_DIR/mingyue-agent"
    log_info "Binary installed to $INSTALL_DIR/mingyue-agent"
}

install_config() {
    log_info "Installing configuration..."

    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        log_warn "Configuration already exists at $CONFIG_DIR/config.yaml"
        log_warn "Backing up to $CONFIG_DIR/config.yaml.bak"
        cp "$CONFIG_DIR/config.yaml" "$CONFIG_DIR/config.yaml.bak"
    fi

    if [[ -f "config.example.yaml" ]]; then
        cp config.example.yaml "$CONFIG_DIR/config.yaml"
        chown root:root "$CONFIG_DIR/config.yaml"
        chmod 644 "$CONFIG_DIR/config.yaml"
        log_info "Configuration installed to $CONFIG_DIR/config.yaml"
    else
        log_warn "config.example.yaml not found, skipping configuration installation"
    fi
}

install_systemd_service() {
    if ! command -v systemctl &> /dev/null; then
        log_warn "systemd not available, skipping service installation"
        return
    fi

    log_info "Installing systemd service..."

    cat > "$SYSTEMD_DIR/mingyue-agent.service" <<EOF
[Unit]
Description=Mingyue Agent - Local management service for home servers
Documentation=https://github.com/KOPElan/mingyue-agent
After=network.target

[Service]
Type=simple
User=$USER
Group=$GROUP
ExecStart=$INSTALL_DIR/mingyue-agent start --config $CONFIG_DIR/config.yaml
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
ReadWritePaths=$LOG_DIR $RUN_DIR
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
EOF

    systemctl daemon-reload
    log_info "Systemd service installed"
}

enable_service() {
    if ! command -v systemctl &> /dev/null; then
        return
    fi

    log_info "Enabling service..."
    systemctl enable mingyue-agent.service
    log_info "Service enabled"
}

print_next_steps() {
    log_info "Installation completed successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Edit configuration: $CONFIG_DIR/config.yaml"
    echo "2. Configure allowed paths and security settings"
    echo "3. Start the service: systemctl start mingyue-agent"
    echo "4. Check status: systemctl status mingyue-agent"
    echo "5. View logs: journalctl -u mingyue-agent -f"
    echo ""
    echo "API endpoints will be available at:"
    echo "  - HTTP: http://localhost:8080"
    echo "  - gRPC: localhost:9090"
    echo "  - Health check: http://localhost:8080/healthz"
    echo ""
    echo "For more information, see:"
    echo "  - README: https://github.com/KOPElan/mingyue-agent"
    echo "  - API docs: docs/API.md"
}

main() {
    log_info "Starting Mingyue Agent installation..."

    check_root
    check_system
    create_user
    create_directories
    install_binary
    install_config
    install_systemd_service
    enable_service
    print_next_steps
}

# Run main function
main
