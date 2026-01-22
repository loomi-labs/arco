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

echo "Arco systemd service installed. Enable with: systemctl --user enable --now arco"
