#!/bin/bash

set -e  # Exit on any error

if [ "$EUID" -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi

echo "Uninstalling European Smart Meter..."

# Stop the service
echo "Stopping service..."
systemctl stop esm-interpreter-api.service 2>/dev/null || echo "Service was not running"

# Disable the service
echo "Disabling service..."
systemctl disable esm-interpreter-api.service 2>/dev/null || echo "Service was not enabled"

# Remove service file
SERVICE_FILE="/etc/systemd/system/esm-interpreter-api.service"
if [ -f "$SERVICE_FILE" ]; then
    echo "Removing service file..."
    rm "$SERVICE_FILE"
else
    echo "Service file not found"
fi

# Reload systemd
echo "Reloading systemd..."
systemctl daemon-reload

# Remove binary only
INSTALL_DIR="/usr/bin/european_smart_meter"
BINARY_PATH="$INSTALL_DIR/interpreter_api"
if [ -f "$BINARY_PATH" ]; then
    echo "Removing interpreter_api binary..."
    rm "$BINARY_PATH"
else
    echo "Binary not found"
fi

echo ""
echo "✅ European Smart Meter interpreter has been uninstalled!"
echo ""
echo "Removed:"
echo "  - Service: esm-interpreter-api.service"
echo "  - Binary: $BINARY_PATH"
echo ""
echo "Preserved:"
echo "  - Directory: $INSTALL_DIR (for other components)"
echo "  - User serial port permissions (for other applications)"