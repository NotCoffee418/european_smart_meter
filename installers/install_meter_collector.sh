#!/bin/bash

set -e  # Exit on any error

if [ "$EUID" -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi


echo "Installing European Smart Meter Collector..."


# Create installation directory
INSTALL_DIR="/usr/bin/european_smart_meter"
CONFIG_DIR="/etc/european_smart_meter"
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        BINARY_NAME="meter_collector-linux-amd64"
        ;;
    aarch64)
        BINARY_NAME="meter_collector-linux-arm64"
        ;;
    armv6l)
        BINARY_NAME="meter_collector-linux-arm6"
        ;;
    armv7l)
        BINARY_NAME="meter_collector-linux-arm7"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        echo "Supported: x86_64, aarch64, armv6l, armv7l"
        exit 1
        ;;
esac

echo "Detected architecture: $ARCH (using $BINARY_NAME)"

# Get latest release info
echo "Fetching latest release..."
LATEST_URL=$(curl -s https://api.github.com/repos/NotCoffee418/european_smart_meter/releases/latest | grep "browser_download_url.*$BINARY_NAME" | cut -d '"' -f 4)

if [ -z "$LATEST_URL" ]; then
    echo "Error: Could not find download URL for $BINARY_NAME"
    exit 1
fi

echo "Downloading from: $LATEST_URL"

# Stop service if it exists (for updates)
echo "Stopping existing service if running..."
systemctl stop esm-meter-collector.service 2>/dev/null || true
sleep 1

# Download the binary
curl -L -o "$INSTALL_DIR/meter_collector" "$LATEST_URL"
chmod +x "$INSTALL_DIR/meter_collector"

echo "Binary installed to $INSTALL_DIR/meter_collector"

# Create config file if it doesn't exist
if [ ! -f "$CONFIG_DIR/meter_collector.toml" ]; then
CONFIG_FILE="$CONFIG_DIR/meter_collector.toml"
cat > "$CONFIG_FILE" << EOF
interpreter_api_host = "localhost:9039"
tls_enabled = false
EOF
fi
echo "Created config file at $CONFIG_FILE"

# Create directory for database if it doesn't exist
mkdir -p "/var/lib/european_smart_meter"


# Create systemd service
SERVICE_FILE="/etc/systemd/system/esm-meter-collector.service"
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=European Smart Meter Collector
After=network.target

[Service]
Type=simple
User=root
ExecStart=$INSTALL_DIR/meter_collector
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

echo "Created systemd service"

# Reload systemd, enable and start service
echo "Starting service..."
systemctl daemon-reload
systemctl enable esm-meter-collector.service
systemctl restart esm-meter-collector.service

# Inform completion
echo ""
echo "Installation complete!"
echo "Service status: systemctl status esm-meter-collector"
echo "View logs: journalctl -u esm-meter-collector -f"