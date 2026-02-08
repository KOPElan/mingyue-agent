#!/bin/bash
# Mingyue Agent Setup Verification Script
# This script verifies that all required directories and permissions are correctly configured

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/mingyue-agent"
LOG_DIR="/var/log/mingyue-agent"
RUN_DIR="/var/run/mingyue-agent"
DATA_DIR="/var/lib/mingyue-agent"
USER="mingyue-agent"
GROUP="mingyue-agent"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_directory() {
    local dir="$1"
    local desc="$2"
    local owner="$3"
    local required="${4:-true}"
    
    if [[ ! -d "$dir" ]]; then
        if [[ "$required" == "true" ]]; then
            log_error "$desc does not exist: $dir"
            echo "         Create with: sudo mkdir -p $dir"
            return 1
        else
            log_warn "$desc does not exist (optional): $dir"
            return 0
        fi
    fi
    
    # Check permissions
    local dir_owner=$(stat -c '%U:%G' "$dir" 2>/dev/null || stat -f '%Su:%Sg' "$dir" 2>/dev/null)
    local dir_perms=$(stat -c '%a' "$dir" 2>/dev/null || stat -f '%A' "$dir" 2>/dev/null)
    
    if [[ "$owner" != "" && "$dir_owner" != "$owner" ]]; then
        log_error "$desc has incorrect owner: $dir_owner (expected: $owner)"
        echo "         Fix with: sudo chown -R $owner $dir"
        return 1
    fi
    
    # Check if writable
    if [[ "$owner" == "$USER:$GROUP" ]]; then
        if sudo -u "$USER" test -w "$dir" 2>/dev/null; then
            log_success "$desc ($dir_perms) - $dir"
            return 0
        else
            log_error "$desc exists but is not writable by $USER"
            echo "         Fix with: sudo chown -R $owner $dir && sudo chmod 755 $dir"
            return 1
        fi
    else
        log_success "$desc ($dir_perms) - $dir"
        return 0
    fi
}

check_file() {
    local file="$1"
    local desc="$2"
    local required="${3:-true}"
    
    if [[ ! -f "$file" ]]; then
        if [[ "$required" == "true" ]]; then
            log_error "$desc does not exist: $file"
            return 1
        else
            log_warn "$desc does not exist (optional): $file"
            return 0
        fi
    fi
    
    log_success "$desc - $file"
    return 0
}

check_user() {
    if id "$USER" &>/dev/null; then
        log_success "User '$USER' exists"
        return 0
    else
        log_error "User '$USER' does not exist"
        echo "         Create with: sudo useradd -r -s /bin/false -d /nonexistent $USER"
        return 1
    fi
}

check_binary() {
    if [[ -x "$INSTALL_DIR/mingyue-agent" ]]; then
        local version=$($INSTALL_DIR/mingyue-agent --version 2>/dev/null || echo "unknown")
        log_success "Binary installed and executable (version: $version)"
        return 0
    else
        log_error "Binary not found or not executable: $INSTALL_DIR/mingyue-agent"
        return 1
    fi
}

check_systemd_service() {
    if ! command -v systemctl &> /dev/null; then
        log_warn "systemd not available on this system"
        return 0
    fi
    
    if systemctl list-unit-files | grep -q "^mingyue-agent.service"; then
        local status=$(systemctl is-enabled mingyue-agent.service 2>/dev/null || echo "disabled")
        log_success "Systemd service installed (status: $status)"
        
        # Check if service is running
        if systemctl is-active --quiet mingyue-agent.service; then
            log_success "Service is currently running"
        else
            log_warn "Service is not running"
            echo "         Start with: sudo systemctl start mingyue-agent"
        fi
        return 0
    else
        log_error "Systemd service not installed"
        return 1
    fi
}

check_config_values() {
    local config_file="$CONFIG_DIR/config.yaml"
    
    if [[ ! -f "$config_file" ]]; then
        log_error "Configuration file not found: $config_file"
        return 1
    fi
    
    log_info "Checking configuration values..."
    
    # Check for default/insecure values that should be changed
    if grep -q "change-this-to-a-secure-key-32b" "$config_file" 2>/dev/null; then
        log_warn "Encryption key is still set to default value"
        echo "         Please update encryption_key in $config_file"
    fi
    
    log_success "Configuration file validated"
    return 0
}

main() {
    echo ""
    echo "========================================"
    echo "  Mingyue Agent Setup Verification"
    echo "========================================"
    echo ""
    
    local all_ok=true
    
    log_info "Checking user and group..."
    check_user || all_ok=false
    echo ""
    
    log_info "Checking binary installation..."
    check_binary || all_ok=false
    echo ""
    
    log_info "Checking required directories..."
    check_directory "$CONFIG_DIR" "Configuration directory" "root:root" true || all_ok=false
    check_directory "$LOG_DIR" "Log directory" "$USER:$GROUP" true || all_ok=false
    check_directory "$RUN_DIR" "Runtime directory" "$USER:$GROUP" true || all_ok=false
    check_directory "$DATA_DIR" "Data directory" "$USER:$GROUP" true || all_ok=false
    echo ""
    
    log_info "Checking application-specific directories..."
    check_directory "$DATA_DIR/share-backups" "Share backups directory" "$USER:$GROUP" true || all_ok=false
    echo ""
    
    log_info "Checking configuration files..."
    check_file "$CONFIG_DIR/config.yaml" "Main configuration" true || all_ok=false
    check_config_values || all_ok=false
    echo ""
    
    log_info "Checking systemd service..."
    check_systemd_service || all_ok=false
    echo ""
    
    echo "========================================"
    if [[ "$all_ok" == "true" ]]; then
        log_success "All checks passed! System is properly configured."
        echo ""
        echo "You can start the service with:"
        echo "  sudo systemctl start mingyue-agent"
        echo ""
        echo "Check logs with:"
        echo "  sudo journalctl -u mingyue-agent -f"
        exit 0
    else
        log_error "Some checks failed. Please review the errors above and fix them."
        echo ""
        echo "Quick fix commands:"
        echo "  sudo mkdir -p $DATA_DIR/share-backups"
        echo "  sudo chown -R $USER:$GROUP $LOG_DIR $RUN_DIR $DATA_DIR"
        echo "  sudo chmod -R 755 $LOG_DIR $RUN_DIR $DATA_DIR"
        exit 1
    fi
}

# Run main function
main
