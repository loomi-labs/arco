#!/bin/bash
# Matrix integration test script for Borg backup testing
# Supports different client/server versions and operating systems

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
CLIENT_VERSION=""
SERVER_VERSION=""
OS_TYPE=""
BASE_IMAGE="ubuntu:24.04"
VERBOSE=false

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

print_usage() {
    echo "Usage: $0 --client-version VERSION --server-version VERSION --os OS [OPTIONS]"
    echo ""
    echo "Required arguments:"
    echo "  --client-version VERSION     Borg client version (e.g., 1.4.0)"
    echo "  --server-version VERSION     Borg server version (e.g., 1.4.0)"
    echo "  --os OS                      Operating system (ubuntu/macos)"
    echo ""
    echo "Optional arguments:"
    echo "  --base-image IMAGE           Docker base image for Linux (default: ubuntu:24.04)"
    echo "  -v, --verbose                Enable verbose test output"
    echo "  -h, --help                   Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --client-version 1.4.0 --server-version 1.4.0 --os ubuntu"
    echo "  $0 --client-version 1.2.8 --server-version 1.2.8 --os macos --verbose"
    echo "  $0 --client-version 1.4.1 --server-version 1.4.1 --os ubuntu --base-image ubuntu:24.04"
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

# Function to determine the appropriate Dockerfile based on base image
get_client_dockerfile() {
    local base_image="$1"
    
    case "$base_image" in
        "ubuntu:20.04")
            echo "ubuntu-20.04.Dockerfile"
            ;;
        "ubuntu:22.04")
            echo "ubuntu-22.04.Dockerfile"
            ;;
        "ubuntu:24.04")
            echo "ubuntu-24.04.Dockerfile"
            ;;
        *)
            log_error "Unsupported base image: $base_image"
            log_error "Supported base images: ubuntu:20.04, ubuntu:22.04, ubuntu:24.04"
            exit 1
            ;;
    esac
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --client-version)
            CLIENT_VERSION="$2"
            shift 2
            ;;
        --server-version)
            SERVER_VERSION="$2"
            shift 2
            ;;
        --os)
            OS_TYPE="$2"
            shift 2
            ;;
        --base-image)
            BASE_IMAGE="$2"
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

# Validate required arguments
if [ -z "$CLIENT_VERSION" ]; then
    log_error "Client version is required"
    print_usage
    exit 1
fi

if [ -z "$SERVER_VERSION" ]; then
    log_error "Server version is required"
    print_usage
    exit 1
fi

if [ -z "$OS_TYPE" ]; then
    log_error "OS type is required"
    print_usage
    exit 1
fi

# Validate OS type
if [ "$OS_TYPE" != "ubuntu" ] && [ "$OS_TYPE" != "macos" ]; then
    log_error "OS type must be 'ubuntu' or 'macos'"
    print_usage
    exit 1
fi

log_info "Starting matrix integration tests"
log_info "Client version: ${CLIENT_VERSION}"
log_info "Server version: ${SERVER_VERSION}"
log_info "OS type: ${OS_TYPE}"
if [ "$OS_TYPE" = "ubuntu" ]; then
    log_info "Base image: ${BASE_IMAGE}"
fi

# Run tests based on OS type
if [ "$OS_TYPE" = "ubuntu" ]; then
    log_info "Running Ubuntu Docker-based tests"
    
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
    
    # Container and image names
    CLIENT_IMAGE="borg-client:${CLIENT_VERSION}"
    SERVER_IMAGE="borg-server:${SERVER_VERSION}"
    CLIENT_CONTAINER="borg-client-test"
    NETWORK_NAME="borg-integration-test-network"
    
    # Cleanup function
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
    
    # Determine which Dockerfile to use for the client
    CLIENT_DOCKERFILE=$(get_client_dockerfile "${BASE_IMAGE}")
    
    # Build client container
    log_info "Building client container with base image: ${BASE_IMAGE}"
    log_info "Using Dockerfile: ${CLIENT_DOCKERFILE}"
    docker build \
        --build-arg CLIENT_BORG_VERSION="${CLIENT_VERSION}" \
        --build-arg SERVER_BORG_VERSION="${SERVER_VERSION}" \
        -t "${CLIENT_IMAGE}" \
        -f "${PROJECT_ROOT}/docker/borg-client/${CLIENT_DOCKERFILE}" \
        "${PROJECT_ROOT}" || {
        log_error "Failed to build client container"
        exit 1
    }
    
    log_success "Client container built successfully"
    
    # Build server container for integration tests
    log_info "Building server container with base image: ${BASE_IMAGE}"
    docker build \
        --build-arg BASE_IMAGE="${BASE_IMAGE}" \
        --build-arg BORG_VERSION="${SERVER_VERSION}" \
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
    log_info "Creating Docker network: ${NETWORK_NAME}"
    docker network create "${NETWORK_NAME}" 2>/dev/null || log_info "Network already exists"
    
    # Prepare test command
    TEST_ARGS="-test.v"
    if [ "$VERBOSE" = true ]; then
        TEST_ARGS="${TEST_ARGS} -test.run=TestBorgRepositoryOperations"
    fi
    
    # Get current user's docker group ID
    DOCKER_GID=$(stat -c '%g' /var/run/docker.sock)
    
    # Run integration tests in container
    log_info "Running integration tests..."
    docker run \
        --rm \
        --name "${CLIENT_CONTAINER}" \
        --network "${NETWORK_NAME}" \
        --privileged \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v "${PROJECT_ROOT}/docker:/app/docker:ro" \
        -e CLIENT_BORG_VERSION="${CLIENT_VERSION}" \
        -e SERVER_BORG_VERSION="${SERVER_VERSION}" \
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
    
elif [ "$OS_TYPE" = "macos" ]; then
    log_info "Running macOS native tests"
    
    # Check if we're actually on macOS
    if [[ "$OSTYPE" != "darwin"* ]]; then
        log_error "macOS tests can only run on macOS systems"
        exit 1
    fi
    
    # Install Borg if not already installed
    if ! command -v borg &> /dev/null; then
        log_info "Installing Borg via Homebrew"
        brew install borgbackup
    fi
    
    # Verify Borg version
    INSTALLED_VERSION=$(borg --version | cut -d' ' -f2)
    log_info "Installed Borg version: ${INSTALLED_VERSION}"
    
    # Check if we need to install a specific version
    if [ "$INSTALLED_VERSION" != "$CLIENT_VERSION" ]; then
        log_warning "Installed Borg version ($INSTALLED_VERSION) differs from requested ($CLIENT_VERSION)"
        log_warning "Using installed version for testing"
    fi
    
    # Setup SSH server for testing
    log_info "Setting up SSH server for testing"
    
    # Enable SSH server if not already enabled
    if ! sudo systemsetup -getremotelogin | grep -q "On"; then
        log_info "Enabling SSH server"
        sudo systemsetup -setremotelogin on
    fi
    
    # Create borg user for testing
    if ! id -u borg &> /dev/null; then
        log_info "Creating borg user for testing"
        sudo dscl . -create /Users/borg
        sudo dscl . -create /Users/borg UserShell /bin/bash
        sudo dscl . -create /Users/borg RealName "Borg Test User"
        sudo dscl . -create /Users/borg UniqueID 1000
        sudo dscl . -create /Users/borg PrimaryGroupID 1000
        sudo dscl . -create /Users/borg NFSHomeDirectory /Users/borg
        sudo dscl . -passwd /Users/borg test123
        
        # Create directories
        sudo mkdir -p /Users/borg/.ssh
        sudo mkdir -p /Users/borg/repositories
        
        # Copy SSH key
        if [ -f "${PROJECT_ROOT}/docker/borg-client/borg_test_key.pub" ]; then
            sudo cp "${PROJECT_ROOT}/docker/borg-client/borg_test_key.pub" /Users/borg/.ssh/authorized_keys
        else
            log_error "SSH public key not found at ${PROJECT_ROOT}/docker/borg-client/borg_test_key.pub"
            exit 1
        fi
        
        # Set permissions
        sudo chown -R borg:staff /Users/borg
        sudo chmod 700 /Users/borg/.ssh
        sudo chmod 600 /Users/borg/.ssh/authorized_keys
    fi
    
    # Build integration test binary
    log_info "Building integration test binary"
    cd "${PROJECT_ROOT}"
    CGO_ENABLED=1 go test -c -o integration-tests ./backend/borg/integration
    
    # Set environment variables for macOS testing
    export CLIENT_BORG_VERSION="$CLIENT_VERSION"
    export SERVER_BORG_VERSION="$SERVER_VERSION"
    export BORG_SSH_HOST="localhost"
    export BORG_SSH_PORT="22"
    export BORG_SSH_USER="borg"
    export BORG_SSH_KEY_PATH="${PROJECT_ROOT}/docker/borg-client/borg_test_key"
    
    # Run integration tests
    log_info "Running integration tests"
    if [ "$VERBOSE" = true ]; then
        ./integration-tests -test.v
    else
        ./integration-tests
    fi
    
    # Cleanup<
    log_info "Cleaning up test binary"
    rm -f integration-tests
fi

log_success "Matrix integration tests completed successfully"
log_info "Client version: ${CLIENT_VERSION}"
log_info "Server version: ${SERVER_VERSION}"
log_info "OS type: ${OS_TYPE}"
if [ "$OS_TYPE" = "ubuntu" ]; then
    log_info "Base image: ${BASE_IMAGE}"
fi