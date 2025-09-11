package statemachine

import (
	"context"
	"time"

	"github.com/chris-tomich/adtenum"
	"github.com/loomi-labs/arco/backend/app/types"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
)

// ============================================================================
// STATE ADT (ALGEBRAIC DATA TYPE)
// ============================================================================

// cancelCtx holds context and cancel function for cancellable operations
type cancelCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// State variant structs
type Idle struct{}

type Queued struct {
	NextOperation Operation `json:"nextOperation"`
	QueueLength   int       `json:"queueLength"`
}

type BackingUp struct {
	BackupID  types.BackupId            `json:"backupId"`
	Progress  *borgtypes.BackupProgress `json:"progress,omitempty"`
	StartedAt time.Time                 `json:"startedAt"`
	cancelCtx cancelCtx                 // private context and cancel function
}

type Pruning struct {
	BackupID  types.BackupId `json:"backupId"`
	StartedAt time.Time      `json:"startedAt"`
	cancelCtx cancelCtx      // private context and cancel function
}

type Deleting struct {
	ArchiveID int       `json:"archiveId"`
	StartedAt time.Time `json:"startedAt"`
	cancelCtx cancelCtx // private context and cancel function
}

type Refreshing struct {
	StartedAt time.Time `json:"startedAt"`
	cancelCtx cancelCtx // private context and cancel function
}

type Mounted struct {
	Mounts []MountInfo `json:"mounts"`
}

type Error struct {
	ErrorType  ErrorType   `json:"errorType"`
	Message    string      `json:"message"`
	Action     ErrorAction `json:"action"`
	OccurredAt time.Time   `json:"occurredAt"`
}

// RepositoryState ADT definition
type RepositoryState adtenum.Enum[RepositoryState]

// Implement adtVariant marker interface for all state structs
func (Idle) isADTVariant() RepositoryState       { var zero RepositoryState; return zero }
func (Queued) isADTVariant() RepositoryState     { var zero RepositoryState; return zero }
func (BackingUp) isADTVariant() RepositoryState  { var zero RepositoryState; return zero }
func (Pruning) isADTVariant() RepositoryState    { var zero RepositoryState; return zero }
func (Deleting) isADTVariant() RepositoryState   { var zero RepositoryState; return zero }
func (Refreshing) isADTVariant() RepositoryState { var zero RepositoryState; return zero }
func (Mounted) isADTVariant() RepositoryState    { var zero RepositoryState; return zero }
func (Error) isADTVariant() RepositoryState      { var zero RepositoryState; return zero }

// ============================================================================
// SUPPORTING TYPES
// ============================================================================

// MountType defines the type of mount
type MountType string

const (
	MountTypeRepository MountType = "repository"
	MountTypeArchive    MountType = "archive"
)

// MountInfo contains mount information for archives and repositories
type MountInfo struct {
	MountType MountType `json:"mountType"`
	ArchiveID *int      `json:"archiveId,omitempty"`
	MountPath string    `json:"mountPath"`
}

// Error types for repository operations
type ErrorType string

const (
	ErrorTypeSSHKey     ErrorType = "sshKey"
	ErrorTypePassphrase ErrorType = "passphrase"
	ErrorTypeLocked     ErrorType = "locked"
)

// Actions that can be taken to resolve errors
type ErrorAction string

const (
	ErrorActionNone          ErrorAction = "none"
	ErrorActionRegenerateSSH ErrorAction = "regenerateSSH"
	ErrorActionBreakLock     ErrorAction = "breakLock"
)

// ============================================================================
// ADT ENUM DEFINITION
// ============================================================================

// ============================================================================
// STATE UTILITY FUNCTIONS
// ============================================================================

// GetStateTypeName returns a string representation of the state type for debugging
func GetStateTypeName(state RepositoryState) string {
	switch state.(type) {
	case IdleVariant:
		return "Idle"
	case BackingUpVariant:
		return "BackingUp"
	case PruningVariant:
		return "Pruning"
	case DeletingVariant:
		return "Deleting"
	case RefreshingVariant:
		return "Refreshing"
	case MountedVariant:
		return "Mounted"
	case ErrorVariant:
		return "Error"
	default:
		return "Unknown"
	}
}

// IsActiveState returns true if the state represents an active operation
func IsActiveState(state RepositoryState) bool {
	switch state.(type) {
	case BackingUpVariant, PruningVariant, DeletingVariant, RefreshingVariant:
		return true
	default:
		return false
	}
}

// IsIdleState returns true if the repository is idle
func IsIdleState(state RepositoryState) bool {
	_, ok := state.(IdleVariant)
	return ok
}

// IsQueuedState returns true if the repository has queued operations
func IsQueuedState(state RepositoryState) bool {
	_, ok := state.(QueuedVariant)
	return ok
}

// IsMountedState returns true if the repository is mounted
func IsMountedState(state RepositoryState) bool {
	_, ok := state.(MountedVariant)
	return ok
}

// IsErrorState returns true if the repository is in error state
func IsErrorState(state RepositoryState) bool {
	_, ok := state.(ErrorVariant)
	return ok
}

// GetCancel extracts cancel context from active states
func GetCancel(state RepositoryState) (context.CancelFunc, bool) {
	switch s := state.(type) {
	case BackingUpVariant:
		data := s()
		return data.cancelCtx.cancel, true
	case PruningVariant:
		data := s()
		return data.cancelCtx.cancel, true
	case DeletingVariant:
		data := s()
		return data.cancelCtx.cancel, true
	case RefreshingVariant:
		data := s()
		return data.cancelCtx.cancel, true
	default:
		return nil, false
	}
}

// CreateCancelContext creates a new cancel context for active operations
func CreateCancelContext(parent context.Context) cancelCtx {
	ctx, cancel := context.WithCancel(parent)
	return cancelCtx{
		ctx:    ctx,
		cancel: cancel,
	}
}

// ============================================================================
// STATE FACTORY METHODS
// ============================================================================

// CreateIdleState creates a new idle state
func CreateIdleState() RepositoryState {
	return NewRepositoryStateIdle(Idle{})
}

// CreateQueuedState creates a new queued state with operation info
func CreateQueuedState(nextOperation Operation, queueLength int) RepositoryState {
	return NewRepositoryStateQueued(Queued{
		NextOperation: nextOperation,
		QueueLength:   queueLength,
	})
}

// CreateBackingUpState creates a new backing up state with context
func CreateBackingUpState(ctx context.Context, backupId types.BackupId) RepositoryState {
	return NewRepositoryStateBackingUp(BackingUp{
		BackupID:  backupId,
		Progress:  nil,
		StartedAt: time.Now(),
		cancelCtx: CreateCancelContext(ctx),
	})
}

// CreatePruningState creates a new pruning state with context
func CreatePruningState(ctx context.Context, backupId types.BackupId) RepositoryState {
	return NewRepositoryStatePruning(Pruning{
		BackupID:  backupId,
		StartedAt: time.Now(),
		cancelCtx: CreateCancelContext(ctx),
	})
}

// CreateDeletingState creates a new deleting state with context
func CreateDeletingState(ctx context.Context, archiveId int) RepositoryState {
	return NewRepositoryStateDeleting(Deleting{
		ArchiveID: archiveId,
		StartedAt: time.Now(),
		cancelCtx: CreateCancelContext(ctx),
	})
}

// CreateRefreshingState creates a new refreshing state with context
func CreateRefreshingState(ctx context.Context) RepositoryState {
	return NewRepositoryStateRefreshing(Refreshing{
		StartedAt: time.Now(),
		cancelCtx: CreateCancelContext(ctx),
	})
}

// CreateMountedState creates a new mounted state with the given mounts
func CreateMountedState(mounts []MountInfo) RepositoryState {
	return NewRepositoryStateMounted(Mounted{
		Mounts: mounts,
	})
}

// CreateErrorState creates a new error state
func CreateErrorState(errorType ErrorType, message string, action ErrorAction) RepositoryState {
	return NewRepositoryStateError(Error{
		ErrorType:  errorType,
		Message:    message,
		Action:     action,
		OccurredAt: time.Now(),
	})
}
