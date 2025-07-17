//go:build integration

package borg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/loomi-labs/arco/backend/borg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
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
	ctx             context.Context
	network         testcontainers.Network
	serverContainer testcontainers.Container
	borg            Borg
	logger          *zap.SugaredLogger
	serverHostPort  string
}

// setupBorgEnvironment creates and starts the borg client-server environment
func (s *TestIntegrationSuite) setupBorgEnvironment(t *testing.T) {
	s.ctx = context.Background()

	// Setup logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	s.logger = logger.Sugar()

	// Create a network for the containers with unique name
	networkName := fmt.Sprintf("borg-test-network-%d", time.Now().UnixNano())
	networkRequest := testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name: networkName,
		},
	}

	network, err := testcontainers.GenericNetwork(s.ctx, networkRequest)
	require.NoError(t, err)
	s.network = network

	// Start borg server container
	s.startBorgServer(t, networkName)

	// Setup SSH connection
	s.setupSSHConnection(t)
}

// startBorgServer starts the borg server container
func (s *TestIntegrationSuite) startBorgServer(t *testing.T, networkName string) {
	// Get SSH keys directory
	wd, err := os.Getwd()
	require.NoError(t, err)
	sshKeysDir := filepath.Join(wd, "..", "..", "docker", "ssh-keys")

	// Generate SSH keys if needed
	s.generateSSHKeys(t, sshKeysDir)

	// Build server container from Dockerfile
	serverRequest := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    filepath.Join(wd, "..", "..", "docker", "borg-server"),
			Dockerfile: "Dockerfile",
			BuildArgs: map[string]*string{
				"BUILDTIME": &[]string{fmt.Sprintf("%d", time.Now().UnixNano())}[0],
			},
		},
		ExposedPorts: []string{"22/tcp"},
		WaitingFor:   wait.ForListeningPort("22/tcp").WithStartupTimeout(60 * time.Second),
		Networks:     []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"borg-server"},
		},
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
}

// generateSSHKeys generates SSH keys if they don't exist
func (s *TestIntegrationSuite) generateSSHKeys(t *testing.T, sshKeysDir string) {
	keyGenScript := filepath.Join(sshKeysDir, "generate-keys.sh")

	// Check if keys already exist
	privateKey := filepath.Join(sshKeysDir, "borg_test_key")
	if _, err := os.Stat(privateKey); os.IsNotExist(err) {
		// Generate keys
		cmd := exec.Command("/bin/bash", keyGenScript)
		cmd.Dir = sshKeysDir
		err := cmd.Run()
		require.NoError(t, err)
	}
}

// setupSSHConnection configures SSH connection from host to server
func (s *TestIntegrationSuite) setupSSHConnection(t *testing.T) {
	// Get SSH keys directory
	wd, err := os.Getwd()
	require.NoError(t, err)
	sshKeysDir := filepath.Join(wd, "..", "..", "docker", "ssh-keys")

	// Generate SSH keys if needed
	s.generateSSHKeys(t, sshKeysDir)

	// Wait for SSH server to fully start
	time.Sleep(3 * time.Second)

	// Test SSH connectivity from host to server container
	privateKeyPath := filepath.Join(sshKeysDir, "borg_test_key")

	// Debug: Print SSH connection details
	t.Logf("SSH connection details:")
	t.Logf("  Host: localhost:%s", s.serverHostPort)
	t.Logf("  User: %s", sshUser)
	t.Logf("  Private key: %s", privateKeyPath)

	// Check if private key exists
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Fatalf("Private key does not exist: %s", privateKeyPath)
	}

	cmd := exec.Command("ssh",
		"-o", "ConnectTimeout=10",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		"-i", privateKeyPath,
		"-p", s.serverHostPort,
		fmt.Sprintf("%s@localhost", sshUser),
		"echo", "SSH connection test")

	// Debug: Print command and capture output
	t.Logf("Running SSH command: %s", cmd.String())
	sshOutput, sshErr := cmd.CombinedOutput()
	if sshErr != nil {
		t.Logf("SSH command failed with output: %s", string(sshOutput))
	}
	require.NoError(t, sshErr, "SSH connection should work")

	// Update borg instance with SSH key
	s.borg = NewBorg("/usr/bin/borg", s.logger, []string{privateKeyPath}, nil)
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

// createTestRepository creates a test repository for integration tests
func (s *TestIntegrationSuite) createTestRepository(t *testing.T) string {
	repoName := fmt.Sprintf("test-repo-%d", time.Now().UnixNano())

	// Create SSH-based repository URL with host port
	repoPath := fmt.Sprintf("ssh://%s@localhost:%s/home/borg/repositories/%s", sshUser, s.serverHostPort, repoName)

	// Don't pre-create the repository directory - let borg init create it with proper permissions
	// This prevents permission issues where manually created directories have incorrect ownership

	return repoPath
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
		repoPath := suite.createTestRepository(t)

		// Test repository initialization
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed: %v", status.GetError())
	})

	t.Run("Info", func(t *testing.T) {
		repoPath := suite.createTestRepository(t)

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
		repoPath := suite.createTestRepository(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Test empty repository list
		list, status := suite.borg.List(suite.ctx, repoPath, testPassword)
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
		repoPath := suite.createTestRepository(t)
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		archiveName, status := suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			progressChan,
		)
		close(progressChan)

		assert.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed: %v", status.GetError())
		assert.NotEmpty(t, archiveName, "Archive name should not be empty")
		assert.True(t, strings.HasPrefix(archiveName, "test-archive"), "Archive name should have correct prefix")
	})

	t.Run("CreateAndList", func(t *testing.T) {
		repoPath := suite.createTestRepository(t)
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		archiveName, status := suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			progressChan,
		)
		close(progressChan)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// List archives
		list, status := suite.borg.List(suite.ctx, repoPath, testPassword)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository list should succeed: %v", status.GetError())
		assert.NotNil(t, list, "List response should not be nil")
		assert.Len(t, list.Archives, 1, "Repository should have one archive")
		assert.Equal(t, archiveName, list.Archives[0].Name, "Archive name should match")
	})
}

// TestBorgDeleteOperations tests delete operations
func TestBorgDeleteOperations(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	t.Run("DeleteArchive", func(t *testing.T) {
		repoPath := suite.createTestRepository(t)
		dataDir := suite.createTestData(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Create backup
		progressChan := make(chan types.BackupProgress, 10)
		archiveName, status := suite.borg.Create(
			suite.ctx,
			repoPath,
			testPassword,
			"test-archive",
			[]string{dataDir},
			[]string{},
			progressChan,
		)
		close(progressChan)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// Delete archive
		status = suite.borg.DeleteArchive(suite.ctx, repoPath, archiveName, testPassword)
		assert.True(t, status.IsCompletedWithSuccess(), "Archive deletion should succeed: %v", status.GetError())

		// Verify archive is deleted
		list, status := suite.borg.List(suite.ctx, repoPath, testPassword)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository list should succeed")
		assert.Empty(t, list.Archives, "Repository should have no archives after deletion")
	})

	t.Run("DeleteRepository", func(t *testing.T) {
		repoPath := suite.createTestRepository(t)

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
		repoPath := suite.createTestRepository(t)
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
			progressChan,
		)
		close(progressChan)
		require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

		// Compact repository
		status = suite.borg.Compact(suite.ctx, repoPath, testPassword)
		assert.True(t, status.IsCompletedWithSuccess(), "Repository compaction should succeed: %v", status.GetError())
	})

	t.Run("Prune", func(t *testing.T) {
		repoPath := suite.createTestRepository(t)
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
				progressChan,
			)
			close(progressChan)
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

	repoPath := suite.createTestRepository(t)
	dataDir := suite.createTestData(t)

	// Initialize repository
	status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
	require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

	// Create backup
	progressChan := make(chan types.BackupProgress, 10)
	archiveName, status := suite.borg.Create(
		suite.ctx,
		repoPath,
		testPassword,
		"original-archive",
		[]string{dataDir},
		[]string{},
		progressChan,
	)
	close(progressChan)
	require.True(t, status.IsCompletedWithSuccess(), "Archive creation should succeed")

	// Rename archive
	newName := "renamed-archive"
	status = suite.borg.Rename(suite.ctx, repoPath, archiveName, testPassword, newName)
	assert.True(t, status.IsCompletedWithSuccess(), "Archive rename should succeed: %v", status.GetError())

	// Verify archive is renamed
	list, status := suite.borg.List(suite.ctx, repoPath, testPassword)
	assert.True(t, status.IsCompletedWithSuccess(), "Repository list should succeed")
	assert.Len(t, list.Archives, 1, "Repository should have one archive")
	assert.Equal(t, newName, list.Archives[0].Name, "Archive should have new name")
}

// TestBorgBreakLockOperation tests break lock functionality
func TestBorgBreakLockOperation(t *testing.T) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(t)
	defer suite.teardownBorgEnvironment(t)

	repoPath := suite.createTestRepository(t)

	// Initialize repository
	status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
	require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

	// Break lock (should succeed even if no lock exists)
	status = suite.borg.BreakLock(suite.ctx, repoPath, testPassword)
	assert.True(t, status.IsCompletedWithSuccess(), "Break lock should succeed: %v", status.GetError())
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
	})

	t.Run("WrongPassword", func(t *testing.T) {
		repoPath := suite.createTestRepository(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Try to access with wrong password
		_, status = suite.borg.Info(suite.ctx, repoPath, "wrongpassword")
		assert.True(t, status.HasError(), "Info should fail with wrong password")
	})

	t.Run("InvalidArchiveName", func(t *testing.T) {
		repoPath := suite.createTestRepository(t)

		// Initialize repository
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		require.True(t, status.IsCompletedWithSuccess(), "Repository initialization should succeed")

		// Try to delete non-existent archive
		status = suite.borg.DeleteArchive(suite.ctx, repoPath, "nonexistent-archive", testPassword)
		assert.True(t, status.HasError(), "Delete should fail for non-existent archive")
	})
}

// BenchmarkBorgOperations provides benchmarks for borg operations
func BenchmarkBorgOperations(b *testing.B) {
	suite := &TestIntegrationSuite{}
	suite.setupBorgEnvironment(&testing.T{})
	defer suite.teardownBorgEnvironment(&testing.T{})

	b.Run("Init", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			repoPath := suite.createTestRepository(&testing.T{})
			status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
			if !status.IsCompletedWithSuccess() {
				b.Fatalf("Repository initialization failed: %v", status.GetError())
			}
		}
	})

	b.Run("Info", func(b *testing.B) {
		repoPath := suite.createTestRepository(&testing.T{})
		status := suite.borg.Init(suite.ctx, repoPath, testPassword, false)
		if !status.IsCompletedWithSuccess() {
			b.Fatalf("Repository initialization failed: %v", status.GetError())
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, status := suite.borg.Info(suite.ctx, repoPath, testPassword)
			if !status.IsCompletedWithSuccess() {
				b.Fatalf("Repository info failed: %v", status.GetError())
			}
		}
	})
}
