package state

import (
	"context"
	"fmt"
	"time"

	"github.com/loomi-labs/arco/backend/app/types"
)

// StartBackup starts a backup operation on a repository
func (rsm *RepositoryStateManager) StartBackup(ctx context.Context, repoID int, options BackupOptions) error {
	return rsm.startOperation(ctx, repoID, OperationTypeBackup, RepoStatusBackingUp,
		fmt.Sprintf("Starting backup: %s", options.Reason), options)
}

// CompleteBackup marks a backup operation as successfully completed
func (rsm *RepositoryStateManager) CompleteBackup(ctx context.Context, repoID int, stats BackupStats) error {
	return rsm.completeOperation(ctx, repoID,
		fmt.Sprintf("Backup completed: %d files added, %d modified", stats.FilesAdded, stats.FilesModified),
		stats)
}

// FailBackup marks a backup operation as failed
func (rsm *RepositoryStateManager) FailBackup(ctx context.Context, repoID int, err error) {
	rsm.failOperation(ctx, repoID, err)
}

// StartPruning starts a pruning operation on a repository
func (rsm *RepositoryStateManager) StartPruning(ctx context.Context, repoID int, options PruneOptions) error {
	return rsm.startOperation(ctx, repoID, OperationTypePrune, RepoStatusPruning,
		fmt.Sprintf("Starting prune: %s", options.Reason), options)
}

// CompletePruning marks a pruning operation as successfully completed
func (rsm *RepositoryStateManager) CompletePruning(ctx context.Context, repoID int, stats PruneStats) error {
	return rsm.completeOperation(ctx, repoID,
		fmt.Sprintf("Pruning completed: freed %d bytes", stats.BytesFreed),
		stats)
}

// FailPruning marks a pruning operation as failed
func (rsm *RepositoryStateManager) FailPruning(ctx context.Context, repoID int, err error) {
	rsm.failOperation(ctx, repoID, err)
}

// StartDeleting starts a deletion operation on a repository
func (rsm *RepositoryStateManager) StartDeleting(ctx context.Context, repoID int, reason string) error {
	if len(reason) < 10 {
		return fmt.Errorf("deletion requires a detailed reason (minimum 10 characters)")
	}
	return rsm.startOperation(ctx, repoID, OperationTypeDelete, RepoStatusDeleting,
		fmt.Sprintf("Starting deletion: %s", reason), nil)
}

// CompleteDeleting marks a deletion operation as successfully completed
func (rsm *RepositoryStateManager) CompleteDeleting(ctx context.Context, repoID int) error {
	err := rsm.completeOperation(ctx, repoID, "Repository deletion completed", nil)
	if err == nil {
		// Remove repository from management after successful deletion
		rsm.removeRepository(repoID)
	}
	return err
}

// FailDeleting marks a deletion operation as failed
func (rsm *RepositoryStateManager) FailDeleting(ctx context.Context, repoID int, err error) {
	rsm.failOperation(ctx, repoID, err)
}

// MountRepository mounts a repository at the specified path
func (rsm *RepositoryStateManager) MountRepository(ctx context.Context, repoID int, mountPath string) error {
	repo := rsm.getRepository(repoID)

	// Transition to mounted state
	if err := repo.transitionTo(ctx, RepoStatusMounted, fmt.Sprintf("Mounting at %s", mountPath)); err != nil {
		return fmt.Errorf("cannot mount repository: %w", err)
	}

	// Store mount information
	repo.mu.Lock()
	repo.mountInfo = &MountInfo{
		Path:      mountPath,
		MountedAt: time.Now(),
		ProcessID: 0, // Will be set by the actual mount process
	}
	repo.mu.Unlock()

	rsm.eventEmitter.EmitEvent(ctx, types.EventRepoMounted(repoID))
	return nil
}

// UnmountRepository unmounts a previously mounted repository
func (rsm *RepositoryStateManager) UnmountRepository(ctx context.Context, repoID int) error {
	repo := rsm.getRepository(repoID)

	// Validate we're actually mounted
	if repo.GetStatus() != RepoStatusMounted {
		return fmt.Errorf("repository is not mounted, current state: %s", repo.GetStatus())
	}

	// Transition back to idle
	if err := repo.transitionTo(ctx, RepoStatusIdle, "Unmounting repository"); err != nil {
		return fmt.Errorf("failed to unmount: %w", err)
	}

	// Clear mount info
	repo.mu.Lock()
	repo.mountInfo = nil
	repo.mu.Unlock()

	rsm.eventEmitter.EmitEvent(ctx, types.EventRepoUnmounted(repoID))
	return nil
}

// GetMountInfo returns mount information for a repository if it's mounted
func (rsm *RepositoryStateManager) GetMountInfo(repoID int) (*MountInfo, error) {
	repo := rsm.getRepository(repoID)

	if repo.GetStatus() != RepoStatusMounted {
		return nil, fmt.Errorf("repository is not mounted")
	}

	return repo.GetMountInfo(), nil
}

// StartGeneralOperation starts a general operation (like refresh) on a repository
func (rsm *RepositoryStateManager) StartGeneralOperation(ctx context.Context, repoID int, operationName string) error {
	return rsm.startOperation(ctx, repoID, OperationTypeGeneral, RepoStatusPerformingOperation,
		fmt.Sprintf("Starting operation: %s", operationName), nil)
}

// CompleteGeneralOperation marks a general operation as successfully completed
func (rsm *RepositoryStateManager) CompleteGeneralOperation(ctx context.Context, repoID int, operationName string) error {
	return rsm.completeOperation(ctx, repoID, fmt.Sprintf("Operation completed: %s", operationName), nil)
}

// FailGeneralOperation marks a general operation as failed
func (rsm *RepositoryStateManager) FailGeneralOperation(ctx context.Context, repoID int, err error) {
	rsm.failOperation(ctx, repoID, err)
}

// CanTransitionTo checks if a transition is valid for a repository
func (rsm *RepositoryStateManager) CanTransitionTo(repoID int, targetState RepoStatus) (bool, string) {
	repo := rsm.getRepository(repoID)
	currentState := repo.GetStatus()

	// Use the state machine's validation logic
	err := repo.stateMachine.ValidateTransition(repoID, currentState, targetState, "validation check")
	if err != nil {
		return false, err.Error()
	}

	// Additional business rule checks can be added here
	validator := NewBusinessRuleValidator()
	ctx := TransitionContext{
		RepoID: repoID,
		Reason: "validation check",
	}

	if err := validator.ValidateTransition(ctx, currentState, targetState); err != nil {
		return false, err.Error()
	}

	return true, ""
}

// RecoverFromError attempts to recover a repository from error state
func (rsm *RepositoryStateManager) RecoverFromError(ctx context.Context, repoID int, resolution string) error {
	repo := rsm.getRepository(repoID)

	if repo.GetStatus() != RepoStatusError {
		return fmt.Errorf("repository is not in error state")
	}

	// Transition from error to idle
	if err := repo.transitionTo(ctx, RepoStatusIdle, fmt.Sprintf("Recovered from error: %s", resolution)); err != nil {
		return fmt.Errorf("failed to recover from error: %w", err)
	}

	// Clear error data
	repo.mu.Lock()
	repo.lastError = nil
	repo.operationData = nil
	repo.mu.Unlock()

	rsm.eventEmitter.EmitEvent(ctx, types.EventRepoRecovered(repoID))
	return nil
}
