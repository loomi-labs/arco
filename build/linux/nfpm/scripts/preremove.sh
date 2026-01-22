#!/bin/bash
# Stop service for all users (best effort, don't fail)
for user_runtime in /run/user/*; do
    uid=$(basename "$user_runtime")
    systemctl --user -M "$uid@" stop arco 2>/dev/null || true
    systemctl --user -M "$uid@" disable arco 2>/dev/null || true
done
exit 0
