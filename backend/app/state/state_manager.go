package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/loomi-labs/arco/backend/app/types"
	"go.uber.org/zap"
)

// OperationType represents the type of operation being performed
type OperationType string

const (
	OperationTypeBackup  OperationType = "backup"
	OperationTypePrune   OperationType = "prune"
	OperationTypeDelete  OperationType = "delete"
	OperationTypeMount   OperationType = "mount"
	OperationTypeGeneral OperationType = "general"
)

// OperationData stores metadata about the current operation
type OperationData struct {
	Type      OperationType `json:"type"`
	StartTime time.Time     `json:"start_time"`
	Options   interface{}   `json:"options,omitempty"`
}

type RepoStatus string

const (
	RepoStatusIdle                RepoStatus = "idle"
	RepoStatusBackingUp           RepoStatus = "backingUp"
	RepoStatusPruning             RepoStatus = "pruning"
	RepoStatusDeleting            RepoStatus = "deleting"
	RepoStatusMounted             RepoStatus = "mounted"
	RepoStatusPerformingOperation RepoStatus = "performingOperation"
	RepoStatusError               RepoStatus = "error"
)

var AvailableRepoStatuses = []RepoStatus{
	RepoStatusIdle,
	RepoStatusBackingUp,
	RepoStatusPruning,
	RepoStatusDeleting,
	RepoStatusMounted,
	RepoStatusPerformingOperation,
	RepoStatusError,
}

func (rs RepoStatus) String() string {
	return string(rs)
}

type RepoErrorType string

const (
	RepoErrorTypeNone        RepoErrorType = "none"
	RepoErrorTypeSSHKey      RepoErrorType = "sshKey"
	RepoErrorTypePassphrase  RepoErrorType = "passphrase"
	RepoErrorTypeLockTimeout RepoErrorType = "lockTimeout"
)

func (ret RepoErrorType) String() string {
	return string(ret)
}

type RepoErrorAction string

const (
	RepoErrorActionNone             RepoErrorAction = "none"
	RepoErrorActionRegenerateSSH    RepoErrorAction = "regenerateSSH"
	RepoErrorActionUnlockRepository RepoErrorAction = "unlockRepository"
)

func (rea RepoErrorAction) String() string {
	return string(rea)
}

// MountInfo stores information about mounted repositories
type MountInfo struct {
	Path      string    `json:"path"`
	MountedAt time.Time `json:"mounted_at"`
	ProcessID int       `json:"process_id"`
}

// BackupStats represents backup operation statistics
type BackupStats struct {
	ArchiveSize   int64  `json:"archive_size"`
	FilesAdded    int    `json:"files_added"`
	FilesModified int    `json:"files_modified"`
	Duration      string `json:"duration"`
}

// PruneStats represents prune operation statistics
type PruneStats struct {
	BytesFreed    int64  `json:"bytes_freed"`
	ArchivesKept  int    `json:"archives_kept"`
	ArchivesFreed int    `json:"archives_freed"`
	Duration      string `json:"duration"`
}

// PruneOptions contains options for pruning operations
type PruneOptions struct {
	Reason      string `json:"reason"`
	KeepDaily   int    `json:"keep_daily,omitempty"`
	KeepWeekly  int    `json:"keep_weekly,omitempty"`
	KeepMonthly int    `json:"keep_monthly,omitempty"`
}

// BackupOptions contains options for backup operations
type BackupOptions struct {
	Reason      string   `json:"reason"`
	Paths       []string `json:"paths,omitempty"`
	ExcludeFrom string   `json:"exclude_from,omitempty"`
}

// Repository represents a managed repository with its state machine
type Repository struct {
	id              int
	stateMachine    *RepoStateMachine
	operationData   *OperationData
	mountInfo       *MountInfo
	lastError       error
	lastBackupStats *BackupStats
	lastPruneStats  *PruneStats
	errorType       RepoErrorType
	errorMessage    string
	errorAction     RepoErrorAction
	warningMessage  string
	mu              sync.RWMutex
}

// NewRepository creates a new managed repository
func NewRepository(id int, eventEmitter types.EventEmitter) *Repository {
	stateMachine := NewRepoStateMachine()
	executor := NewTransitionExecutor(stateMachine, eventEmitter)

	repo := &Repository{
		id:           id,
		stateMachine: stateMachine,
		errorType:    RepoErrorTypeNone,
		errorAction:  RepoErrorActionNone,
	}

	// Store the executor in the repository for transition execution
	// We'll access it through the state machine
	stateMachine.executor = executor

	return repo
}

// GetStatus returns the current repository status
func (r *Repository) GetStatus() RepoStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stateMachine.GetCurrentState(r.id)
}

// GetOperationData returns current operation metadata
func (r *Repository) GetOperationData() *OperationData {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.operationData
}

// GetMountInfo returns mount information if mounted
func (r *Repository) GetMountInfo() *MountInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mountInfo
}

// GetLastError returns the last error encountered
func (r *Repository) GetLastError() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastError
}

// transitionTo executes a state transition with proper locking
func (r *Repository) transitionTo(ctx context.Context, targetState RepoStatus, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	currentState := r.stateMachine.GetCurrentState(r.id)

	transitionCtx := TransitionContext{
		RepoID:  r.id,
		Context: ctx,
	}

	_, err := r.stateMachine.executor.ExecuteTransition(transitionCtx, currentState, targetState)
	return err
}

// RepositoryStateManager manages all repository states centrally
type RepositoryStateManager struct {
	log          *zap.SugaredLogger
	repositories map[int]*Repository
	eventEmitter types.EventEmitter
	mu           sync.RWMutex
}

// NewRepositoryStateManager creates a new state manager
func NewRepositoryStateManager(log *zap.SugaredLogger, eventEmitter types.EventEmitter) *RepositoryStateManager {
	return &RepositoryStateManager{
		log:          log,
		repositories: make(map[int]*Repository),
		eventEmitter: eventEmitter,
	}
}

// getRepository safely gets a repository, creating it if it doesn't exist
func (rsm *RepositoryStateManager) getRepository(repoID int) *Repository {
	rsm.mu.Lock()
	defer rsm.mu.Unlock()

	if repo, exists := rsm.repositories[repoID]; exists {
		return repo
	}

	// Create new repository if it doesn't exist
	repo := NewRepository(repoID, rsm.eventEmitter)
	rsm.repositories[repoID] = repo
	return repo
}

// startOperation starts a generic operation on a repository
func (rsm *RepositoryStateManager) startOperation(ctx context.Context, repoID int, operation OperationType, targetState RepoStatus, reason string, options interface{}) error {
	repo := rsm.getRepository(repoID)

	if err := repo.transitionTo(ctx, targetState, reason); err != nil {
		return fmt.Errorf("cannot start %s operation: %w", operation, err)
	}

	repo.mu.Lock()
	repo.operationData = &OperationData{
		Type:      operation,
		StartTime: time.Now(),
		Options:   options,
	}
	repo.mu.Unlock()

	rsm.eventEmitter.EmitEvent(ctx, types.EventRepoOperationStarted(repoID, string(operation)))
	return nil
}

// completeOperation marks an operation as successfully completed
func (rsm *RepositoryStateManager) completeOperation(ctx context.Context, repoID int, reason string, results interface{}) error {
	repo := rsm.getRepository(repoID)

	currentStatus := repo.GetStatus()
	if currentStatus == RepoStatusIdle {
		return fmt.Errorf("repository %d is already idle", repoID)
	}

	if err := repo.transitionTo(ctx, RepoStatusIdle, reason); err != nil {
		return fmt.Errorf("failed to complete operation: %w", err)
	}

	repo.mu.Lock()
	// Store operation-specific results
	if repo.operationData != nil {
		switch repo.operationData.Type {
		case OperationTypeBackup:
			if stats, ok := results.(BackupStats); ok {
				repo.lastBackupStats = &stats
			}
		case OperationTypePrune:
			if stats, ok := results.(PruneStats); ok {
				repo.lastPruneStats = &stats
			}
		}
	}
	repo.operationData = nil
	repo.lastError = nil
	repo.mu.Unlock()

	rsm.eventEmitter.EmitEvent(ctx, types.EventRepoOperationCompleted(repoID))
	return nil
}

// failOperation marks an operation as failed and transitions to error state
func (rsm *RepositoryStateManager) failOperation(ctx context.Context, repoID int, err error) {
	repo := rsm.getRepository(repoID)

	reason := fmt.Sprintf("Operation failed: %v", err)
	if transErr := repo.transitionTo(ctx, RepoStatusError, reason); transErr != nil {
		// Force transition if regular transition fails
		repo.mu.Lock()
		repo.stateMachine.executor.ForceTransition(TransitionContext{
			RepoID:  repoID,
			Context: ctx,
		}, repo.GetStatus(), RepoStatusError)
		repo.mu.Unlock()
	}

	repo.mu.Lock()
	repo.lastError = err
	repo.operationData = nil
	repo.mu.Unlock()

	rsm.eventEmitter.EmitEvent(ctx, types.EventRepoOperationFailed(repoID))
}

// forceReset forces a repository to idle state for emergency situations
func (rsm *RepositoryStateManager) forceReset(ctx context.Context, repoID int, reason string) {
	repo := rsm.getRepository(repoID)

	currentStatus := repo.GetStatus()

	repo.mu.Lock()
	repo.stateMachine.executor.ForceTransition(TransitionContext{
		RepoID:  repoID,
		Context: ctx,
	}, currentStatus, RepoStatusIdle)

	// Clear all operation data
	repo.operationData = nil
	repo.mountInfo = nil
	repo.lastError = nil
	repo.mu.Unlock()

	rsm.eventEmitter.EmitEvent(ctx, types.EventRepoForceReset(repoID))
}

// TODO: do we need the following func's???

// removeRepository removes a repository from management
func (rsm *RepositoryStateManager) removeRepository(repoID int) {
	rsm.mu.Lock()
	defer rsm.mu.Unlock()
	delete(rsm.repositories, repoID)
}

// listRepositories returns all managed repositories
func (rsm *RepositoryStateManager) listRepositories() map[int]*Repository {
	rsm.mu.RLock()
	defer rsm.mu.RUnlock()

	result := make(map[int]*Repository)
	for id, repo := range rsm.repositories {
		result[id] = repo
	}
	return result
}

// getRepositoryStatus returns the current status of a repository
func (rsm *RepositoryStateManager) getRepositoryStatus(repoID int) (RepoStatus, error) {
	rsm.mu.RLock()
	defer rsm.mu.RUnlock()

	if repo, exists := rsm.repositories[repoID]; exists {
		return repo.GetStatus(), nil
	}

	return RepoStatusIdle, nil // Default to idle for unknown repositories
}

// IsBackupRunning checks if a backup operation is currently running for a repository
func (rsm *RepositoryStateManager) IsBackupRunning(repoID int) bool {
	rsm.mu.RLock()
	defer rsm.mu.RUnlock()

	if repo, exists := rsm.repositories[repoID]; exists {
		return repo.GetStatus() == RepoStatusBackingUp
	}

	return false
}

// HasError checks if a repository has errors that need fixing
func (rsm *RepositoryStateManager) HasError(repoID int) bool {
	rsm.mu.RLock()
	defer rsm.mu.RUnlock()

	if repo, exists := rsm.repositories[repoID]; exists {
		return repo.GetStatus() == RepoStatusError
	}

	return false
}

// SetWarning sets repository warning state
func (rsm *RepositoryStateManager) SetWarning(ctx context.Context, repoID int, message string) {
	repo := rsm.getRepository(repoID)
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.warningMessage = message
	// Note: warnings don't change the repository state, they're just informational
}

// ClearError clears repository error state
func (rsm *RepositoryStateManager) ClearError(ctx context.Context, repoID int) {
	repo := rsm.getRepository(repoID)
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.errorType = RepoErrorTypeNone
	repo.errorMessage = ""
	repo.errorAction = RepoErrorActionNone
}

// ClearWarning clears repository warning state
func (rsm *RepositoryStateManager) ClearWarning(ctx context.Context, repoID int) {
	repo := rsm.getRepository(repoID)
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.warningMessage = ""
}

// GetErrorInfo returns the current error information for a repository
func (rsm *RepositoryStateManager) GetErrorInfo(repoID int) (RepoErrorType, string, RepoErrorAction) {
	repo := rsm.getRepository(repoID)
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	return repo.errorType, repo.errorMessage, repo.errorAction
}

// GetWarningMessage returns the current warning message for a repository
func (rsm *RepositoryStateManager) GetWarningMessage(repoID int) string {
	repo := rsm.getRepository(repoID)
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	return repo.warningMessage
}
