#!/bin/bash
set -e

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
systemctl --global enable arco.service

echo "Arco backup service enabled. It will start automatically on login."
echo "To disable for current user: systemctl --user disable arco"
echo "To disable for all users:    sudo systemctl --global disable arco"

# Update desktop database for .desktop file changes
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q /usr/share/applications
fi

# Update MIME database for custom URL schemes
if command -v update-mime-database >/dev/null 2>&1; then
  update-mime-database -n /usr/share/mime
fi
