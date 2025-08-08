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

# Check if SSH daemon can start
echo "🔍 Checking SSH configuration..."
/usr/sbin/sshd -t
if [ $? -eq 0 ]; then
    echo "✅ SSH configuration is valid"
else
    echo "❌ SSH configuration has errors"
    exit 1
fi

# Check SSH daemon status before starting
echo "📋 SSH daemon status check..."
ps aux | grep sshd || echo "No SSH processes running yet"

echo "🚀 Starting SSH daemon in foreground mode..."
exec /usr/sbin/sshd -D -e