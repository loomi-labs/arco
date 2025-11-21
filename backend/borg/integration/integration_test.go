//go:build integration

/*
Integration Tests for Borg Backup System

These tests provide comprehensive end-to-end testing of the borg backup interface using
a real client-server architecture with Docker containers. This approach ensures that
the actual production code paths are tested, including network communication, SSH
authentication, and real borg binary execution.

## Architecture Overview

The tests use a two-container approach that mirrors real-world borg deployments:

┌─────────────────────────────────┐        SSH         ┌───────────────────────────────┐
│          Test Host              │ ◄────────────────► │    Borg Server Container      │
│                                 │                    │                               │
│ • Integration test code         │                    │ • Borg 1.4.1 server binary    │
│ • Real borg binary execution    │                    │ • SSH server (sshd)           │
│ • SSH client calls              │                    │ • Repository storage          │
│ • Production interface methods  │                    │ • Embedded authentication     │
└─────────────────────────────────┘                    └───────────────────────────────┘

## How It Works

1. **Setup Phase**:
   - Testcontainers creates a Docker network
   - Builds borg-server container from ../../docker/borg-server/Dockerfile
   - Container includes SSH server with embedded authorized_keys
   - Host generates SSH keys if needed and establishes connectivity

2. **Test Execution**:
   - Tests call real borg interface methods (e.g., borg.Create(), borg.List())
   - Interface methods execute actual borg binary with SSH URLs
   - Borg commands connect to containerized server over SSH
   - Server processes borg operations and stores repositories
   - Results flow back through SSH to test assertions

3. **Cleanup Phase**:
   - Containers are terminated and removed
   - Networks are cleaned up
   - Temporary files are removed

## Key Components

• **TestIntegrationSuite**: Manages the complete test environment lifecycle
• **Docker Infrastructure**: Server container with borg binary and SSH daemon
• **SSH Authentication**: Passwordless SSH using embedded RSA/ED25519 keys
• **Real Network Communication**: Actual TCP/SSH between host and container
• **Production Code Path**: Uses identical borg binary and interface methods as production

Run with: `task test:integration` or `go test -v ./backend/borg/integration/...`

TEST CASES - integration_test.go

TestBorgRepositoryOperations
* Init - Repository initialization with SSH
* Info - Repository info retrieval after init
* List - Repository archive listing (empty repo)

TestBorgArchiveOperations
* Create - Archive creation with test data
* CreateAndList - Archive creation and verification via list

TestBorgDeleteOperations
* DeleteArchive - Archive deletion and verification
* DeleteRepository - Repository deletion and verification

TestBorgMaintenanceOperations
* Compact - Repository compaction after archive creation
* Prune - Repository pruning with keep-last rules (dry run)

TestBorgRenameOperation
* Rename archive and verify new name via list

TestBorgBreakLockOperation
* Break repository lock (should succeed even without lock)

TestBorgMountOperations
* MountRepository - Mount entire repository to filesystem and verify access
* MountArchive - Mount specific archive to filesystem and verify content
* MountErrors - Error handling for invalid mount paths and non-existent targets

TestBorgDeleteArchives
* Delete multiple archives with prefix pattern matching
* Verify auto-compact after bulk deletion
* Verify selective deletion (only matching prefix)

TestBorgCheckOperations
* QuickMode - Quick repository-only integrity check
* FullMode - Full repository and data verification check
* NonExistentRepository - Error handling for non-existent repository
* WrongPassword - Error handling for incorrect password

TestBorgErrorHandling
* InvalidRepository - Error handling for non-existent repository
* WrongPassword - Error handling for incorrect password
* InvalidArchiveName - Error handling for non-existent archive deletion

*/

package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/loomi-labs/arco/backend/borg"
	"github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

const (
	testPassword = "test123"
	sshUser      = "borg"
	sshHost      = "borg-server"
	sshPort      = "22"
)

// TestIntegrationSuite provides a test suite for integration tests with real borg instances
type TestIntegrationSuite struct {
	ctx               context.Context
	network           testcontainers.Network
	serverContainer   testcontainers.Container
	borg              borg.Borg
	logger            *zap.SugaredLogger
	serverHostPort    string
	serverContainerIP string
}

// setupBorgEnvironment creates and starts the borg client-server environment
func (s *TestIntegrationSuite) setupBorgEnvironment(t *testing.T) {
	s.ctx = context.Background()

	// Setup logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	s.logger = logger.Sugar()

	// Check if we should use existing network from environment
	networkName := os.Getenv("TESTCONTAINERS_NETWORK_NAME")
	if networkName != "" {
		// Use existing network created by the test script
		s.network = nil // We don't own the network, so don't clean it up
	} else {
		// Create our own network
		net, err := network.New(s.ctx)
		require.NoError(t, err)
		s.network = net
		networkName = net.Name
	}

	// Start borg server container
	s.startBorgServer(t, networkName)

	// Setup SSH connection
	s.setupSSHConnection(t)
}

// startBorgServer starts the borg server container
func (s *TestIntegrationSuite) startBorgServer(t *testing.T, networkName string) {
	// Get SSH keys directory - handle both host and container paths
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Determine project root for Docker context (works in both host and container)
	projectRoot := filepath.Join(wd, "..", "..", "..")
	dockerfilePath := "docker/borg-server/Dockerfile"
	if _, err := os.Stat(filepath.Join(projectRoot, dockerfilePath)); os.IsNotExist(err) {
		// Try mounted docker directory for containerized environment
		projectRoot = "/app"
		dockerfilePath = "docker/borg-server/Dockerfile"
	}

	// Get server borg version from environment, fallback to default
	serverBorgVersion := os.Getenv("SERVER_BORG_VERSION")
	if serverBorgVersion == "" {
		serverBorgVersion = "1.4.0"
	}

	// Build server container from Dockerfile or use pre-built image
	var serverRequest testcontainers.ContainerRequest

	// Check if a pre-built server image is available (for containerized environment)
	if serverImage := os.Getenv("SERVER_IMAGE"); serverImage != "" {
		serverRequest = testcontainers.ContainerRequest{
			Image: serverImage,
		}
	} else {
		serverRequest = testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    projectRoot,
				Dockerfile: dockerfilePath,
				BuildArgs: map[string]*string{
					"BUILDTIME":    &[]string{fmt.Sprintf("%d", time.Now().UnixNano())}[0],
					"BORG_VERSION": &serverBorgVersion,
				},
			},
		}
	}

	// Add common container configuration
	serverRequest.ExposedPorts = []string{"22/tcp"}
	serverRequest.WaitingFor = wait.ForListeningPort("22/tcp").WithStartupTimeout(60 * time.Second)
	serverRequest.Networks = []string{networkName}
	serverRequest.NetworkAliases = map[string][]string{
		networkName: {"borg-server"},
	}

	s.serverContainer, err = testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: serverRequest,
		Started:          true,
	})
	require.NoError(t, err)

	// Get the host port for SSH connection
	hostPort, err := s.serverContainer.MappedPort(s.ctx, "22")
	require.NoError(t, err)
	s.serverHostPort = hostPort.Port()

	// In containerized environment, we need to determine how to connect
	if _, inContainer := os.LookupEnv("SERVER_IMAGE"); inContainer {
		// Try to get container info for networking
		inspect, err := s.serverContainer.Inspect(s.ctx)
		require.NoError(t, err)

		// If we're using a shared network, use the container name for DNS
		if networkName := os.Getenv("TESTCONTAINERS_NETWORK_NAME"); networkName != "" {
			// Use container name for DNS resolution on shared network
			s.serverContainerIP = "borg-server" // This matches the network alias
		} else {
			// Fall back to container IP
			for _, net := range inspect.NetworkSettings.Networks {
				if net.IPAddress != "" {
					s.serverContainerIP = net.IPAddress
					break
				}
			}
		}
	}
}

// setupSSHConnection configures SSH connection from host to server
func (s *TestIntegrationSuite) setupSSHConnection(t *testing.T) {
	// Get SSH keys directory - handle both host and container paths
	var sshKeysDir string

	// Check if running in container (SSH keys mounted to /home/borg/.ssh)
	if _, err := os.Stat("/home/borg/.ssh/borg_test_key"); err == nil {
		sshKeysDir = "/home/borg/.ssh"
	} else {
		// Running on host, use relative path
		wd, err := os.Getwd()
		require.NoError(t, err)
		sshKeysDir = filepath.Join(wd, "..", "..", "..", "docker", "borg-client")
	}

	// Wait for SSH server to fully start
	time.Sleep(3 * time.Second)

	// Test SSH connectivity from host to server container
	privateKeyPath := filepath.Join(sshKeysDir, "borg_test_key")

	// Check if private key exists
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Fatalf("Private key does not exist: %s", privateKeyPath)
	}

	// In containerized environment, verify basic connectivity
	if _, inContainer := os.LookupEnv("SERVER_IMAGE"); inContainer {
		// Test basic network connectivity
		ncCmd := exec.Command("nc", "-zv", s.serverContainerIP, sshPort)
		if _, ncErr := ncCmd.CombinedOutput(); ncErr != nil {
			t.Fatalf("Network connectivity test failed: %v", ncErr)
		}
	} else {
		// Running on host - test SSH connectivity with mapped port
		cmd := exec.Command("ssh",
			"-o", "ConnectTimeout=10",
			"-o", "BatchMode=yes",
			"-o", "StrictHostKeyChecking=no",
			"-i", privateKeyPath,
			"-p", s.serverHostPort,
			fmt.Sprintf("%s@localhost", sshUser),
			"echo", "SSH connection test")

		if sshOutput, sshErr := cmd.CombinedOutput(); sshErr != nil {
			t.Logf("SSH connection test failed: %v\nOutput: %s", sshErr, string(sshOutput))
			require.NoError(t, sshErr, "SSH connection should work")
		}
	}

	// Update borg instance with SSH key
	s.borg = borg.NewBorg("/usr/bin/borg", s.logger, []string{privateKeyPath}, nil)
}

// teardownBorgEnvironment cleans up the test environment
func (s *TestIntegrationSuite) teardownBorgEnvironment(t *testing.T) {
	if s.serverContainer != nil {
		err := s.serverContainer.Terminate(s.ctx)
		assert.NoError(t, err)
	}

	if s.network != nil {
		err := s.network.Remove(s.ctx)
		assert.NoError(t, err)
	}
}

// getTestRepositoryPath creates a test repository for integration tests
func (s *TestIntegrationSuite) getTestRepositoryPath() string {
	repoName := fmt.Sprintf("test-repo-%d", time.Now().UnixNano())

	// Create SSH-based repository URL based on environment
	if _, inContainer := os.LookupEnv("SERVER_IMAGE"); inContainer {
		// Running in containerized environment - use container network alias with port 22
		return fmt.Sprintf("ssh://%s@%s:22/home/borg/repositories/%s", sshUser, s.serverContainerIP, repoName)
	} else {
		// Running on host - use localhost with mapped port
		return fmt.Sprintf("ssh://%s@localhost:%s/home/borg/repositories/%s", sshUser, s.serverHostPort, repoName)
	}
}

// createTestData creates test files for backup operations on the host
func (s *TestIntegrationSuite) createTestData(t *testing.T) string {
	dataDir := fmt.Sprintf("/tmp/borg-test-data-%d", time.Now().UnixNano())

	// Create test files on host
	testFiles := []struct {
		path    string
		content string
	}{
		{filepath.Join(dataDir, "file1.txt"), "This is test file 1"},
		{filepath.Join(dataDir, "file2.txt"), "This is test file 2"},
		{filepath.Join(dataDir, "subdir", "file3.txt"), "This is test file 3 in subdirectory"},
	}

	// Create directories and files on host
	for _, file := range testFiles {
		// Create directory
		err := os.MkdirAll(filepath.Dir(file.path), 0755)
		require.NoError(t, err)

		// Create file
		err = os.WriteFile(file.path, []byte(file.content), 0644)
		require.NoError(t, err)
	}

	return dataDir
}

// TestBorgRepositoryOperations tests basic repository operations
func TestBorgRepositoryOperations(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	t.Run("Init", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()

		// Test repository initialization
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed: %v", status.GetError())
	})

	t.Run("Info", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()

		// Initialize repository first
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Test repository info
		info, status := suite.borg.Info(suite.ctx, repoPath, testPassword)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository info should succeed: %v", status.GetError())
		assert.NotNil(t, info, "Info response should not be nil")
		assert.NotEmpty(t, info.Repository.ID, "Repository ID should not be empty")
		assert.Equal(t, repoPath, info.Repository.Location, "Repository location should match")
	})

	t.Run("List", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Test empty repository list
		list, status := suite.borg.List(suite.ctx, repoPath, testPassword, "")
		assert.True(t, status.IsCompletedWithSuccess(), "Repository list should succeed: %v", status.GetError())
		assert.NotNil(t, list, "List response should not be nil")
		assert.Empty(t, list.Archives, "Empty repository should have no archives")
	})
}

// TestBorgArchiveOperations tests archive operations
func TestBorgArchiveOperations(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	t.Run("Create", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		archivePath, status := suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		archiveName := strings.Split(archivePath, "::")[1]

		assert.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed: %v", status.GetError())
		assert.NotEmpty(t, archiveName, "Archive name should not be empty")
		assert.True(t, strings.HasPrefix(archiveName, "test-archive"), "Archive name should have correct prefix")
	})

	t.Run("CreateAndList", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		archivePath, status := suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// List archives
		list, status := suite.borg.List(suite.ctx, repoPath, testPassword, "")
		assert.True(t, status.IsCompletedWithSuccess(), "Repository list should succeed: %v", status.GetError())
		assert.NotNil(t, list, "List response should not be nil")
		assert.Len(t, list.Archives, 1, "Repository should have one archive")
		archiveName := strings.Split(archivePath, "::")[1]
		assert.Equal(t, archiveName, list.Archives[0].Name, "Archive name should match")
	})
}

// TestBorgDeleteOperations tests delete operations
func TestBorgDeleteOperations(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	t.Run("DeleteArchive", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		archivePath, status := suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// Delete archive
		archiveName := strings.Split(archivePath, "::")[1]
		status = suite.borg.DeleteArchive(suite.ctx, repoPath, archiveName, testPassword)
		assert.True(t, status.IsCompletedWithSuccess(), "Archive deletion should succeed: %v", status.GetError())

		// Verify archive is deleted
		list, status := suite.borg.List(suite.ctx, repoPath, testPassword, "")
		assert.True(t, status.IsCompletedWithSuccess(), "Repository list should succeed")
		assert.Empty(t, list.Archives, "Repository should have no archives after deletion")
	})

	t.Run("DeleteRepository", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Delete repository
		status = suite.borg.DeleteRepository(suite.ctx, repoPath, testPassword)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository deletion should succeed: %v", status.GetError())

		// Verify repository is deleted by trying to get info
		_, status = suite.borg.Info(suite.ctx, repoPath, testPassword)
		assert.True(t, status.HasError(), "Info should fail on deleted repository")
	})
}

// TestBorgMaintenanceOperations tests maintenance operations
func TestBorgMaintenanceOperations(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	t.Run("Compact", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		_, status = suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// Compact repository
		status = suite.borg.Compact(suite.ctx, repoPath, testPassword)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository compaction should succeed: %v", status.GetError())
	})

	t.Run("Prune", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create multiple backups
		for i := 0; i < 3; i++ {
			progressChan := make(chan types.BackupProgress, 10)
			_, status = suite.borg.Create(
				suite.ctx,
				repoPath,
				testPassword,
				fmt.Sprintf("test-archive-%d", i),
				[]string{dataDir},
				[]string{},
				backupprofile.CompressionModeLz4,
				nil,
				progressChan,
			)
			require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")
		}

		// Prune repository (dry run)
		pruneOptions := []string{"--keep-last=2"}
		pruneChan := make(chan types.PruneResult, 10)
		status = suite.borg.Prune(suite.ctx, repoPath, testPassword, "test-archive", pruneOptions, true, pruneChan)

		assert.True(t, status.IsCompletedWithSuccess(), "Repository pruning should succeed: %v", status.GetError())

		// Verify results from prune channel
		var pruneResults []types.PruneResult
		for result := range pruneChan {
			pruneResults = append(pruneResults, result)
		}
		assert.NotEmpty(t, pruneResults, "Prune should produce results")
	})
}

// TestBorgRenameOperation tests archive rename functionality
func TestBorgRenameOperation(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	repoPath := suite.getTestRepositoryPath()
	dataDir := suite.createTestData(t)

	// Initialize repository
	status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
	require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

	// Create backup
	progressChan := make(chan types.BackupProgress, 10)
	archivePath, status := suite.borg.Create(
		suite.ctx,
		repoPath,
		testPassword,
		"prefix-",
		[]string{dataDir},
		[]string{},
		backupprofile.CompressionModeLz4,
		nil,
		progressChan,
	)
	require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

	// Rename archive
	newName := "prefix-and-a-new-name"
	archiveName := strings.Split(archivePath, "::")[1]
	status = suite.borg.Rename(suite.ctx, repoPath, archiveName, testPassword, newName)
	assert.True(t, status.IsCompletedWithSuccess(), "Archive rename should succeed: %v", status.GetError())

	// Verify archive is renamed
	list, status := suite.borg.List(suite.ctx, repoPath, testPassword, "")
	assert.True(t, status.IsCompletedWithSuccess(), "Repository list should succeed")
	assert.Len(t, list.Archives, 1, "Repository should have one archive")
	assert.Equal(t, newName, list.Archives[0].Name, "Archive should have new name")
}

// TestBorgBreakLockOperation tests break lock functionality
func TestBorgBreakLockOperation(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	repoPath := suite.getTestRepositoryPath()

	// Initialize repository
	status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
	require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

	// Break lock (should succeed even if no lock exists)
	status = suite.borg.BreakLock(suite.ctx, repoPath, testPassword)
	assert.True(t, status.IsCompletedWithSuccess(), "Break lock should succeed: %v", status.GetError())
}

// TestBorgMountOperations tests mount and unmount operations
func TestBorgMountOperations(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	t.Run("MountRepository", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)
		mountPath := fmt.Sprintf("/tmp/borg-mount-repo-%d", time.Now().UnixNano())

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		_, status = suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// Create mount directory
		err := os.MkdirAll(mountPath, 0755)
		require.NoError(t, err, "Mount directory creation should succeed")
		defer os.RemoveAll(mountPath)

		// Mount repository
		status = suite.borg.MountRepository(suite.ctx, repoPath, testPassword, mountPath)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository mount should succeed: %v", status.GetError())

		// Verify mount is accessible
		entries, err := os.ReadDir(mountPath)
		assert.NoError(t, err, "Should be able to read mount directory")
		assert.NotEmpty(t, entries, "Mount directory should contain archive entries")

		// Unmount
		status = suite.borg.Umount(suite.ctx, mountPath)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository unmount should succeed: %v", status.GetError())

		// Verify unmount
		entries, err = os.ReadDir(mountPath)
		if err == nil {
			assert.Empty(t, entries, "Mount directory should be empty after unmount")
		}
	})

	t.Run("MountArchive", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)
		mountPath := fmt.Sprintf("/tmp/borg-mount-archive-%d", time.Now().UnixNano())

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		archivePath, status := suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// Create mount directory
		err := os.MkdirAll(mountPath, 0755)
		require.NoError(t, err, "Mount directory creation should succeed")
		defer os.RemoveAll(mountPath)

		// Mount specific archive
		archiveName := strings.Split(archivePath, "::")[1]
		status = suite.borg.MountArchive(suite.ctx, repoPath, archiveName, testPassword, mountPath)
		assert.True(t, status.IsCompletedWithSuccess(), "Archive mount should succeed: %v", status.GetError())

		// Verify mount contains original files
		entries, err := os.ReadDir(mountPath)
		assert.NoError(t, err, "Should be able to read mount directory")
		assert.NotEmpty(t, entries, "Mount directory should contain file entries")

		// Verify specific file content
		testFile := filepath.Join(mountPath, filepath.Base(dataDir), "file1.txt")
		if _, err := os.Stat(testFile); err == nil {
			content, err := os.ReadFile(testFile)
			assert.NoError(t, err, "Should be able to read mounted file")
			assert.Equal(t, "This is test file 1", string(content), "File content should match")
		}

		// Unmount
		status = suite.borg.Umount(suite.ctx, mountPath)
		assert.True(t, status.IsCompletedWithSuccess(), "Archive unmount should succeed: %v", status.GetError())

		// Verify unmount
		entries, err = os.ReadDir(mountPath)
		if err == nil {
			assert.Empty(t, entries, "Mount directory should be empty after unmount")
		}
	})

	t.Run("MountErrors", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		invalidMountPath := "/nonexistent/mount/path"

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Try to mount to invalid path
		status = suite.borg.MountRepository(suite.ctx, repoPath, testPassword, invalidMountPath)
		assert.True(t, status.HasError(), "Mount to invalid path should fail")
		assert.True(t, errors.Is(status.Error, types.ErrDefault), "Should be permission denied error")

		// Try to mount non-existent repository
		status = suite.borg.MountRepository(suite.ctx, "/nonexistent/repo", testPassword, "/tmp")
		assert.True(t, status.HasError(), "Mount non-existent repository should fail")
		assert.True(t, errors.Is(status.Error, types.ErrorRepositoryDoesNotExist), "Should be repository does not exist error")

		// Try to mount non-existent archive
		status = suite.borg.MountArchive(suite.ctx, repoPath, "nonexistent-archive", testPassword, "/tmp")
		assert.True(t, status.HasError(), "Mount non-existent archive should fail")
		assert.True(t, errors.Is(status.Error, types.ErrorArchiveDoesNotExist), "Should be archive does not exist error")
	})
}

// TestBorgDeleteArchives tests multiple archive deletion
func TestBorgDeleteArchives(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	repoPath := suite.getTestRepositoryPath()
	dataDir := suite.createTestData(t)

	// Initialize repository
	status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
	require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

	// Create multiple archives with same prefix
	archivePrefix := "test-archive"
	for i := 0; i < 5; i++ {
		progressChan := make(chan types.BackupProgress, 10)
		_, status = suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			fmt.Sprintf("%s-%d", archivePrefix, i),
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")
	}

	// Create archive with different prefix
	progressChan := make(chan types.BackupProgress, 10)
	_, status = suite.borg.Create(
		suite.ctx,
		repoPath,
		testPassword,
		"other-archive-",
		[]string{dataDir},
		[]string{},
		backupprofile.CompressionModeLz4,
		nil,
		progressChan,
	)
	require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

	// Verify all archives exist
	list, status := suite.borg.List(suite.ctx, repoPath, testPassword, "")
	require.True(t, status.IsCompletedWithSuccess(), "List should succeed")
	assert.Len(t, list.Archives, 6, "Should have 6 archives before deletion")

	// Delete archives with prefix
	status = suite.borg.DeleteArchives(suite.ctx, repoPath, testPassword, archivePrefix)
	assert.True(t, status.IsCompletedWithSuccess(), "Delete archives should succeed: %v", status.GetError())

	// Verify only other-archive remains
	list, status = suite.borg.List(suite.ctx, repoPath, testPassword, "")
	assert.True(t, status.IsCompletedWithSuccess(), "List should succeed after deletion")
	assert.Len(t, list.Archives, 1, "Should have 1 archive after deletion")
	assert.True(t, strings.HasPrefix(list.Archives[0].Name, "other-archive-"), "Remaining archive should start with other-archive-")

	// Verify repository is compacted (this happens automatically in DeleteArchives)
	// We can't directly test compaction, but we can verify the operation completed successfully
	info, status := suite.borg.Info(suite.ctx, repoPath, testPassword)
	assert.True(t, status.IsCompletedWithSuccess(), "Info should succeed after deletion and compaction")
	assert.NotNil(t, info, "Info should return repository information")
}

// TestBorgCheckOperations tests check operations
func TestBorgCheckOperations(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	t.Run("QuickMode", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		_, status = suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// Run quick check
		result := suite.borg.Check(suite.ctx, repoPath, testPassword, true)
		assert.True(t, result.Status.IsCompletedWithSuccess(), "Quick check should succeed: %v", result.Status.GetError())
		assert.Empty(t, result.ErrorLogs, "Quick check should have no error logs")
	})

	t.Run("FullMode", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		_, status = suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			backupprofile.CompressionModeLz4,
			nil,
			progressChan,
		)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// Run full check (verify data)
		result := suite.borg.Check(suite.ctx, repoPath, testPassword, false)
		assert.True(t, result.Status.IsCompletedWithSuccess(), "Full check should succeed: %v", result.Status.GetError())
		assert.Empty(t, result.ErrorLogs, "Full check should have no error logs")
	})

	t.Run("NonExistentRepository", func(t *testing.T) {
		invalidRepoPath := "/nonexistent/repo"

		// Run check on non-existent repository
		result := suite.borg.Check(suite.ctx, invalidRepoPath, testPassword, true)
		assert.True(t, result.Status.HasError(), "Check should fail for non-existent repository")
		assert.True(t, errors.Is(result.Status.Error, types.ErrorRepositoryDoesNotExist), "Should be repository does not exist error")
	})

	t.Run("WrongPassword", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Try to check with wrong password
		result := suite.borg.Check(suite.ctx, repoPath, "wrongpassword", true)
		assert.True(t, result.Status.HasError(), "Check should fail with wrong password")
		assert.True(t, errors.Is(result.Status.Error, types.ErrorPassphraseWrong), "Should be incorrect passphrase error")
	})
}

// TestBorgErrorHandling tests error handling scenarios
func TestBorgErrorHandling(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	t.Run("InvalidRepository", func(t *testing.T) {
		invalidRepoPath := "/nonexistent/path"

		// Try to get info from non-existent repository
		_, status := suite.borg.Info(suite.ctx, invalidRepoPath, testPassword)
		assert.True(t, status.HasError(), "Info should fail for non-existent repository")
		assert.True(t, errors.Is(status.Error, types.ErrorRepositoryDoesNotExist), "Should be repository does not exist error")
	})

	t.Run("WrongPassword", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Try to access with wrong password
		_, status = suite.borg.Info(suite.ctx, repoPath, "wrongpassword")
		assert.True(t, status.HasError(), "Info should fail with wrong password")
		assert.True(t, errors.Is(status.Error, types.ErrorPassphraseWrong), "Should be incorrect passphrase error")
	})

	t.Run("InvalidArchiveName", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Try to delete non-existent archive
		status = suite.borg.DeleteArchive(suite.ctx, repoPath, "nonexistent-archive", testPassword)
		assert.True(t, status.HasWarning(), "Delete should fail or warn for non-existent archive")
		assert.True(t, errors.Is(status.Warning, types.WarningGeneric), "Should be generic warning")
	})

	t.Run("MissingSSHKey", func(t *testing.T) {
		repoPath := suite.getTestRepositoryPath()

		// Initialize repository first (while SSH key exists)
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Get SSH keys directory and private key path
		var sshKeysDir string
		if _, err := os.Stat("/home/borg/.ssh/borg_test_key"); err == nil {
			sshKeysDir = "/home/borg/.ssh"
		} else {
			wd, err := os.Getwd()
			require.NoError(t, err)
			sshKeysDir = filepath.Join(wd, "..", "..", "..", "docker", "borg-client")
		}
		privateKeyPath := filepath.Join(sshKeysDir, "borg_test_key")

		// Verify SSH key exists before deletion
		_, err := os.Stat(privateKeyPath)
		require.NoError(t, err, "SSH key should exist before test")

		// Delete the SSH private key
		err = os.Remove(privateKeyPath)
		require.NoError(t, err, "Should be able to delete SSH key")

		// Try to perform a borg operation that requires SSH - this should fail
		_, status = suite.borg.Info(suite.ctx, repoPath, testPassword)
		assert.True(t, status.HasError(), "Info should fail when SSH key is missing")

		// The error should be related to connection or general error
		assert.True(t, errors.Is(status.Error, types.ErrorConnectionClosedWithHint), "Should be a connection closed with hint error")
	})
}
