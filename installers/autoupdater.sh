#!/bin/bash
set -e

# Check if running as root
if [[ $EUID -eq 0 ]]; then
   echo "Error: This script should not be run as root. Use sudo when needed." >&2
   exit 1
fi

# Check if sudo is available
if ! command -v sudo &> /dev/null; then
    echo "Error: sudo is required but not installed." >&2
    exit 1
fi

echo "Setting up bulletproof auto-updates for Raspberry Pi..."
echo "This will configure your Pi to automatically update everything and reboot when needed."
echo

# Install unattended upgrades
echo "Installing unattended-upgrades..."
sudo apt update
sudo apt install -y unattended-upgrades apt-listchanges

# Configure periodic updates
echo "Configuring automatic update schedule..."
cat << 'EOF' | sudo tee /etc/apt/apt.conf.d/20auto-upgrades > /dev/null
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
EOF

# Configure what to update and when to reboot
echo "Configuring update sources and reboot behavior..."
cat << 'EOF' | sudo tee /etc/apt/apt.conf.d/50unattended-upgrades > /dev/null
Unattended-Upgrade::Origins-Pattern {
    "origin=Debian,codename=${distro_codename},label=Debian";
    "origin=Debian,codename=${distro_codename},label=Debian-Security";
    "origin=Raspbian,codename=${distro_codename},label=Raspbian";
    "origin=Grafana Labs";
};

// Update everything - no package blacklist
Unattended-Upgrade::Package-Blacklist {
};

// Automatically reboot when needed (kernel updates, etc.)
Unattended-Upgrade::Automatic-Reboot "true";
Unattended-Upgrade::Automatic-Reboot-Time "03:00";

// Clean up old packages
Unattended-Upgrade::Remove-Unused-Dependencies "true";
Unattended-Upgrade::Remove-New-Unused-Dependencies "true";

// Don't interrupt running processes unnecessarily
Unattended-Upgrade::MinimalSteps "true";

// Log everything
Unattended-Upgrade::SyslogEnable "true";
Unattended-Upgrade::SyslogFacility "daemon";
EOF

# Auto-restart services after updates
echo "Configuring service restarts after updates..."
cat << 'EOF' | sudo tee /etc/apt/apt.conf.d/99auto-restart > /dev/null
DPkg::Post-Invoke {
    "if [ -d /run/systemd/system ]; then systemctl daemon-reload; fi";
    "if systemctl is-active --quiet grafana-server && dpkg -l grafana 2>/dev/null | grep -q '^ii'; then systemctl restart grafana-server; fi";
    "if systemctl is-active --quiet nginx && dpkg -l nginx 2>/dev/null | grep -q '^ii'; then systemctl restart nginx; fi";
};
EOF

# Enable and start the service
echo "Enabling unattended-upgrades service..."
sudo systemctl enable unattended-upgrades
sudo systemctl start unattended-upgrades

# Test the configuration
echo "Testing configuration..."
if sudo unattended-upgrade --dry-run; then
    echo "✓ Configuration test passed"
else
    echo "⚠ Configuration test failed - check logs"
fi

echo
echo "✓ Auto-updates configured successfully!"
echo
echo "Your Raspberry Pi will now:"
echo "  • Check for updates daily"
echo "  • Install ALL available updates automatically"
echo "  • Restart services when packages are updated"
echo "  • Reboot at 3:00 AM when kernel updates require it"
echo "  • Clean up old packages weekly"
echo "  • Log all activities to /var/log/unattended-upgrades/"
echo
echo "To check status:"
echo "  sudo systemctl status unattended-upgrades"
echo "  sudo cat /var/log/unattended-upgrades/unattended-upgrades.log"
echo
echo "To manually trigger an update check:"
echo "  sudo unattended-upgrade --dry-run"