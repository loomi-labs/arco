package repository

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"fmt"
	"github.com/ydb-platform/ydb-go-sdk/v3/log"
	"os"
	"path/filepath"
	"strings"

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

// Helper function to convert proto location to enum
func getLocationEnum(location arcov1.RepositoryLocation) cloudrepository.Location {
	switch location {
	case arcov1.RepositoryLocation_REPOSITORY_LOCATION_EU:
		return cloudrepository.LocationEU
	case arcov1.RepositoryLocation_REPOSITORY_LOCATION_US:
		return cloudrepository.LocationUS
	default:
		log.Error(errors.New("unknown location"))
		return cloudrepository.LocationEU // Default to EU
	}
}

type CloudRepositoryStatus string

const (
	CloudRepositoryStatusSuccess         CloudRepositoryStatus = "success"
	CloudRepositoryStatusRateLimitError  CloudRepositoryStatus = "rateLimitError"
	CloudRepositoryStatusConnectionError CloudRepositoryStatus = "connectionError"
	CloudRepositoryStatusNotFound        CloudRepositoryStatus = "notFound"
	CloudRepositoryStatusError           CloudRepositoryStatus = "error"
)

// CloudRepositoryService contains the business logic for ArcoCloud repositories
type CloudRepositoryService struct {
	log       *zap.SugaredLogger
	db        *ent.Client
	state     *state.State
	config    *types.Config
	rpcClient arcov1connect.RepositoryServiceClient
}

// CloudRepositoryServiceInternal provides backend-only methods that should not be exposed to frontend
type CloudRepositoryServiceInternal struct {
	*CloudRepositoryService
	arcov1connect.UnimplementedRepositoryServiceHandler
}

// NewCloudRepositoryService creates a new cloud repository service
func NewCloudRepositoryService(log *zap.SugaredLogger, state *state.State, config *types.Config) *CloudRepositoryServiceInternal {
	return &CloudRepositoryServiceInternal{
		CloudRepositoryService: &CloudRepositoryService{
			log:    log,
			state:  state,
			config: config,
		},
	}
}

// Init initializes the service with database and RPC client
func (si *CloudRepositoryServiceInternal) Init(db *ent.Client, rpcClient arcov1connect.RepositoryServiceClient) {
	si.db = db
	si.rpcClient = rpcClient
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *CloudRepositoryService) mustHaveDB() {
	if s.db == nil {
		panic("CloudRepositoryService: database client is nil")
	}
}

// SSH Key Management Functions

// ensureArcoCloudSSHKey ensures that the ArcoCloud SSH key pair exists
func (s *CloudRepositoryService) ensureArcoCloudSSHKey() error {
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
func (s *CloudRepositoryService) getArcoCloudPublicKey() (string, error) {
	publicKeyPath := s.getArcoCloudSSHKeyPath() + ".pub"
	content, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

// getArcoCloudSSHKeyPath returns the private key path for Borg operations
func (s *CloudRepositoryService) getArcoCloudSSHKeyPath() string {
	return filepath.Join(s.config.SSHDir, "arco_cloud_ed25519")
}

// Repository Type Detection

// IsCloudRepository checks if a repository is an ArcoCloud repository
func (s *CloudRepositoryService) IsCloudRepository(repo *ent.Repository) bool {
	// Check if repository has a cloud repository relationship
	if repo.Edges.CloudRepository != nil {
		return repo.Edges.CloudRepository.CloudID != ""
	}
	return false
}

// Error Handling

// handleRPCError maps Connect RPC errors to CloudRepositoryStatus
func (s *CloudRepositoryService) handleRPCError(operation string, err error) error {
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

// TODO: do we need this???
// syncCloudRepository creates or updates a local repository entity with cloud metadata
func (s *CloudRepositoryService) syncCloudRepository(ctx context.Context, cloudRepo *arcov1.Repository) (*ent.Repository, error) {
	s.mustHaveDB()

	// Check if local repository already exists by ArcoCloud ID
	if cloudRepo.Id != "" {
		if localRepo, err := s.db.Repository.Query().
			Where(repository.HasCloudRepositoryWith(
				cloudrepository.CloudIDEQ(cloudRepo.Id),
			)).
			First(ctx); err == nil {
			// Update existing repository
			updateQuery := s.db.Repository.UpdateOne(localRepo).
				SetName(cloudRepo.Name).
				SetURL(cloudRepo.RepoUrl)
			return updateQuery.Save(ctx)
		}
	}

	// Check if repository exists by location (repo URL)
	if localRepo, err := s.db.Repository.Query().
		Where(repository.URLEQ(cloudRepo.RepoUrl)).
		First(ctx); err == nil {
		// Create or update cloud repository association
		_, err := s.db.CloudRepository.Create().
			SetCloudID(cloudRepo.Id).
			SetStorageUsedBytes(cloudRepo.StorageUsedBytes).
			SetLocation(getLocationEnum(cloudRepo.Location)).
			SetRepository(localRepo).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create cloud repository: %w", err)
		}

		// Update repository name if needed
		updateQuery := s.db.Repository.UpdateOne(localRepo).
			SetName(cloudRepo.Name)
		return updateQuery.Save(ctx)
	}

	// Create new local repository with cloud association
	localRepo, err := s.db.Repository.Create().
		SetName(cloudRepo.Name).
		SetURL(cloudRepo.RepoUrl).
		SetPassword(""). // Will be set later
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Create cloud repository association
	_, err = s.db.CloudRepository.Create().
		SetCloudID(cloudRepo.Id).
		SetStorageUsedBytes(cloudRepo.StorageUsedBytes).
		SetLocation(getLocationEnum(cloudRepo.Location)).
		SetRepository(localRepo).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud repository: %w", err)
	}

	return localRepo, nil
}

// Frontend-exposed business logic methods

// AddCloudRepository creates a new ArcoCloud repository
func (s *CloudRepositoryService) AddCloudRepository(ctx context.Context, name string, location arcov1.RepositoryLocation) (*arcov1.Repository, error) {
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
func (s *CloudRepositoryService) DeleteCloudRepository(ctx context.Context, repositoryID string) error {
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
func (s *CloudRepositoryService) ListCloudRepositories(ctx context.Context) ([]*ent.Repository, error) {
	// Call cloud service
	req := connect.NewRequest(&arcov1.ListRepositoriesRequest{})

	resp, err := s.rpcClient.ListRepositories(ctx, req)
	if err != nil {
		return nil, s.handleRPCError("list repositories", err)
	}

	// Sync all cloud repositories with local entities
	var localRepos []*ent.Repository
	for _, cloudRepo := range resp.Msg.Repositories {
		localRepo, err := s.syncCloudRepository(ctx, cloudRepo)
		if err != nil {
			s.log.Warnf("Failed to sync repository %s: %v", cloudRepo.Id, err)
			continue
		}
		localRepos = append(localRepos, localRepo)
	}

	s.log.Debugf("Listed %d ArcoCloud repositories", len(localRepos))
	return localRepos, nil
}

// GetCloudRepository retrieves detailed information about a specific ArcoCloud repository
func (s *CloudRepositoryService) GetCloudRepository(ctx context.Context, repositoryID string) (*ent.Repository, error) {
	// Call cloud service
	req := connect.NewRequest(&arcov1.GetRepositoryRequest{
		RepositoryId: repositoryID,
	})

	resp, err := s.rpcClient.GetRepository(ctx, req)
	if err != nil {
		return nil, s.handleRPCError("get repository", err)
	}

	// Sync with local entity
	localRepo, err := s.syncCloudRepository(ctx, resp.Msg.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to sync local repository: %w", err)
	}

	return localRepo, nil
}

// ReplaceCloudRepositorySSHKey regenerates the local SSH key and updates the cloud repository
func (s *CloudRepositoryService) ReplaceCloudRepositorySSHKey(ctx context.Context, repositoryID string) error {
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
	req := connect.NewRequest(&arcov1.ReplaceSSHKeyRequest{
		RepositoryId: repositoryID,
		SshKey:       publicKey,
	})

	resp, err := s.rpcClient.ReplaceSSHKey(ctx, req)
	if err != nil {
		return s.handleRPCError("replace SSH key", err)
	}

	if !resp.Msg.Success {
		return fmt.Errorf("SSH key replacement failed")
	}

	s.log.Infof("Replaced SSH key for repository %s, new fingerprint: %s", repositoryID, resp.Msg.SshKeyFingerprint)
	return nil
}

// GetArcoCloudRepositorySSHKeyPath returns the private key path for Borg operations
// This is exposed to allow the Borg client to use the SSH key for repository operations
func (s *CloudRepositoryService) GetArcoCloudRepositorySSHKeyPath() string {
	return s.getArcoCloudSSHKeyPath()
}
