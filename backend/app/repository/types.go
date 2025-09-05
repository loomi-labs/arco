package repository

import (
	"time"

	"github.com/chris-tomich/adtenum"
	"github.com/loomi-labs/arco/backend/app/statemachine"
	"github.com/loomi-labs/arco/backend/app/types"
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

	// Current state (ADT enum from statemachine package)
	State statemachine.RepositoryState `json:"state"`

	// Metadata
	ArchiveCount    int        `json:"archiveCount"`
	LastBackupTime  *time.Time `json:"lastBackupTime,omitempty"`
	LastBackupError string     `json:"lastBackupError,omitempty"`
	StorageUsed     int64      `json:"storageUsed"`
}

// GetState implements the statemachine.Repository interface
func (r *Repository) GetState() statemachine.RepositoryState {
	return r.State
}

// GetID implements the statemachine.Repository interface
func (r *Repository) GetID() int {
	return r.ID
}

// RepositoryWithQueue extends Repository with queue information for frontend
type RepositoryWithQueue struct {
	Repository       `json:",inline"`
	QueuedOperations []QueuedOperation `json:"queuedOperations"`
	ActiveOperation  *QueuedOperation  `json:"activeOperation,omitempty"`
}

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
	ID         string          `json:"id"`         // Unique operation ID (UUID) - enables idempotency and deduplication
	Operation  Operation       `json:"operation"`  // ADT containing type and parameters
	Status     OperationStatus `json:"status"`     // ADT containing status, progress, error
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