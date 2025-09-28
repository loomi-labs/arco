package statemachine

import (
	"context"
	"time"

	"github.com/chris-tomich/adtenum"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/negrel/assert"
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
	Data      Backup
	cancelCtx cancelCtx
}

type Pruning struct {
	BackupID  types.BackupId `json:"backupId"`
	cancelCtx cancelCtx
}

type Deleting struct {
	ArchiveID int       `json:"archiveId"`
	StartedAt time.Time `json:"startedAt"`
	cancelCtx cancelCtx
}

type Refreshing struct {
	StartedAt time.Time `json:"startedAt"`
	cancelCtx cancelCtx
}

type Mounting struct {
	MountType MountType `json:"mountType"`
	ArchiveID *int      `json:"archiveId,omitempty"`
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
func (Mounting) isADTVariant() RepositoryState   { var zero RepositoryState; return zero }
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
	ErrorTypeGeneral    ErrorType = "general"
	ErrorTypeSSHKey     ErrorType = "sshKey"
	ErrorTypePassphrase ErrorType = "passphrase"
	ErrorTypeLocked     ErrorType = "locked"
)

// Actions that can be taken to resolve errors
type ErrorAction string

const (
	ErrorActionNone             ErrorAction = "none"
	ErrorActionRegenerateSSH    ErrorAction = "regenerateSSH"
	ErrorActionChangePassphrase ErrorAction = "changePassphrase"
	ErrorActionBreakLock        ErrorAction = "breakLock"
)

// ============================================================================
// ADT ENUM DEFINITION
// ============================================================================

// ============================================================================
// STATE UTILITY FUNCTIONS
// ============================================================================

// GetStateTypeName returns a string representation of the state type for debugging
func GetStateTypeName(state RepositoryState) string {
	switch GetRepositoryStateType(state) {
	case RepositoryStateTypeIdle:
		return "Idle"
	case RepositoryStateTypeBackingUp:
		return "BackingUp"
	case RepositoryStateTypePruning:
		return "Pruning"
	case RepositoryStateTypeDeleting:
		return "Deleting"
	case RepositoryStateTypeRefreshing:
		return "Refreshing"
	case RepositoryStateTypeMounting:
		return "Mounting"
	case RepositoryStateTypeMounted:
		return "Mounted"
	case RepositoryStateTypeQueued:
		return "Queued"
	case RepositoryStateTypeError:
		return "Error"
	default:
		assert.Fail("Unhandled RepositoryStateType in GetStateTypeName")
		return "Unknown"
	}
}

// IsActiveState returns true if the state represents an active operation
func IsActiveState(state RepositoryState) bool {
	switch GetRepositoryStateType(state) {
	case RepositoryStateTypeBackingUp, RepositoryStateTypePruning, RepositoryStateTypeDeleting, RepositoryStateTypeRefreshing, RepositoryStateTypeMounting:
		return true
	case RepositoryStateTypeIdle, RepositoryStateTypeQueued, RepositoryStateTypeMounted, RepositoryStateTypeError:
		return false
	default:
		assert.Fail("Unhandled RepositoryStateType in IsActiveState")
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

// IsMountingState returns true if the repository is mounting
func IsMountingState(state RepositoryState) bool {
	_, ok := state.(MountingVariant)
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

// GetCancelCtxOrDefault gets the context of a cancellable active state of return the provided context
func GetCancelCtxOrDefault(defaultContext context.Context, state RepositoryState) context.Context {
	switch GetRepositoryStateType(state) {
	case RepositoryStateTypeBackingUp:
		backingUpVariant := state.(BackingUpVariant)
		data := backingUpVariant()
		return data.cancelCtx.ctx
	case RepositoryStateTypePruning:
		pruningVariant := state.(PruningVariant)
		data := pruningVariant()
		return data.cancelCtx.ctx
	case RepositoryStateTypeDeleting:
		deletingVariant := state.(DeletingVariant)
		data := deletingVariant()
		return data.cancelCtx.ctx
	case RepositoryStateTypeRefreshing:
		refreshingVariant := state.(RefreshingVariant)
		data := refreshingVariant()
		return data.cancelCtx.ctx
	case RepositoryStateTypeIdle, RepositoryStateTypeQueued, RepositoryStateTypeMounted, RepositoryStateTypeMounting, RepositoryStateTypeError:
		return defaultContext
	default:
		assert.Fail("Unhandled RepositoryStateType in GetCancelCtxOrDefault")
		return defaultContext
	}
}

// GetCancel extracts cancel context from active states
func GetCancel(state RepositoryState) (context.CancelFunc, bool) {
	switch GetRepositoryStateType(state) {
	case RepositoryStateTypeBackingUp:
		backingUpVariant := state.(BackingUpVariant)
		data := backingUpVariant()
		return data.cancelCtx.cancel, true
	case RepositoryStateTypePruning:
		pruningVariant := state.(PruningVariant)
		data := pruningVariant()
		return data.cancelCtx.cancel, true
	case RepositoryStateTypeDeleting:
		deletingVariant := state.(DeletingVariant)
		data := deletingVariant()
		return data.cancelCtx.cancel, true
	case RepositoryStateTypeRefreshing:
		refreshingVariant := state.(RefreshingVariant)
		data := refreshingVariant()
		return data.cancelCtx.cancel, true
	case RepositoryStateTypeIdle, RepositoryStateTypeQueued, RepositoryStateTypeMounted, RepositoryStateTypeMounting, RepositoryStateTypeError:
		return nil, false
	default:
		assert.Fail("Unhandled RepositoryStateType in GetCancel")
		return nil, false
	}
}

// createCancelContext creates a new cancel context for active operations
func createCancelContext(parent context.Context) cancelCtx {
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
func CreateBackingUpState(ctx context.Context, data Backup) RepositoryState {
	return NewRepositoryStateBackingUp(BackingUp{
		Data:      data,
		cancelCtx: createCancelContext(ctx),
	})
}

// CreatePruningState creates a new pruning state with context and backup ID
func CreatePruningState(ctx context.Context, backupID types.BackupId) RepositoryState {
	return NewRepositoryStatePruning(Pruning{
		BackupID:  backupID,
		cancelCtx: createCancelContext(ctx),
	})
}

// CreateDeletingState creates a new deleting state with context
func CreateDeletingState(ctx context.Context, archiveId int) RepositoryState {
	return NewRepositoryStateDeleting(Deleting{
		ArchiveID: archiveId,
		StartedAt: time.Now(),
		cancelCtx: createCancelContext(ctx),
	})
}

// CreateRefreshingState creates a new refreshing state with context
func CreateRefreshingState(ctx context.Context) RepositoryState {
	return NewRepositoryStateRefreshing(Refreshing{
		StartedAt: time.Now(),
		cancelCtx: createCancelContext(ctx),
	})
}

// CreateMountingState creates a new mounting state
func CreateMountingState(archiveID *int) RepositoryState {
	mountType := MountTypeRepository
	if archiveID != nil {
		mountType = MountTypeArchive
	}
	return NewRepositoryStateMounting(Mounting{
		MountType: mountType,
		ArchiveID: archiveID,
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
