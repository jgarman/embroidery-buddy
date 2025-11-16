#!/bin/bash
set -e

# Installation script for embroidery-usbd systemd service
# This script must be run as root

if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (use sudo)"
  exit 1
fi

# Configuration
BINARY_PATH="/usr/local/bin/embroidery-usbd"
CONFIG_DIR="/etc/embroidery-usbd"
CONFIG_FILE="${CONFIG_DIR}/config.json"
DATA_DIR="/var/lib/embroidery-usbd"
SERVICE_FILE="/etc/systemd/system/embroidery-usbd.service"

echo "Installing embroidery-usbd service..."

# Create directories
echo "Creating directories..."
mkdir -p "${CONFIG_DIR}"
mkdir -p "${DATA_DIR}"

# Copy binary
if [ -f "./embroidery-usbd" ]; then
    echo "Installing binary to ${BINARY_PATH}..."
    cp ./embroidery-usbd "${BINARY_PATH}"
    chmod +x "${BINARY_PATH}"
else
    echo "Error: embroidery-usbd binary not found in current directory"
    echo "Please build it first with: make build"
    exit 1
fi

# Generate default config if it doesn't exist
if [ ! -f "${CONFIG_FILE}" ]; then
    echo "Generating default configuration..."
    "${BINARY_PATH}" -generate-config -config "${CONFIG_FILE}"

    # Update config with production defaults
    echo "You may want to edit ${CONFIG_FILE} to customize:"
    echo "  - Disk image path and size"
    echo "  - USB gadget identifiers"
    echo "  - Server port and CORS settings"
    echo "  - mDNS service name"
else
    echo "Config file already exists at ${CONFIG_FILE}"
fi

# Copy service file
echo "Installing systemd service..."
if [ -f "./embroidery-usbd.service" ]; then
    cp ./embroidery-usbd.service "${SERVICE_FILE}"
else
    echo "Warning: embroidery-usbd.service not found, creating basic service file"
    cat > "${SERVICE_FILE}" << 'EOF'
[Unit]
Description=Embroidery Buddy USB Gadget Server
After=network.target avahi-daemon.service

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/embroidery-usbd -config /etc/embroidery-usbd/config.json
WorkingDirectory=/var/lib/embroidery-usbd
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF
fi

# Reload systemd
echo "Reloading systemd daemon..."
systemctl daemon-reload

# Enable service
echo "Enabling embroidery-usbd service..."
systemctl enable embroidery-usbd.service

echo ""
echo "Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Edit configuration (optional): ${CONFIG_FILE}"
echo "  2. Start the service: sudo systemctl start embroidery-usbd"
echo "  3. Check status: sudo systemctl status embroidery-usbd"
echo "  4. View logs: sudo journalctl -u embroidery-usbd -f"
echo ""
