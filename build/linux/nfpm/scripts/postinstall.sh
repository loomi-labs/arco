#!/bin/bash
set -e

# Create systemd user directory if it doesn't exist
install -d /etc/systemd/user

# Create systemd user service file
cat > /etc/systemd/user/arco.service << 'EOF'
[Unit]
Description=Arco Backup Service

[Service]
Type=simple
ExecStartPre=sleep 5
ExecStart=/usr/local/bin/arco --hidden
Restart=on-failure

[Install]
WantedBy=default.target
EOF

# Enable the service globally for all users (takes effect on next login)
# Handle systems without systemd gracefully
if command -v systemctl >/dev/null 2>&1; then
    systemctl --global enable arco.service 2>/dev/null || true
    echo "Arco backup service enabled. It will start automatically on login."
    echo "To disable for current user: systemctl --user disable arco"
    echo "To disable for all users:    sudo systemctl --global disable arco"
else
    echo "Note: systemd not found. Arco service auto-start is not available."
fi

# Update desktop database for .desktop file changes
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q /usr/share/applications
fi

# Update MIME database for custom URL schemes
if command -v update-mime-database >/dev/null 2>&1; then
  update-mime-database -n /usr/share/mime
fi
