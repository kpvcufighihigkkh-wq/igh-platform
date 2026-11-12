#!/bin/bash
set -euo pipefail

INSTALL_DIR="/opt/igh-edge"
SERVICE_USER="igh"
SERVICE_FILE="/etc/systemd/system/igh-edge.service"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "============================================"
echo "  IGH Edge Node - Linux Installation"
echo "============================================"
echo

# Check root
if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: Please run as root (sudo ./install.sh)"
    exit 1
fi

# Create service user
if ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
    echo "Creating user $SERVICE_USER..."
    useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
fi

# Create install directory
echo "Installing to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR/data"

# Copy files
cp "$SCRIPT_DIR/igh-edge" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/igh-edge"

if [ ! -f "$INSTALL_DIR/edge.yaml" ]; then
    cp "$SCRIPT_DIR/edge.yaml" "$INSTALL_DIR/"
    echo "NOTE: Default edge.yaml copied. Please edit it before starting the service."
fi

# Set permissions
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"

# Install systemd service
cp "$SCRIPT_DIR/igh-edge.service" "$SERVICE_FILE"
systemctl daemon-reload
systemctl enable igh-edge

echo
echo "============================================"
echo "  Installation complete!"
echo
echo "  Next steps:"
echo "    1. Edit $INSTALL_DIR/edge.yaml"
echo "    2. sudo systemctl start igh-edge"
echo "    3. sudo systemctl status igh-edge"
echo "    4. journalctl -u igh-edge -f  (view logs)"
echo "============================================"
