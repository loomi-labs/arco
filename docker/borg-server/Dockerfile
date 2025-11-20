# Borg Server Container for Integration Tests
# Runs borg in server mode over SSH for client connections

# Builder stage for arco-cli
FROM docker.io/library/golang:1.25-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code (only backend needed for integration tests)
COPY backend/ ./backend/

# Build minimal arco binary for borg-url detection (no CGO needed)
RUN CGO_ENABLED=0 GOOS=linux go build -tags integration -o /arco-cli ./backend/cmd/integration

# Ubuntu 24.04 runtime environment
FROM ubuntu:24.04

# Build arguments for borg version
ARG BORG_VERSION=1.4.2

# Install SSH and required packages
RUN apt-get update && apt-get install -y \
    openssh-server \
    openssh-client \
    netcat-openbsd \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create borg user and directories (handle existing user/group conflicts)
RUN if ! getent group borg; then \
        if getent group 1000; then \
            groupadd -g 1001 borg; \
        else \
            groupadd -g 1000 borg; \
        fi; \
    fi && \
    if ! getent passwd borg; then \
        if getent passwd 1000; then \
            useradd -m -u 1001 -g borg -s /bin/bash borg; \
        else \
            useradd -m -u 1000 -g borg -s /bin/bash borg; \
        fi; \
    fi && \
    mkdir -p /home/borg/.ssh /home/borg/repositories /var/run/sshd && \
    chown -R borg:borg /home/borg && \
    chmod 700 /home/borg/.ssh && \
    chmod 755 /home/borg/repositories

# Copy Arco binary for borg-url detection (must be before borg install)
COPY --from=builder /arco-cli /usr/local/bin/arco-cli
RUN chmod +x /usr/local/bin/arco-cli

# Download and install borg binary using dynamic URL detection
RUN BORG_URL=$(/usr/local/bin/arco-cli --show-borg-url) && \
    echo "Detected Borg URL for this system: $BORG_URL" && \
    curl -L "$BORG_URL" -o /usr/local/bin/borg && \
    chmod +x /usr/local/bin/borg && \
    chown root:root /usr/local/bin/borg && \
    ln -s /usr/local/bin/borg /usr/bin/borg

# Copy SSH authorized keys
COPY docker/borg-server/authorized_keys /home/borg/.ssh/authorized_keys
RUN chown borg:borg /home/borg/.ssh/authorized_keys && \
    chmod 600 /home/borg/.ssh/authorized_keys

# Configure SSH server
RUN ssh-keygen -A && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config && \
    echo "AllowUsers borg" >> /etc/ssh/sshd_config && \
    echo "LogLevel INFO" >> /etc/ssh/sshd_config && \
    echo "AddressFamily inet" >> /etc/ssh/sshd_config && \
    echo "PermitUserEnvironment yes" >> /etc/ssh/sshd_config && \
    echo "PATH=/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin" > /home/borg/.ssh/environment && \
    chown borg:borg /home/borg/.ssh/environment && \
    chmod 600 /home/borg/.ssh/environment

# Copy and setup entrypoint script
COPY docker/borg-server/entrypoint.sh /usr/local/bin/start-borg-server.sh
RUN chmod +x /usr/local/bin/start-borg-server.sh

# Expose SSH port
EXPOSE 22

# Health check to ensure SSH server is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD nc -z 127.0.0.1 22 || exit 1

# Start the borg server
CMD ["/usr/local/bin/start-borg-server.sh"]