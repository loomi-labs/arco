#!/bin/bash
# Run integration tests in containerized environment with Docker-in-Docker
# Supports testing different borg client and server versions

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default versions
CLIENT_BORG_VERSION="${CLIENT_BORG_VERSION:-1.4.0}"
SERVER_BORG_VERSION="${SERVER_BORG_VERSION:-1.4.0}"

# Container and image names
CLIENT_IMAGE="borg-client:${CLIENT_BORG_VERSION}"
CLIENT_CONTAINER="borg-client-test"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -c, --client-version VERSION    Borg client version (default: 1.4.0)"
    echo "  -s, --server-version VERSION    Borg server version (default: 1.4.0)"
    echo "  -v, --verbose                   Enable verbose test output"
    echo "  -h, --help                      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run with default versions"
    echo "  $0 -c 1.2.8 -s 1.4.0                 # Test client 1.2.8 against server 1.4.0"
    echo "  $0 --client-version 1.4.1 --verbose  # Run with client 1.4.1 and verbose output"
    echo ""
    echo "Environment variables:"
    echo "  CLIENT_BORG_VERSION    Override default client version"
    echo "  SERVER_BORG_VERSION    Override default server version"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up containers..."
    docker rm -f "${CLIENT_CONTAINER}" 2>/dev/null || true
    if [ -n "${NETWORK_NAME:-}" ]; then
        log_info "Removing Docker network: ${NETWORK_NAME}"
        docker network rm "${NETWORK_NAME}" 2>/dev/null || true
    fi
    log_info "Cleanup completed"
}

# Set up cleanup trap
trap cleanup EXIT

# Parse command line arguments
VERBOSE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--client-version)
            CLIENT_BORG_VERSION="$2"
            shift 2
            ;;
        -s|--server-version)
            SERVER_BORG_VERSION="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Validate Docker is available
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed or not in PATH"
    exit 1
fi

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
    log_error "Docker daemon is not running"
    exit 1
fi

# Check if Docker socket is accessible
if [ ! -S /var/run/docker.sock ]; then
    log_error "Docker socket not found at /var/run/docker.sock"
    exit 1
fi

log_info "Starting containerized integration tests"
log_info "Client Borg version: ${CLIENT_BORG_VERSION}"
log_info "Server Borg version: ${SERVER_BORG_VERSION}"

# Build client container
log_info "Building client container..."
docker build \
    --build-arg CLIENT_BORG_VERSION="${CLIENT_BORG_VERSION}" \
    --build-arg SERVER_BORG_VERSION="${SERVER_BORG_VERSION}" \
    -t "${CLIENT_IMAGE}" \
    -f "${PROJECT_ROOT}/docker/borg-client/Dockerfile" \
    "${PROJECT_ROOT}" || {
    log_error "Failed to build client container"
    exit 1
}

log_success "Client container built successfully"

# Build server container for integration tests
log_info "Building server container..."
SERVER_IMAGE="borg-server:${SERVER_BORG_VERSION}"
docker build \
    --build-arg BORG_VERSION="${SERVER_BORG_VERSION}" \
    -t "${SERVER_IMAGE}" \
    -f "${PROJECT_ROOT}/docker/borg-server/Dockerfile" \
    "${PROJECT_ROOT}/docker/borg-server" || {
    log_error "Failed to build server container"
    exit 1
}

log_success "Server container built successfully"

# Clean up Docker networks to avoid subnet exhaustion
log_info "Cleaning up Docker networks..."
docker network prune -f || log_warning "Failed to clean up networks"

# Create dedicated network for integration tests
NETWORK_NAME="borg-integration-test-network"
log_info "Creating Docker network: ${NETWORK_NAME}"
docker network create "${NETWORK_NAME}" 2>/dev/null || log_info "Network already exists"

# Prepare test command
TEST_ARGS="-test.v"
if [ "$VERBOSE" = true ]; then
    # Add additional verbose flags if needed
    TEST_ARGS="${TEST_ARGS} -test.run=TestBorgRepositoryOperations"
fi

# Get current user's docker group ID
DOCKER_GID=$(stat -c '%g' /var/run/docker.sock)

# Run integration tests in container
log_info "Running integration tests..."
# Testcontainers environment variables:
# - RYUK_DISABLED: Disables Ryuk cleanup daemon for Docker-in-Docker
# - CHECKS_DISABLE: Disables startup checks in containerized environment
# - NETWORK_STRATEGY: Reuse existing network instead of creating new one
# - NETWORK_NAME: Use the pre-created network for container communication
docker run \
    --rm \
    --name "${CLIENT_CONTAINER}" \
    --network "${NETWORK_NAME}" \
    --privileged \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v "${PROJECT_ROOT}/docker:/app/docker:ro" \
    -e CLIENT_BORG_VERSION="${CLIENT_BORG_VERSION}" \
    -e SERVER_BORG_VERSION="${SERVER_BORG_VERSION}" \
    -e SERVER_IMAGE="${SERVER_IMAGE}" \
    -e TESTCONTAINERS_RYUK_DISABLED=true \
    -e TESTCONTAINERS_CHECKS_DISABLE=true \
    -e TESTCONTAINERS_NETWORK_STRATEGY=reuse \
    -e TESTCONTAINERS_NETWORK_NAME="${NETWORK_NAME}" \
    -e DOCKER_HOST=unix:///var/run/docker.sock \
    --group-add "${DOCKER_GID}" \
    "${CLIENT_IMAGE}" \
    /usr/local/bin/integration-tests ${TEST_ARGS} || {
    log_error "Integration tests failed"
    exit 1
}

log_success "Integration tests completed successfully"
log_info "Client version: ${CLIENT_BORG_VERSION}"
log_info "Server version: ${SERVER_BORG_VERSION}"