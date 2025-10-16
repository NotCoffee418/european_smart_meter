#!/bin/bash

set -e  # Exit on any error

if [ "$EUID" -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi

echo "Installing European Smart Meter Interpreter API..."

# Get configuration from user
echo ""
echo "Configuration:"

# Auto-detect serial devices
USB_DEVICES=$(ls /dev/ttyUSB* 2>/dev/null || true)
USB_COUNT=$(echo "$USB_DEVICES" | grep -c "ttyUSB" 2>/dev/null || echo "0")

if [ "$USB_COUNT" -eq 0 ]; then
    echo "⚠️  Warning: No /dev/ttyUSB* devices found. Make sure your smart meter is connected."
    exec < /dev/tty
    read -p "Serial device path (default: /dev/ttyUSB0): " SERIAL_DEVICE < /dev/tty
    SERIAL_DEVICE=${SERIAL_DEVICE:-/dev/ttyUSB0}
elif [ "$USB_COUNT" -eq 1 ]; then
    SERIAL_DEVICE="$USB_DEVICES"
    echo "✅ Found serial device: $SERIAL_DEVICE (auto-selected)"
else
    echo "Found multiple USB serial devices:"
    echo "$USB_DEVICES"
    read -p "Serial device path (default: /dev/ttyUSB0): " SERIAL_DEVICE
    SERIAL_DEVICE=${SERIAL_DEVICE:-/dev/ttyUSB0}
fi

echo "Baudrate options:"
echo "  9600   - Older meters"
echo "  115200 - Newer meters (DSMR 4.0+/ESMR 5.x+)"
exec < /dev/tty
read -p "Baudrate (default: 115200): " BAUDRATE
BAUDRATE=${BAUDRATE:-115200}

echo "Using device: $SERIAL_DEVICE at $BAUDRATE baud"
echo ""

# Get the actual user (not root when using sudo)
ACTUAL_USER="${SUDO_USER:-$USER}"
if [ "$ACTUAL_USER" = "root" ]; then
    echo "Warning: Running as actual root user. User permissions may not work correctly."
fi

# Permissions for serial port
echo "Setting up serial port permissions for user: $ACTUAL_USER"
usermod -a -G dialout "$ACTUAL_USER"
usermod -a -G tty "$ACTUAL_USER"

# Create installation directory
INSTALL_DIR="/usr/bin/european_smart_meter"
CONFIG_DIR="/etc/european_smart_meter"
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        BINARY_NAME="interpreter_api-linux-amd64"
        ;;
    aarch64)
        BINARY_NAME="interpreter_api-linux-arm64"
        ;;
    armv6l)
        BINARY_NAME="interpreter_api-linux-arm6"
        ;;
    armv7l)
        BINARY_NAME="interpreter_api-linux-arm7"
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
systemctl stop esm-interpreter-api.service 2>/dev/null || true
sleep 1

# Download the binary
curl -L -o "$INSTALL_DIR/interpreter_api" "$LATEST_URL"
chmod +x "$INSTALL_DIR/interpreter_api"

echo "Binary installed to $INSTALL_DIR/interpreter_api"

# Create config file if it doesn't exist
if [ ! -f "$CONFIG_DIR/interpreter_api.toml" ]; then
CONFIG_FILE="$CONFIG_DIR/interpreter_api.toml"
cat > "$CONFIG_FILE" << EOF
# See README.md for more info on the config file
serial_device = "$SERIAL_DEVICE"
baudrate = $BAUDRATE
listen_address = "0.0.0.0"
listen_port = 9039
solar_inverter_ip = "192.168.200.1"
solar_inverter_modbus_port = 502 # nmap -p 80,443,502,6607,8000,8080 192.168.200.1
wlan_connection_id = "preconfigured"  # Check with `nmcli device status`
EOF
fi

echo "Created config file at $CONFIG_FILE"


# Create systemd service
SERVICE_FILE="/etc/systemd/system/esm-interpreter-api.service"
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=European Smart Meter Interpreter API
After=network.target

[Service]
Type=simple
User=root
ExecStart=$INSTALL_DIR/interpreter_api
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
systemctl enable esm-interpreter-api.service
systemctl restart esm-interpreter-api.service

# Wait a bit for service to start
echo "Waiting for service to start..."
sleep 5

# Test the service
echo "Testing service..."
if command -v python3 &> /dev/null; then
    if curl -s http://localhost:9039/latest | python3 -m json.tool > /dev/null 2>&1; then
        echo "✅ Service is running and responding with valid JSON!"
    else
        echo "❌ Service test failed. Check status with:"
        echo "systemctl status esm-interpreter-api"
        echo "journalctl -u esm-interpreter-api -f"
        echo "You may need to update the config file at /etc/european_smart_meter/interpreter_api.toml"
        exit 1
    fi
else
    echo "⚠️  python3 not found - couldn't test JSON response, but service is probably fine"
    echo "Manual test: curl http://localhost:9039/latest | python3 -m json.tool"
fi

echo ""
echo "Installation complete!"
echo "Service status: systemctl status esm-interpreter-api"
echo "View logs: journalctl -u esm-interpreter-api -f"