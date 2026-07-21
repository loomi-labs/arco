# Borg Client Container for Integration Tests - Ubuntu 20.04

# Must stay on bullseye (glibc 2.31) so the CGO integration-test binary runs on
# the ubuntu:20.04 runtime below. No golang:1.26-bullseye image exists, so we keep
# GOTOOLCHAIN=auto to fetch the go.mod-required toolchain on top of the bullseye base.
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

# Download dependencies (auto-download the go.mod-required toolchain on bullseye)
RUN GOTOOLCHAIN=auto go mod download

# Copy source code (only backend needed for integration tests)
COPY backend/ ./backend/

# Build integration test binary (auto toolchain on bullseye for glibc 2.31 compat)
RUN GOTOOLCHAIN=auto CGO_ENABLED=1 GOOS=linux go test -tags=integration -c -o /integration-tests ./backend/borg/integration

# Build minimal arco binary for borg-url detection (no CGO needed)
RUN GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux go build -tags integration -o /arco-cli ./backend/cmd/integration

# Ubuntu 20.04 specific runtime environment
FROM ubuntu:20.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC

# Re-declare build arguments
ARG CLIENT_BORG_VERSION=1.4.5
ARG SERVER_BORG_VERSION=1.4.5

# Install required packages. Ship both FUSE generations: FUSE 2 (fusermount +
# libfuse.so.2) for llfuse-based Borg builds and FUSE 3 (fusermount3 +
# libfuse.so.3) for the pyfuse3-based builds used since Borg 1.4.3.
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    openssh-client \
    docker.io \
    netcat-openbsd \
    dnsutils \
    iputils-ping \
    fuse3 \
    libfuse2 \
    libfuse3-3 \
    tzdata \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Create borg user and directories (simplified for Ubuntu 20.04)
RUN groupadd -g 1000 borg && \
    useradd -m -u 1000 -g borg -s /bin/bash borg && \
    usermod -aG docker borg && \
    (getent group fuse || groupadd fuse) && \
    usermod -aG fuse borg

# Copy Arco binary for borg-url detection (must be before borg install)
COPY --from=builder /arco-cli /usr/local/bin/arco-cli
RUN chmod +x /usr/local/bin/arco-cli

# Download and install borg binary using dynamic URL detection
RUN BORG_URL=$(/usr/local/bin/arco-cli --show-borg-url) && \
    if [ -z "$BORG_URL" ]; then \
        echo "Error: Failed to detect Borg URL" >&2; \
        exit 1; \
    fi && \
    echo "Detected Borg URL for this system: $BORG_URL" && \
    curl -L "$BORG_URL" -o /usr/local/bin/borg && \
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