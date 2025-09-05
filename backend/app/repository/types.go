package repository

import (
	"context"
	"time"

	"github.com/chris-tomich/adtenum"
	"github.com/loomi-labs/arco/backend/app/types"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
)

// ============================================================================
// CORE DATA STRUCTURES
// ============================================================================

// Repository represents the consolidated repository data structure
type Repository struct {
	// Core fields
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`

	// Cloud info
	IsCloud bool   `json:"isCloud"`
	CloudID string `json:"cloudId,omitempty"`

	// Current state (ADT enum)
	State RepositoryState `json:"state"`

	// Metadata
	ArchiveCount    int        `json:"archiveCount"`
	LastBackupTime  *time.Time `json:"lastBackupTime,omitempty"`
	LastBackupError string     `json:"lastBackupError,omitempty"`
	StorageUsed     int64      `json:"storageUsed"`
}

// RepositoryWithQueue extends Repository with queue information for frontend
type RepositoryWithQueue struct {
	Repository       `json:",inline"`
	QueuedOperations []QueuedOperation `json:"queuedOperations"`
	ActiveOperation  *QueuedOperation  `json:"activeOperation,omitempty"`
}

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

// ADT enum definition
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
var NewStateIdle func(StateIdle) IdleVariant = adtenum.CreateOneVariantValueConstructor[IdleVariant]()
var NewStateQueued func(StateQueued) QueuedVariant = adtenum.CreateOneVariantValueConstructor[QueuedVariant]()
var NewStateBackingUp func(StateBackingUp) BackingUpVariant = adtenum.CreateOneVariantValueConstructor[BackingUpVariant]()
var NewStatePruning func(StatePruning) PruningVariant = adtenum.CreateOneVariantValueConstructor[PruningVariant]()
var NewStateDeleting func(StateDeleting) DeletingVariant = adtenum.CreateOneVariantValueConstructor[DeletingVariant]()
var NewStateRefreshing func(StateRefreshing) RefreshingVariant = adtenum.CreateOneVariantValueConstructor[RefreshingVariant]()
var NewStateMounted func(StateMounted) MountedVariant = adtenum.CreateOneVariantValueConstructor[MountedVariant]()
var NewStateError func(StateError) ErrorVariant = adtenum.CreateOneVariantValueConstructor[ErrorVariant]()

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
// OPERATION ADT
// ============================================================================

// Operation variant structs (type-safe parameters)
type OpBackup struct {
	BackupID types.BackupId `json:"backupId"`
}

type OpPrune struct {
	BackupID types.BackupId `json:"backupId"`
}

type OpDelete struct {
	// Repository delete - no additional params
}

type OpArchiveRefresh struct {
	// Archive refresh - no additional params
}

type OpArchiveDelete struct {
	ArchiveID int `json:"archiveId"`
}

type OpArchiveRename struct {
	ArchiveID int    `json:"archiveId"`
	Prefix    string `json:"prefix"`
	Name      string `json:"name"`
}

// Operation ADT definition
type Operation adtenum.Enum[Operation]

// Operation variant wrappers
type BackupVariant adtenum.OneVariantValue[OpBackup]
type PruneVariant adtenum.OneVariantValue[OpPrune]
type DeleteVariant adtenum.OneVariantValue[OpDelete]
type ArchiveRefreshVariant adtenum.OneVariantValue[OpArchiveRefresh]
type ArchiveDeleteVariant adtenum.OneVariantValue[OpArchiveDelete]
type ArchiveRenameVariant adtenum.OneVariantValue[OpArchiveRename]

// Operation constructors
var NewOpBackup func(OpBackup) BackupVariant = adtenum.CreateOneVariantValueConstructor[BackupVariant]()
var NewOpPrune func(OpPrune) PruneVariant = adtenum.CreateOneVariantValueConstructor[PruneVariant]()
var NewOpDelete func(OpDelete) DeleteVariant = adtenum.CreateOneVariantValueConstructor[DeleteVariant]()
var NewOpArchiveRefresh func(OpArchiveRefresh) ArchiveRefreshVariant = adtenum.CreateOneVariantValueConstructor[ArchiveRefreshVariant]()
var NewOpArchiveDelete func(OpArchiveDelete) ArchiveDeleteVariant = adtenum.CreateOneVariantValueConstructor[ArchiveDeleteVariant]()
var NewOpArchiveRename func(OpArchiveRename) ArchiveRenameVariant = adtenum.CreateOneVariantValueConstructor[ArchiveRenameVariant]()

// Implement EnumType for operation variants
func (v BackupVariant) EnumType() Operation         { return v }
func (v PruneVariant) EnumType() Operation          { return v }
func (v DeleteVariant) EnumType() Operation         { return v }
func (v ArchiveRefreshVariant) EnumType() Operation { return v }
func (v ArchiveDeleteVariant) EnumType() Operation  { return v }
func (v ArchiveRenameVariant) EnumType() Operation  { return v }

// ============================================================================
// OPERATION STATUS ADT
// ============================================================================

// Status variant structs (type-safe status information)
type StatusQueued struct {
	Position int `json:"position"` // Position in queue
}

type StatusRunning struct {
	Progress  *Progress `json:"progress,omitempty"`
	StartedAt time.Time `json:"startedAt"`
}

type StatusCompleted struct {
	CompletedAt time.Time `json:"completedAt"`
}

type StatusFailed struct {
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failedAt"`
	CanRetry bool      `json:"canRetry"`
}

type StatusExpired struct {
	ExpiredAt time.Time `json:"expiredAt"`
}

// Progress represents generic progress information
type Progress struct {
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Message string `json:"message,omitempty"`
}

// OperationStatus ADT definition
type OperationStatus adtenum.Enum[OperationStatus]

// Status variant wrappers
type QueuedStatusVariant adtenum.OneVariantValue[StatusQueued]
type RunningStatusVariant adtenum.OneVariantValue[StatusRunning]
type CompletedStatusVariant adtenum.OneVariantValue[StatusCompleted]
type FailedStatusVariant adtenum.OneVariantValue[StatusFailed]
type ExpiredStatusVariant adtenum.OneVariantValue[StatusExpired]

// Status constructors
var NewStatusQueued func(StatusQueued) QueuedStatusVariant = adtenum.CreateOneVariantValueConstructor[QueuedStatusVariant]()
var NewStatusRunning func(StatusRunning) RunningStatusVariant = adtenum.CreateOneVariantValueConstructor[RunningStatusVariant]()
var NewStatusCompleted func(StatusCompleted) CompletedStatusVariant = adtenum.CreateOneVariantValueConstructor[CompletedStatusVariant]()
var NewStatusFailed func(StatusFailed) FailedStatusVariant = adtenum.CreateOneVariantValueConstructor[FailedStatusVariant]()
var NewStatusExpired func(StatusExpired) ExpiredStatusVariant = adtenum.CreateOneVariantValueConstructor[ExpiredStatusVariant]()

// Implement EnumType for status variants
func (v QueuedStatusVariant) EnumType() OperationStatus    { return v }
func (v RunningStatusVariant) EnumType() OperationStatus   { return v }
func (v CompletedStatusVariant) EnumType() OperationStatus { return v }
func (v FailedStatusVariant) EnumType() OperationStatus    { return v }
func (v ExpiredStatusVariant) EnumType() OperationStatus   { return v }

// ============================================================================
// QUEUED OPERATION
// ============================================================================

// QueuedOperation represents a queued repository operation
type QueuedOperation struct {
	ID         string          `json:"id"`        // Unique operation ID (UUID) - enables idempotency and deduplication
	Operation  Operation       `json:"operation"` // ADT containing type and parameters
	Status     OperationStatus `json:"status"`    // ADT containing status, progress, error
	RepoID     int             `json:"repoId"`
	CreatedAt  time.Time       `json:"createdAt"`
	ValidUntil time.Time       `json:"validUntil"` // Auto-expire if not started
}

// ============================================================================
// QUEUE MANAGEMENT
// ============================================================================

// OperationWeight defines the resource intensity of operations
type OperationWeight int

const (
	WeightLight OperationWeight = iota // Quick operations (refresh, rename, single archive delete)
	WeightHeavy                        // Resource-intensive operations (backup, prune, repo delete)
)

// GetOperationWeight determines operation weight for concurrency control
func GetOperationWeight(op Operation) OperationWeight {
	switch op.(type) {
	case BackupVariant, PruneVariant, DeleteVariant:
		return WeightHeavy
	case ArchiveRefreshVariant, ArchiveDeleteVariant, ArchiveRenameVariant:
		return WeightLight
	default:
		return WeightLight
	}
}

// ============================================================================
// SUPPORTING TYPES
// ============================================================================

// ExaminePruningResult represents the result of examining pruning operations
type ExaminePruningResult struct {
	BackupID               types.BackupId `json:"backupId"`
	RepositoryName         string         `json:"repositoryName"`
	CntArchivesToBeDeleted int            `json:"cntArchivesToBeDeleted"`
	Error                  error          `json:"error,omitempty"`
}

// TestRepoConnectionResult represents the result of testing repository connection
type TestRepoConnectionResult struct {
	Success         bool `json:"success"`
	NeedsPassword   bool `json:"needsPassword"`
	IsPasswordValid bool `json:"isPasswordValid"`
	IsBorgRepo      bool `json:"isBorgRepo"`
}

// PaginatedArchivesRequest represents a request for paginated archives
type PaginatedArchivesRequest struct {
	// Required
	RepositoryId int `json:"repositoryId"`
	Page         int `json:"page"`
	PageSize     int `json:"pageSize"`
	// Optional filters can be added here
}

// PaginatedArchivesResponse represents the response for paginated archives
type PaginatedArchivesResponse struct {
	Archives []*ent.Archive `json:"archives"`
	Total    int            `json:"total"`
}

// PruningDates represents pruning date information for archives
type PruningDates struct {
	Dates []PruningDate `json:"dates"`
}

// PruningDate represents pruning information for a single archive
type PruningDate struct {
	ArchiveId int       `json:"archiveId"`
	Date      time.Time `json:"date"`
}
