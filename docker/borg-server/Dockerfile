# Borg Server Container for Integration Tests
# Runs borg in server mode over SSH for client connections

FROM ubuntu:22.04

# Build arguments for borg version
ARG BORG_VERSION=1.4.1

# Install SSH and required packages
RUN apt-get update && apt-get install -y \
    openssh-server \
    openssh-client \
    netcat-openbsd \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create borg user and directories
RUN groupadd -g 1000 borg && \
    useradd -m -u 1000 -g borg -s /bin/bash borg && \
    mkdir -p /home/borg/.ssh /home/borg/repositories /var/run/sshd && \
    chown -R borg:borg /home/borg && \
    chmod 700 /home/borg/.ssh && \
    chmod 755 /home/borg/repositories

# Download and install borg binary with version-specific URL
RUN if [ "${BORG_VERSION}" = "1.4.1" ]; then \
        BORG_BINARY="borg-linux-glibc231"; \
    elif [ "${BORG_VERSION}" = "1.4.0" ]; then \
        BORG_BINARY="borg-linux-glibc236"; \
    else \
        echo "Unsupported Borg version: ${BORG_VERSION}" && exit 1; \
    fi && \
    curl -L "https://github.com/borgbackup/borg/releases/download/${BORG_VERSION}/${BORG_BINARY}" -o /usr/local/bin/borg && \
    chmod +x /usr/local/bin/borg && \
    chown root:root /usr/local/bin/borg && \
    ln -s /usr/local/bin/borg /usr/bin/borg

# Copy SSH authorized keys
COPY authorized_keys /home/borg/.ssh/authorized_keys
RUN chown borg:borg /home/borg/.ssh/authorized_keys && \
    chmod 600 /home/borg/.ssh/authorized_keys

# Configure SSH server
RUN ssh-keygen -A && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config && \
    echo "AllowUsers borg" >> /etc/ssh/sshd_config && \
    echo "LogLevel INFO" >> /etc/ssh/sshd_config && \
    echo "AddressFamily inet" >> /etc/ssh/sshd_config && \
    echo "PermitUserEnvironment yes" >> /etc/ssh/sshd_config && \
    echo "borg:test123" | chpasswd && \
    echo "PATH=/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin" > /home/borg/.ssh/environment && \
    chown borg:borg /home/borg/.ssh/environment && \
    chmod 600 /home/borg/.ssh/environment

# Copy and setup entrypoint script
COPY entrypoint.sh /usr/local/bin/start-borg-server.sh
RUN chmod +x /usr/local/bin/start-borg-server.sh

# Expose SSH port
EXPOSE 22

# Health check to ensure SSH server is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD nc -z 127.0.0.1 22 || exit 1

# Start the borg server
CMD ["/usr/local/bin/start-borg-server.sh"]