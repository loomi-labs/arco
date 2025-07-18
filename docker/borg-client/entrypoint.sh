#!/bin/bash
set -e

# Check Docker socket access
if [ -S /var/run/docker.sock ]; then
    echo "✅ Docker socket available at /var/run/docker.sock"
    docker version --format '{{.Server.Version}}' || echo "⚠️  Docker daemon not accessible"
else
    echo "❌ Docker socket not found at /var/run/docker.sock"
    echo "   Please mount Docker socket: -v /var/run/docker.sock:/var/run/docker.sock"
    exit 1
fi

# Verify SSH keys are available
if [ -r /home/borg/.ssh/borg_test_key ]; then
    echo "✅ SSH keys are available"
else
    echo "❌ SSH private key not found or not readable"
    exit 1
fi

# Show network configuration if available
if [ -n "${TESTCONTAINERS_NETWORK_NAME}" ]; then
    echo "🌐 Using Docker network: ${TESTCONTAINERS_NETWORK_NAME}"
fi

# Run integration tests
echo "🚀 Running integration tests..."
echo "   Client Borg: ${CLIENT_BORG_VERSION}"
echo "   Server Borg: ${SERVER_BORG_VERSION}"

exec /usr/local/bin/integration-tests -test.v