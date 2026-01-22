#!/bin/bash
# Disable the service globally before removing the unit file
if command -v systemctl >/dev/null 2>&1; then
    systemctl --global disable arco.service 2>/dev/null || true
    systemctl --global daemon-reload 2>/dev/null || true
fi
rm -f /etc/systemd/user/arco.service
exit 0
