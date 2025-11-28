//go:generate go run ../../../scripts/generate_adts.go

package statemachine

import (
	"github.com/chris-tomich/adtenum"
	"github.com/loomi-labs/arco/backend/app/types"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/negrel/assert"
)

// ============================================================================
// OPERATION ADT
// ============================================================================

type Backup struct {
	BackupID types.BackupId            `json:"backupId"`
	Progress *borgtypes.BackupProgress `json:"progress,omitempty"`
}

type Prune struct {
	BackupID types.BackupId `json:"backupId"`
}

type Delete struct {
	RepositoryID int `json:"repositoryId"`
}

type ArchiveRefresh struct {
	RepositoryID int `json:"repositoryId"`
}

type ArchiveDelete struct {
	ArchiveID int `json:"archiveId"`
}

type ArchiveRename struct {
	ArchiveID int    `json:"archiveId"`
	Prefix    string `json:"prefix"`
	Name      string `json:"name"`
}

type ArchiveComment struct {
	ArchiveID int    `json:"archiveId"`
	Comment   string `json:"comment"`
}

type Mount struct {
	RepositoryID int    `json:"repositoryId"`
	MountPath    string `json:"mountPath"`
}

type MountArchive struct {
	ArchiveID int    `json:"archiveId"`
	MountPath string `json:"mountPath"`
}

type Unmount struct {
	RepositoryID int    `json:"repositoryId"`
	MountPath    string `json:"mountPath"`
}

type UnmountArchive struct {
	ArchiveID int    `json:"archiveId"`
	MountPath string `json:"mountPath"`
}

type ExaminePrune struct {
	BackupID    types.BackupId             `json:"backupId"`
	PruningRule *ent.PruningRule           `json:"pruningRule"`
	SaveResults bool                       `json:"saveResults"`
	ResultCh    chan borgtypes.PruneResult `json:"-"` // Channel to receive results
}

type Check struct {
	RepositoryID      int  `json:"repositoryId"`
	QuickVerification bool `json:"quickVerification"`
}

// Operation ADT definition
type Operation adtenum.Enum[Operation]

// Implement adtVariant marker interface for all operation structs
func (Backup) isADTVariant() Operation         { var zero Operation; return zero }
func (Prune) isADTVariant() Operation          { var zero Operation; return zero }
func (Delete) isADTVariant() Operation         { var zero Operation; return zero }
func (ArchiveRefresh) isADTVariant() Operation { var zero Operation; return zero }
func (ArchiveDelete) isADTVariant() Operation  { var zero Operation; return zero }
func (ArchiveRename) isADTVariant() Operation  { var zero Operation; return zero }
func (ArchiveComment) isADTVariant() Operation { var zero Operation; return zero }
func (Mount) isADTVariant() Operation          { var zero Operation; return zero }
func (MountArchive) isADTVariant() Operation   { var zero Operation; return zero }
func (Unmount) isADTVariant() Operation        { var zero Operation; return zero }
func (UnmountArchive) isADTVariant() Operation { var zero Operation; return zero }
func (ExaminePrune) isADTVariant() Operation   { var zero Operation; return zero }
func (Check) isADTVariant() Operation          { var zero Operation; return zero }

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
	switch GetOperationType(op) {
	case OperationTypeBackup, OperationTypePrune, OperationTypeDelete, OperationTypeCheck:
		return WeightHeavy
	case OperationTypeArchiveRefresh, OperationTypeArchiveDelete, OperationTypeArchiveRename, OperationTypeArchiveComment, OperationTypeMount, OperationTypeMountArchive, OperationTypeUnmount, OperationTypeUnmountArchive, OperationTypeExaminePrune:
		return WeightLight
	default:
		assert.Fail("Unhandled OperationType in GetOperationWeight")
		return WeightLight
	}
}
