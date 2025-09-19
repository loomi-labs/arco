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

	// Repository type with associated data
	Type LocationUnion `json:"type"`

	// Current state
	State statemachine.RepositoryStateUnion `json:"state"`

	// Metadata
	ArchiveCount      int        `json:"archiveCount"`
	LastBackupTime    *time.Time `json:"lastBackupTime,omitempty"`
	LastBackupError   string     `json:"lastBackupError,omitempty"`
	LastBackupWarning string     `json:"lastBackupWarning,omitempty"`
	StorageUsed       int64      `json:"storageUsed"`
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
// REPOSITORY TYPE ADT
// ============================================================================

// Repository type variants
type Local struct{}
type Remote struct{}
type ArcoCloud struct {
	CloudID string `json:"cloudId"`
}

// Location ADT definition
type Location adtenum.Enum[Location]

// Implement adtVariant marker interface for all type structs
func (Local) isADTVariant() Location     { var zero Location; return zero }
func (Remote) isADTVariant() Location    { var zero Location; return zero }
func (ArcoCloud) isADTVariant() Location { var zero Location; return zero }

// ============================================================================
// OPERATION STATUS ADT
// ============================================================================

type Queued struct {
	Position int `json:"position"` // Position in queue
}

type Running struct {
	Progress  *Progress `json:"progress,omitempty"`
	StartedAt time.Time `json:"startedAt"`
}

type Completed struct {
	CompletedAt time.Time `json:"completedAt"`
}

type Failed struct {
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failedAt"`
}

type Expired struct {
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
func (Queued) isADTVariant() OperationStatus    { var zero OperationStatus; return zero }
func (Running) isADTVariant() OperationStatus   { var zero OperationStatus; return zero }
func (Completed) isADTVariant() OperationStatus { var zero OperationStatus; return zero }
func (Failed) isADTVariant() OperationStatus    { var zero OperationStatus; return zero }
func (Expired) isADTVariant() OperationStatus   { var zero OperationStatus; return zero }

// ============================================================================
// QUEUED OPERATION
// ============================================================================

// QueuedOperation represents a queued repository operation
type QueuedOperation struct {
	ID              string                 `json:"id"` // Unique operation ID (UUID) - enables idempotency and deduplication
	RepoID          int                    `json:"repoId"`
	BackupProfileID *int                   `json:"backupProfileId"`
	Operation       statemachine.Operation `json:"operation"` // ADT containing type and parameters
	Status          OperationStatus        `json:"status"`    // ADT containing status, progress, error
	CreatedAt       time.Time              `json:"createdAt"`
	ValidUntil      *time.Time             `json:"validUntil"` // Auto-expire if not started
	Immediate       bool                   `json:"immediate"`  // Must start immediately or fail
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

// testRepoConnectionResult represents the internal result of testing repository connection
type testRepoConnectionResult struct {
	Success         bool
	IsPasswordValid bool
	IsBorgRepo      bool
}

// BackupProfileFilter represents filters for backup profiles
type BackupProfileFilter struct {
	Id              int    `json:"id,omitempty"`
	Name            string `json:"name"`
	IsAllFilter     bool   `json:"isAllFilter"`
	IsUnknownFilter bool   `json:"isUnknownFilter"`
}

// PaginatedArchivesRequest represents a request for paginated archives
type PaginatedArchivesRequest struct {
	// Required
	RepositoryId int `json:"repositoryId"`
	Page         int `json:"page"`
	PageSize     int `json:"pageSize"`
	// Optional
	BackupProfileFilter *BackupProfileFilter `json:"backupProfileFilter,omitempty"`
	Search              string               `json:"search,omitempty"`
	StartDate           time.Time            `json:"startDate,omitempty"`
	EndDate             time.Time            `json:"endDate,omitempty"`
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

// UpdateRequest represents fields that can be updated for a repository
type UpdateRequest struct {
	Name string `json:"name,omitempty"` // Repository name
}
