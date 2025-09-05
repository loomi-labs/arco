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
