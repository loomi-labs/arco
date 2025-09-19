package statemachine

import (
	"github.com/chris-tomich/adtenum"
	"github.com/loomi-labs/arco/backend/app/types"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
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

// Operation ADT definition
type Operation adtenum.Enum[Operation]

// Implement adtVariant marker interface for all operation structs
func (Backup) isADTVariant() Operation         { var zero Operation; return zero }
func (Prune) isADTVariant() Operation          { var zero Operation; return zero }
func (Delete) isADTVariant() Operation         { var zero Operation; return zero }
func (ArchiveRefresh) isADTVariant() Operation { var zero Operation; return zero }
func (ArchiveDelete) isADTVariant() Operation  { var zero Operation; return zero }
func (ArchiveRename) isADTVariant() Operation  { var zero Operation; return zero }
func (Mount) isADTVariant() Operation          { var zero Operation; return zero }
func (MountArchive) isADTVariant() Operation   { var zero Operation; return zero }
func (Unmount) isADTVariant() Operation        { var zero Operation; return zero }
func (UnmountArchive) isADTVariant() Operation { var zero Operation; return zero }

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
	case ArchiveRefreshVariant, ArchiveDeleteVariant, ArchiveRenameVariant, MountVariant, MountArchiveVariant, UnmountVariant, UnmountArchiveVariant:
		return WeightLight
	default:
		return WeightLight
	}
}
