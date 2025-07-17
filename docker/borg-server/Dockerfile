# Borg Server Container for Integration Tests
# Runs borg in server mode over SSH for client connections

FROM ubuntu:22.04

# Install SSH and required packages
RUN apt-get update && apt-get install -y \
    openssh-server \
    openssh-client \
    netcat-openbsd \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Download the exact borg binary that Arco uses (from backend/cmd/root.go)
ARG BORG_VERSION=1.4.1
ARG BORG_URL=https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc236

# Create borg user and directories in one layer
RUN groupadd -g 1000 borg && \
    useradd -m -u 1000 -g borg -s /bin/bash borg && \
    mkdir -p /home/borg/.ssh /home/borg/repositories /var/run/sshd && \
    chown -R borg:borg /home/borg && \
    chmod 700 /home/borg/.ssh && \
    chmod 755 /home/borg/repositories

# Download and install borg binary
RUN curl -L "${BORG_URL}" -o /usr/local/bin/borg && \
    chmod +x /usr/local/bin/borg && \
    chown root:root /usr/local/bin/borg && \
    ln -s /usr/local/bin/borg /usr/bin/borg

# Copy SSH authorized keys
COPY authorized_keys /home/borg/.ssh/authorized_keys
RUN chown borg:borg /home/borg/.ssh/authorized_keys && \
    chmod 600 /home/borg/.ssh/authorized_keys

# Set up SSH server and generate host keys
RUN ssh-keygen -A && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config && \
    echo "AllowUsers borg" >> /etc/ssh/sshd_config && \
    echo "LogLevel INFO" >> /etc/ssh/sshd_config && \
    echo "AddressFamily inet" >> /etc/ssh/sshd_config && \
    echo "PermitUserEnvironment yes" >> /etc/ssh/sshd_config && \
    echo "borg:test123" | chpasswd && \
    echo "PATH=/usr/local/bin:/usr/bin:/bin" > /home/borg/.ssh/environment && \
    chown borg:borg /home/borg/.ssh/environment

# Create minimal startup script inline
RUN echo '#!/bin/bash' > /usr/local/bin/start-borg-server.sh && \
    echo 'set -e' >> /usr/local/bin/start-borg-server.sh && \
    echo 'echo "🚀 Starting Borg Server..."' >> /usr/local/bin/start-borg-server.sh && \
    echo 'echo "🔧 Borg version: $(/usr/local/bin/borg --version)"' >> /usr/local/bin/start-borg-server.sh && \
    echo 'echo "📂 Repository path: /home/borg/repositories"' >> /usr/local/bin/start-borg-server.sh && \
    echo 'echo "👤 SSH user: borg"' >> /usr/local/bin/start-borg-server.sh && \
    echo '# Check authorized_keys permissions (read-only mount)' >> /usr/local/bin/start-borg-server.sh && \
    echo 'if [ -f /home/borg/.ssh/authorized_keys ]; then' >> /usr/local/bin/start-borg-server.sh && \
    echo '    echo "✅ authorized_keys file is available"' >> /usr/local/bin/start-borg-server.sh && \
    echo 'else' >> /usr/local/bin/start-borg-server.sh && \
    echo '    echo "❌ authorized_keys file not found"' >> /usr/local/bin/start-borg-server.sh && \
    echo 'fi' >> /usr/local/bin/start-borg-server.sh && \
    echo '# Test borg binary accessibility via SSH' >> /usr/local/bin/start-borg-server.sh && \
    echo 'echo "🧪 Testing borg binary via SSH..."' >> /usr/local/bin/start-borg-server.sh && \
    echo 'if su - borg -c "borg --version" >/dev/null 2>&1; then' >> /usr/local/bin/start-borg-server.sh && \
    echo '    echo "✅ Borg binary accessible via SSH"' >> /usr/local/bin/start-borg-server.sh && \
    echo 'else' >> /usr/local/bin/start-borg-server.sh && \
    echo '    echo "❌ Borg binary NOT accessible via SSH"' >> /usr/local/bin/start-borg-server.sh && \
    echo 'fi' >> /usr/local/bin/start-borg-server.sh && \
    echo 'echo "🎯 Starting SSH server..."' >> /usr/local/bin/start-borg-server.sh && \
    echo 'exec /usr/sbin/sshd -D -e' >> /usr/local/bin/start-borg-server.sh && \
    chmod +x /usr/local/bin/start-borg-server.sh

# Expose SSH port
EXPOSE 22

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD nc -z 127.0.0.1 22 || exit 1

# Start SSH server
CMD ["/usr/local/bin/start-borg-server.sh"]