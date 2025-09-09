package statemachine

import (
	"github.com/chris-tomich/adtenum"
	"github.com/loomi-labs/arco/backend/app/types"
)

// ============================================================================
// OPERATION ADT
// ============================================================================

type OpBackup struct {
	BackupID types.BackupId `json:"backupId"`
}

type OpPrune struct {
	BackupID types.BackupId `json:"backupId"`
}

type OpDelete struct {
	RepositoryID int `json:"repositoryId"`
}

type OpArchiveRefresh struct {
	RepositoryID int `json:"repositoryId"`
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

// Implement adtVariant marker interface for all operation structs
func (OpBackup) isADTVariant() Operation         { var zero Operation; return zero }
func (OpPrune) isADTVariant() Operation          { var zero Operation; return zero }
func (OpDelete) isADTVariant() Operation         { var zero Operation; return zero }
func (OpArchiveRefresh) isADTVariant() Operation { var zero Operation; return zero }
func (OpArchiveDelete) isADTVariant() Operation  { var zero Operation; return zero }
func (OpArchiveRename) isADTVariant() Operation  { var zero Operation; return zero }

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
