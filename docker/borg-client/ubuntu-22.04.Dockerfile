# Borg Client Container for Integration Tests - Ubuntu 22.04
# Extends main Dockerfile with Ubuntu 22.04 specific configuration

# Import from main Dockerfile's builder stage
FROM docker.io/library/golang:1.24-bullseye AS builder

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

# Build integration test binary
RUN CGO_ENABLED=1 GOOS=linux go test -tags=integration -c -o /integration-tests ./backend/borg/integration

# Ubuntu 22.04 specific runtime environment
FROM ubuntu:22.04

# Global build arguments
ARG CLIENT_BORG_VERSION=1.4.0
ARG SERVER_BORG_VERSION=1.4.0

# Install required packages including Docker client and FUSE for mount operations
# Ubuntu 22.04 uses libfuse2
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    openssh-client \
    docker.io \
    netcat-openbsd \
    dnsutils \
    iputils-ping \
    fuse \
    libfuse2 \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Create borg user and directories (simplified for Ubuntu 22.04)
RUN groupadd -g 1000 borg && \
    useradd -m -u 1000 -g borg -s /bin/bash borg && \
    usermod -aG docker borg

# Download and install borg binary with version-specific URL
RUN if [ "${CLIENT_BORG_VERSION}" = "1.4.1" ]; then \
        BORG_BINARY="borg-linux-glibc236"; \
    elif [ "${CLIENT_BORG_VERSION}" = "1.4.0" ]; then \
        BORG_BINARY="borg-linux-glibc231"; \
    else \
        echo "Unsupported Borg version: ${CLIENT_BORG_VERSION}" && exit 1; \
    fi && \
    curl -L "https://github.com/borgbackup/borg/releases/download/${CLIENT_BORG_VERSION}/${BORG_BINARY}" -o /usr/local/bin/borg && \
    chmod +x /usr/local/bin/borg && \
    chown root:root /usr/local/bin/borg && \
    ln -s /usr/local/bin/borg /usr/bin/borg

# Create SSH directory
RUN mkdir -p /home/borg/.ssh && \
    chown -R borg:borg /home/borg/.ssh && \
    chmod 700 /home/borg/.ssh

# Copy SSH keys
COPY docker/borg-client/borg_test_key /home/borg/.ssh/borg_test_key
RUN chown borg:borg /home/borg/.ssh/borg_test_key && \
    chmod 600 /home/borg/.ssh/borg_test_key

# Create working directories for repositories and test data
RUN mkdir -p /tmp/borg-repos /tmp/test-data && \
    chown -R borg:borg /tmp/borg-repos /tmp/test-data

# Copy entrypoint script
COPY docker/borg-client/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Verify borg installation
RUN /usr/local/bin/borg --version

# Copy integration test binary from builder stage
COPY --from=builder /integration-tests /usr/local/bin/integration-tests
RUN chmod +x /usr/local/bin/integration-tests

# Switch to borg user
USER borg
WORKDIR /home/borg

# Set environment variables
ENV CLIENT_BORG_VERSION=${CLIENT_BORG_VERSION}
ENV SERVER_BORG_VERSION=${SERVER_BORG_VERSION}
ENV TESTCONTAINERS_RYUK_DISABLED=true
ENV TESTCONTAINERS_CHECKS_DISABLE=true
ENV DOCKER_HOST=unix:///var/run/docker.sock

# Default entrypoint with command
CMD ["/usr/local/bin/entrypoint.sh"]