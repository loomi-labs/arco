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

// State variant structs (our custom data)
type StateIdle struct{}

type StateQueued struct {
	NextOperation Operation `json:"nextOperation"`
	QueueLength   int       `json:"queueLength"`
}

type StateBackingUp struct {
	BackupID  types.BackupId            `json:"backupId"`
	Progress  *borgtypes.BackupProgress `json:"progress,omitempty"`
	StartedAt time.Time                 `json:"startedAt"`
	cancelCtx cancelCtx                 // private context and cancel function
}

type StatePruning struct {
	BackupID  types.BackupId `json:"backupId"`
	StartedAt time.Time      `json:"startedAt"`
	cancelCtx cancelCtx      // private context and cancel function
}

type StateDeleting struct {
	ArchiveID int       `json:"archiveId"`
	StartedAt time.Time `json:"startedAt"`
	cancelCtx cancelCtx // private context and cancel function
}

type StateRefreshing struct {
	StartedAt time.Time `json:"startedAt"`
	cancelCtx cancelCtx // private context and cancel function
}

type StateMounted struct {
	MountType     MountType         `json:"mountType"`
	ArchiveID     *int              `json:"archiveId,omitempty"`
	MountPath     string            `json:"mountPath"`
	ArchiveMounts map[int]MountInfo `json:"archiveMounts"`
}

type StateError struct {
	ErrorType  ErrorType   `json:"errorType"`
	Message    string      `json:"message"`
	Action     ErrorAction `json:"action"`
	OccurredAt time.Time   `json:"occurredAt"`
}

// ============================================================================
// SUPPORTING TYPES
// ============================================================================

// MountType defines the type of mount
type MountType string

const (
	MountTypeRepository MountType = "repository"
	MountTypeArchive    MountType = "archive"
)

// MountInfo contains mount information for archives
type MountInfo struct {
	ArchiveID int    `json:"archiveId"`
	MountPath string `json:"mountPath"`
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

// RepositoryState ADT definition
type RepositoryState adtenum.Enum[RepositoryState]

// Variant wrappers using adtenum types
type IdleVariant adtenum.OneVariantValue[StateIdle]
type QueuedVariant adtenum.OneVariantValue[StateQueued]
type BackingUpVariant adtenum.OneVariantValue[StateBackingUp]
type PruningVariant adtenum.OneVariantValue[StatePruning]
type DeletingVariant adtenum.OneVariantValue[StateDeleting]
type RefreshingVariant adtenum.OneVariantValue[StateRefreshing]
type MountedVariant adtenum.OneVariantValue[StateMounted]
type ErrorVariant adtenum.OneVariantValue[StateError]

// Constructors
var NewStateIdle = adtenum.CreateOneVariantValueConstructor[IdleVariant]()
var NewStateQueued = adtenum.CreateOneVariantValueConstructor[QueuedVariant]()
var NewStateBackingUp = adtenum.CreateOneVariantValueConstructor[BackingUpVariant]()
var NewStatePruning = adtenum.CreateOneVariantValueConstructor[PruningVariant]()
var NewStateDeleting = adtenum.CreateOneVariantValueConstructor[DeletingVariant]()
var NewStateRefreshing = adtenum.CreateOneVariantValueConstructor[RefreshingVariant]()
var NewStateMounted = adtenum.CreateOneVariantValueConstructor[MountedVariant]()
var NewStateError = adtenum.CreateOneVariantValueConstructor[ErrorVariant]()

// Implement EnumType for each variant
func (v IdleVariant) EnumType() RepositoryState       { return v }
func (v QueuedVariant) EnumType() RepositoryState     { return v }
func (v BackingUpVariant) EnumType() RepositoryState  { return v }
func (v PruningVariant) EnumType() RepositoryState    { return v }
func (v DeletingVariant) EnumType() RepositoryState   { return v }
func (v RefreshingVariant) EnumType() RepositoryState { return v }
func (v MountedVariant) EnumType() RepositoryState    { return v }
func (v ErrorVariant) EnumType() RepositoryState      { return v }

// ============================================================================
// STATE UTILITY FUNCTIONS
// ============================================================================

// GetStateTypeName returns a string representation of the state type for debugging
func GetStateTypeName(state RepositoryState) string {
	switch state.(type) {
	case IdleVariant:
		return "Idle"
	case QueuedVariant:
		return "Queued"
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

// GetCancelContext extracts cancel context from active states
func GetCancelContext(state RepositoryState) (context.Context, context.CancelFunc, bool) {
	switch s := state.(type) {
	case BackingUpVariant:
		data := s()
		return data.cancelCtx.ctx, data.cancelCtx.cancel, true
	case PruningVariant:
		data := s()
		return data.cancelCtx.ctx, data.cancelCtx.cancel, true
	case DeletingVariant:
		data := s()
		return data.cancelCtx.ctx, data.cancelCtx.cancel, true
	case RefreshingVariant:
		data := s()
		return data.cancelCtx.ctx, data.cancelCtx.cancel, true
	default:
		return nil, nil, false
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
	return NewStateIdle(StateIdle{})
}

// CreateQueuedState creates a new queued state with operation info
func CreateQueuedState(nextOperation Operation, queueLength int) RepositoryState {
	return NewStateQueued(StateQueued{
		NextOperation: nextOperation,
		QueueLength:   queueLength,
	})
}

// CreateBackingUpState creates a new backing up state with context
func CreateBackingUpState(ctx context.Context, backupId types.BackupId) RepositoryState {
	return NewStateBackingUp(StateBackingUp{
		BackupID:  backupId,
		Progress:  nil,
		StartedAt: time.Now(),
		cancelCtx: CreateCancelContext(ctx),
	})
}

// CreatePruningState creates a new pruning state with context
func CreatePruningState(ctx context.Context, backupId types.BackupId) RepositoryState {
	return NewStatePruning(StatePruning{
		BackupID:  backupId,
		StartedAt: time.Now(),
		cancelCtx: CreateCancelContext(ctx),
	})
}

// CreateDeletingState creates a new deleting state with context
func CreateDeletingState(ctx context.Context, archiveId int) RepositoryState {
	return NewStateDeleting(StateDeleting{
		ArchiveID: archiveId,
		StartedAt: time.Now(),
		cancelCtx: CreateCancelContext(ctx),
	})
}

// CreateRefreshingState creates a new refreshing state with context
func CreateRefreshingState(ctx context.Context) RepositoryState {
	return NewStateRefreshing(StateRefreshing{
		StartedAt: time.Now(),
		cancelCtx: CreateCancelContext(ctx),
	})
}

// CreateMountedState creates a new mounted state
func CreateMountedState(mountType MountType, mountPath string, archiveId *int, archiveMounts map[int]MountInfo) RepositoryState {
	return NewStateMounted(StateMounted{
		MountType:     mountType,
		ArchiveID:     archiveId,
		MountPath:     mountPath,
		ArchiveMounts: archiveMounts,
	})
}

// CreateErrorState creates a new error state
func CreateErrorState(errorType ErrorType, message string, action ErrorAction) RepositoryState {
	return NewStateError(StateError{
		ErrorType:  errorType,
		Message:    message,
		Action:     action,
		OccurredAt: time.Now(),
	})
}
