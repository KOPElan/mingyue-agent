#!/bin/bash
# Mingyue Agent Uninstallation Script

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

stop_service() {
    if command -v systemctl &> /dev/null; then
        if systemctl is-active --quiet mingyue-agent; then
            log_info "Stopping service..."
            systemctl stop mingyue-agent
        fi

        if systemctl is-enabled --quiet mingyue-agent 2>/dev/null; then
            log_info "Disabling service..."
            systemctl disable mingyue-agent
        fi
    fi
}

remove_service() {
    if [[ -f "$SYSTEMD_DIR/mingyue-agent.service" ]]; then
        log_info "Removing systemd service..."
        rm -f "$SYSTEMD_DIR/mingyue-agent.service"
        if command -v systemctl &> /dev/null; then
            systemctl daemon-reload
        fi
    fi
}

remove_binary() {
    if [[ -f "$INSTALL_DIR/mingyue-agent" ]]; then
        log_info "Removing binary..."
        rm -f "$INSTALL_DIR/mingyue-agent"
    fi
}

remove_config() {
    if [[ -d "$CONFIG_DIR" ]]; then
        log_warn "Configuration directory: $CONFIG_DIR"
        read -p "Remove configuration directory? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Removing configuration..."
            rm -rf "$CONFIG_DIR"
        else
            log_info "Keeping configuration directory"
        fi
    fi
}

remove_logs() {
    if [[ -d "$LOG_DIR" ]]; then
        log_warn "Log directory: $LOG_DIR"
        read -p "Remove log directory? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Removing logs..."
            rm -rf "$LOG_DIR"
        else
            log_info "Keeping log directory"
        fi
    fi
}

remove_run_dir() {
    if [[ -d "$RUN_DIR" ]]; then
        log_info "Removing run directory..."
        rm -rf "$RUN_DIR"
    fi
}

remove_user() {
    if id "$USER" &>/dev/null; then
        read -p "Remove user $USER? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Removing user $USER..."
            userdel "$USER" 2>/dev/null || true
        else
            log_info "Keeping user $USER"
        fi
    fi
}

main() {
    log_info "Starting Mingyue Agent uninstallation..."

    check_root
    stop_service
    remove_service
    remove_binary
    remove_run_dir
    remove_config
    remove_logs
    remove_user

    log_info "Uninstallation completed"
}

# Run main function
main
