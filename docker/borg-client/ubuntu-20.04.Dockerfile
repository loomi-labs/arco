# Borg Client Container for Integration Tests - Ubuntu 20.04
# Focused on Ubuntu 20.04 specific configuration

# Build arguments
ARG CLIENT_BORG_VERSION=1.4.0
ARG SERVER_BORG_VERSION=1.4.0

# Import builder stage from main Dockerfile (shared build context)
FROM golang:1.24-bullseye AS builder

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

# Copy source code
COPY . .

# Build integration test binary
RUN CGO_ENABLED=1 GOOS=linux go test -c -o /integration-tests ./backend/borg/integration

# Ubuntu 20.04 specific runtime environment
FROM ubuntu:20.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC

# Re-declare build arguments
ARG CLIENT_BORG_VERSION=1.4.0
ARG SERVER_BORG_VERSION=1.4.0

# Install required packages - Ubuntu 20.04 uses libfuse2
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
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create borg user and directories (simplified for Ubuntu 20.04)
RUN groupadd -g 1000 borg && \
    useradd -m -u 1000 -g borg -s /bin/bash borg && \
    usermod -aG docker borg

# Download and install borg binary (Ubuntu 20.04 uses glibc 2.31, so use glibc228 binary)
RUN curl -L "https://github.com/borgbackup/borg/releases/download/${CLIENT_BORG_VERSION}/borg-linux-glibc228" -o /usr/local/bin/borg && \
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