#!/bin/bash
set -e

echo "🚀 Starting Borg Server..."
echo "🔧 Borg version: $(/usr/local/bin/borg --version)"
echo "📂 Repository path: /home/borg/repositories"
echo "👤 SSH user: borg"

# Check authorized_keys permissions (read-only mount)
if [ -f /home/borg/.ssh/authorized_keys ]; then
    echo "✅ authorized_keys file is available"
else
    echo "❌ authorized_keys file not found"
fi

# Test borg binary accessibility via SSH
echo "🧪 Testing borg binary via SSH..."
if su - borg -c "borg --version" >/dev/null 2>&1; then
    echo "✅ Borg binary accessible via SSH"
else
    echo "❌ Borg binary NOT accessible via SSH"
fi

echo "🎯 Starting SSH server..."
exec /usr/sbin/sshd -D -e