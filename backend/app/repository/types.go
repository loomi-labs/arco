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

	// Current state (union type for Wails3 serialization)
	State statemachine.RepositoryStateUnion `json:"state"`

	// Metadata
	ArchiveCount      int        `json:"archiveCount"`
	LastBackupTime    *time.Time `json:"lastBackupTime,omitempty"`
	LastBackupError   string     `json:"lastBackupError,omitempty"`
	LastBackupWarning string     `json:"lastBackupWarning,omitempty"`
	StorageUsed       int64      `json:"storageUsed"`
}

// GetState implements the statemachine.Repository interface
func (r *Repository) GetState() statemachine.RepositoryState {
	return r.State.ToRepositoryState()
}

// GetID implements the statemachine.Repository interface
func (r *Repository) GetID() int {
	return r.ID
}

// RepositoryWithQueue extends Repository with queue information for frontend
type RepositoryWithQueue struct {
	Repository       `json:",inline"`
	QueuedOperations []*QueuedOperation `json:"queuedOperations"`
	ActiveOperation  *QueuedOperation   `json:"activeOperation,omitempty"`
}

// ============================================================================
// OPERATION STATUS ADT
// ============================================================================

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

// Implement adtVariant marker interface for all status structs
func (StatusQueued) isADTVariant() OperationStatus    { var zero OperationStatus; return zero }
func (StatusRunning) isADTVariant() OperationStatus   { var zero OperationStatus; return zero }
func (StatusCompleted) isADTVariant() OperationStatus { var zero OperationStatus; return zero }
func (StatusFailed) isADTVariant() OperationStatus    { var zero OperationStatus; return zero }
func (StatusExpired) isADTVariant() OperationStatus   { var zero OperationStatus; return zero }

// ============================================================================
// QUEUED OPERATION
// ============================================================================

// QueuedOperation represents a queued repository operation
type QueuedOperation struct {
	ID         string                 `json:"id"`        // Unique operation ID (UUID) - enables idempotency and deduplication
	Operation  statemachine.Operation `json:"operation"` // ADT containing type and parameters
	Status     OperationStatus        `json:"status"`    // ADT containing status, progress, error
	RepoID     int                    `json:"repoId"`
	CreatedAt  time.Time              `json:"createdAt"`
	ValidUntil time.Time              `json:"validUntil"` // Auto-expire if not started
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
