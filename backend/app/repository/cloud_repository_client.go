package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"github.com/charmbracelet/keygen"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/cloudrepository"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"go.uber.org/zap"
)

type CloudRepositoryStatus string

const (
	CloudRepositoryStatusSuccess         CloudRepositoryStatus = "success"
	CloudRepositoryStatusRateLimitError  CloudRepositoryStatus = "rateLimitError"
	CloudRepositoryStatusConnectionError CloudRepositoryStatus = "connectionError"
	CloudRepositoryStatusNotFound        CloudRepositoryStatus = "notFound"
	CloudRepositoryStatusError           CloudRepositoryStatus = "error"
)

// CloudRepositoryClient contains the business logic for ArcoCloud repositories
type CloudRepositoryClient struct {
	log       *zap.SugaredLogger
	db        *ent.Client
	state     *state.State
	config    *types.Config
	rpcClient arcov1connect.RepositoryServiceClient
}

// NewCloudRepositoryClient creates a new cloud repository client
func NewCloudRepositoryClient(log *zap.SugaredLogger, state *state.State, config *types.Config) *CloudRepositoryClient {
	return &CloudRepositoryClient{
		log:    log,
		state:  state,
		config: config,
	}
}

// Init initializes the client with database and RPC client
func (s *CloudRepositoryClient) Init(db *ent.Client, rpcClient arcov1connect.RepositoryServiceClient) {
	s.db = db
	s.rpcClient = rpcClient
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *CloudRepositoryClient) mustHaveDB() {
	if s.db == nil {
		panic("CloudRepositoryClient: database client is nil")
	}
}

// SSH Key Management Functions

// ensureArcoCloudSSHKey ensures that the ArcoCloud SSH key pair exists
func (s *CloudRepositoryClient) ensureArcoCloudSSHKey() error {
	keyPath := s.getArcoCloudSSHKeyPath()
	sshDir := filepath.Dir(keyPath)

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		s.log.Debugf("ArcoCloud SSH key already exists: %s", keyPath)
		return nil
	}

	// Create ssh directory if it doesn't exist
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Generate new Ed25519 key pair
	_, err := keygen.New(
		keyPath,
		keygen.WithKeyType(keygen.Ed25519),
		keygen.WithWrite(),
	)
	if err != nil {
		return fmt.Errorf("failed to generate SSH key: %w", err)
	}

	s.log.Infof("Generated new ArcoCloud SSH key: %s", keyPath)
	return nil
}

// getArcoCloudPublicKey returns the public key content for API calls
func (s *CloudRepositoryClient) getArcoCloudPublicKey() (string, error) {
	publicKeyPath := s.getArcoCloudSSHKeyPath() + ".pub"
	content, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

// getArcoCloudSSHKeyPath returns the private key path for Borg operations
func (s *CloudRepositoryClient) getArcoCloudSSHKeyPath() string {
	return filepath.Join(s.config.SSHDir, "arco_cloud_ed25519")
}

// Repository Type Detection

// IsCloudRepository checks if a repository is an ArcoCloud repository
func (s *CloudRepositoryClient) IsCloudRepository(repo *ent.Repository) bool {
	// Check if repository has a cloud repository relationship
	if repo.Edges.CloudRepository != nil {
		return repo.Edges.CloudRepository.CloudID != ""
	}
	return false
}

// Error Handling

// handleRPCError maps Connect RPC errors to CloudRepositoryStatus
func (s *CloudRepositoryClient) handleRPCError(operation string, err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeResourceExhausted:
			s.log.Warnf("Rate limit exceeded for %s: %v", operation, err)
			return fmt.Errorf("rate limit exceeded: %w", err)
		case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeAborted:
			s.log.Warnf("Connection error for %s: %v", operation, err)
			return fmt.Errorf("connection error: %w", err)
		case connect.CodeNotFound:
			s.log.Warnf("Resource not found for %s: %v", operation, err)
			return fmt.Errorf("repository not found: %w", err)
		case connect.CodeAlreadyExists:
			s.log.Warnf("Resource already exists for %s: %v", operation, err)
			return fmt.Errorf("repository already exists: %w", err)
		default:
			s.log.Errorf("Cloud repository %s failed: %v", operation, err)
			return fmt.Errorf("cloud repository operation failed: %w", err)
		}
	}
	return err
}

// AddCloudRepository creates a new ArcoCloud repository
func (s *CloudRepositoryClient) AddCloudRepository(ctx context.Context, name string, location arcov1.RepositoryLocation) (*arcov1.Repository, error) {
	// Ensure SSH key exists
	if err := s.ensureArcoCloudSSHKey(); err != nil {
		return nil, fmt.Errorf("failed to ensure SSH key: %w", err)
	}

	// Get public key for API call
	publicKey, err := s.getArcoCloudPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Call cloud service
	req := connect.NewRequest(&arcov1.AddRepositoryRequest{
		Name:     name,
		SshKey:   publicKey,
		Location: location,
	})

	resp, err := s.rpcClient.AddRepository(ctx, req)
	if err != nil {
		return nil, s.handleRPCError("add repository", err)
	}

	s.log.Infof("Created ArcoCloud repository: %s (ID: %s)", name, resp.Msg.Repository.Id)
	return resp.Msg.Repository, nil
}

// DeleteCloudRepository permanently removes an ArcoCloud repository
func (s *CloudRepositoryClient) DeleteCloudRepository(ctx context.Context, repositoryID string) error {
	s.mustHaveDB()

	// Call cloud service to delete
	req := connect.NewRequest(&arcov1.DeleteRepositoryRequest{
		RepositoryId: repositoryID,
	})

	_, err := s.rpcClient.DeleteRepository(ctx, req)
	if err != nil {
		return s.handleRPCError("delete repository", err)
	}

	// Delete local entity
	affected, err := s.db.Repository.Delete().
		Where(repository.HasCloudRepositoryWith(
			cloudrepository.CloudIDEQ(repositoryID),
		)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete local repository: %w", err)
	}

	s.log.Infof("Deleted ArcoCloud repository: %s (%d local records)", repositoryID, affected)
	return nil
}

// ListCloudRepositories retrieves all ArcoCloud repositories owned by the user
func (s *CloudRepositoryClient) ListCloudRepositories(ctx context.Context) ([]*arcov1.Repository, error) {
	// Call cloud service
	req := connect.NewRequest(&arcov1.ListRepositoriesRequest{})

	resp, err := s.rpcClient.ListRepositories(ctx, req)
	if err != nil {
		return nil, s.handleRPCError("list repositories", err)
	}
	return resp.Msg.Repositories, nil
}

// GetCloudRepository retrieves detailed information about a specific ArcoCloud repository
func (s *CloudRepositoryClient) GetCloudRepository(ctx context.Context, repositoryID string) (*arcov1.Repository, error) {
	// Call cloud service
	req := connect.NewRequest(&arcov1.GetRepositoryRequest{
		RepositoryId: repositoryID,
	})

	resp, err := s.rpcClient.GetRepository(ctx, req)
	if err != nil {
		return nil, s.handleRPCError("get repository", err)
	}
	return resp.Msg.Repository, nil
}

// AddOrReplaceSSHKey regenerates the local SSH key and updates the cloud repository
func (s *CloudRepositoryClient) AddOrReplaceSSHKey(ctx context.Context) error {
	// Remove existing SSH key
	keyPath := s.getArcoCloudSSHKeyPath()
	publicKeyPath := keyPath + ".pub"

	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old private key: %w", err)
	}
	if err := os.Remove(publicKeyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old public key: %w", err)
	}

	// Generate new SSH key
	if err := s.ensureArcoCloudSSHKey(); err != nil {
		return fmt.Errorf("failed to generate new SSH key: %w", err)
	}

	// Get new public key
	publicKey, err := s.getArcoCloudPublicKey()
	if err != nil {
		return fmt.Errorf("failed to get new public key: %w", err)
	}

	// Call cloud service to replace SSH key
	req := connect.NewRequest(&arcov1.AddOrReplaceSSHKeyRequest{
		SshKey: publicKey,
	})

	resp, err := s.rpcClient.AddOrReplaceSSHKey(ctx, req)
	if err != nil {
		return s.handleRPCError("replace SSH key", err)
	}

	if !resp.Msg.Success {
		return fmt.Errorf("SSH key replacement failed")
	}

	if resp.Msg.KeyReplaced {
		s.log.Info("SSH key replaced successfully")
	} else {
		s.log.Info("SSH key added successfully")
	}
	return nil
}
