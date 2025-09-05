# Repository Service Interface Design

## Core Data Structures

### Repository Struct
```go
type Repository struct {
    // Core fields
    ID          int       `json:"id"`
    Name        string    `json:"name"`
    URL         string    `json:"url"`
    
    // Cloud info
    IsCloud     bool      `json:"isCloud"`
    CloudID     string    `json:"cloudId,omitempty"`
    
    // State (ADT enum)
    State       RepositoryState `json:"state"`
    
    // Metadata
    ArchiveCount       int         `json:"archiveCount"`
    LastBackupTime     *time.Time  `json:"lastBackupTime,omitempty"`
    LastBackupError    string      `json:"lastBackupError,omitempty"`
    StorageUsed        int64       `json:"storageUsed"`
    
    // Operations
    QueuedOperations   []QueuedOperation `json:"queuedOperations"`
    ActiveOperation    *QueuedOperation  `json:"activeOperation,omitempty"`
}
```

### State ADT (Algebraic Data Type)
Using `github.com/chris-tomich/adtenum` for type-safe state management:

```go
// Context cancellation helper
type cancelCtx struct {
    ctx    context.Context
    cancel context.CancelFunc
}

// State variant structs (our custom data)
type StateIdle struct{}

type StateQueued struct {
    NextOperation OperationType
    QueueLength   int
}

type StateBackingUp struct {
    BackupID  types.BackupId
    Progress  *borgtypes.BackupProgress
    StartedAt time.Time
    cancelCtx cancelCtx // private context and cancel function
}

type StatePruning struct {
    BackupID  types.BackupId
    StartedAt time.Time
    cancelCtx cancelCtx // private context and cancel function
}

type StateDeleting struct {
    ArchiveID int
    StartedAt time.Time
    cancelCtx cancelCtx // private context and cancel function
}

type StateRefreshing struct {
    StartedAt time.Time
    cancelCtx cancelCtx // private context and cancel function
}

type StateMounted struct {
    MountType     MountType
    ArchiveID     *int
    MountPath     string
    ArchiveMounts map[int]*MountInfo
}

type StateError struct {
    ErrorType  ErrorType
    Message    string
    Action     ErrorAction
    OccurredAt time.Time
}

// Error types for repository operations
type ErrorType string

const (
    ErrorTypeSSHKey      ErrorType = "sshKey"
    ErrorTypePassphrase  ErrorType = "passphrase"
    ErrorTypeLocked      ErrorType = "locked"
)

// Actions that can be taken to resolve errors
type ErrorAction string

const (
    ErrorActionNone             ErrorAction = "none"
    ErrorActionRegenerateSSH    ErrorAction = "regenerateSSH"
    ErrorActionBreakLock        ErrorAction = "breakLock"
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
func (v IdleVariant) EnumType() RepositoryState { return v }
func (v QueuedVariant) EnumType() RepositoryState { return v }
func (v BackingUpVariant) EnumType() RepositoryState { return v }
func (v PruningVariant) EnumType() RepositoryState { return v }
func (v DeletingVariant) EnumType() RepositoryState { return v }
func (v RefreshingVariant) EnumType() RepositoryState { return v }
func (v MountedVariant) EnumType() RepositoryState { return v }
func (v ErrorVariant) EnumType() RepositoryState { return v }

// Usage example:
// state := NewStateQueued(StateQueued{NextOperation: BackupOp, QueueLength: 3})
// switch v := state.(type) {
//     case QueuedVariant:
//         queued := v()
//         fmt.Printf("Next op: %v, Queue length: %d", queued.NextOperation, queued.QueueLength)
//     // ... other cases
// }
```

### QueuedOperation
```go
// Operation variant structs (type-safe parameters)
type OpBackup struct {
    BackupID types.BackupId
}

type OpBackups struct {
    BackupIDs []types.BackupId
}

type OpPrune struct {
    BackupID types.BackupId
}

type OpDelete struct {
    // Repository delete - no additional params
}

type OpArchiveRefresh struct {
    // Archive refresh - no additional params
}

type OpArchiveDelete struct {
    ArchiveID int
}

type OpArchiveRename struct {
    ArchiveID int
    Prefix    string
    Name      string
}

// Operation ADT definition
type Operation adtenum.Enum[Operation]

// Operation variant wrappers
type BackupVariant adtenum.OneVariantValue[OpBackup]
type BackupsVariant adtenum.OneVariantValue[OpBackups]
type PruneVariant adtenum.OneVariantValue[OpPrune]
type DeleteVariant adtenum.OneVariantValue[OpDelete]
type ArchiveRefreshVariant adtenum.OneVariantValue[OpArchiveRefresh]
type ArchiveDeleteVariant adtenum.OneVariantValue[OpArchiveDelete]
type ArchiveRenameVariant adtenum.OneVariantValue[OpArchiveRename]

// Operation constructors
var NewOpBackup func(OpBackup) BackupVariant = adtenum.CreateOneVariantValueConstructor[BackupVariant]()
var NewOpBackups func(OpBackups) BackupsVariant = adtenum.CreateOneVariantValueConstructor[BackupsVariant]()
var NewOpPrune func(OpPrune) PruneVariant = adtenum.CreateOneVariantValueConstructor[PruneVariant]()
var NewOpDelete func(OpDelete) DeleteVariant = adtenum.CreateOneVariantValueConstructor[DeleteVariant]()
var NewOpArchiveRefresh func(OpArchiveRefresh) ArchiveRefreshVariant = adtenum.CreateOneVariantValueConstructor[ArchiveRefreshVariant]()
var NewOpArchiveDelete func(OpArchiveDelete) ArchiveDeleteVariant = adtenum.CreateOneVariantValueConstructor[ArchiveDeleteVariant]()
var NewOpArchiveRename func(OpArchiveRename) ArchiveRenameVariant = adtenum.CreateOneVariantValueConstructor[ArchiveRenameVariant]()

// Implement EnumType for operation variants
func (v BackupVariant) EnumType() Operation { return v }
func (v BackupsVariant) EnumType() Operation { return v }
func (v PruneVariant) EnumType() Operation { return v }
func (v DeleteVariant) EnumType() Operation { return v }
func (v ArchiveRefreshVariant) EnumType() Operation { return v }
func (v ArchiveDeleteVariant) EnumType() Operation { return v }
func (v ArchiveRenameVariant) EnumType() Operation { return v }

// Status variant structs (type-safe status information)
type StatusQueued struct {
    Position int // Position in queue
}

type StatusRunning struct {
    Progress  *Progress
    StartedAt time.Time
}

type StatusCompleted struct {
    CompletedAt time.Time
}

type StatusFailed struct {
    Error     string
    FailedAt  time.Time
    CanRetry  bool
}

type StatusExpired struct {
    ExpiredAt time.Time
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
func (v QueuedStatusVariant) EnumType() OperationStatus { return v }
func (v RunningStatusVariant) EnumType() OperationStatus { return v }
func (v CompletedStatusVariant) EnumType() OperationStatus { return v }
func (v FailedStatusVariant) EnumType() OperationStatus { return v }
func (v ExpiredStatusVariant) EnumType() OperationStatus { return v }

// Simplified QueuedOperation struct
type QueuedOperation struct {
    ID         string          // Unique operation ID
    Operation  Operation       // ADT containing type and parameters
    Status     OperationStatus // ADT containing status, progress, error
    RepoID     int
    CreatedAt  time.Time
    ValidUntil time.Time       // Auto-expire if not started
}
```

## Service Methods

### Core Repository
```go
Get(ctx, repoId int) (*Repository, error)
All(ctx) ([]*Repository, error)
GetByBackupId(ctx, bId types.BackupId) (*Repository, error)
Create(ctx, name, location, password string, noPassword bool) (*Repository, error)
CreateCloudRepository(ctx, name, password string, location arcov1.RepositoryLocation) (*Repository, error)
Update(ctx, repoId int, updates map[string]interface{}) (*Repository, error)
Remove(ctx, id int) error // DB only
```

### Queued Operations
```go
QueueDelete(ctx, id int) error

QueueBackup(ctx, backupId types.BackupId) (operationId string, error)
QueueBackups(ctx, backupIds []types.BackupId) (operationIds []string, error)

QueuePrune(ctx, backupId types.BackupId) (operationId string, error)

QueueArchiveRefresh(ctx, repoId int) (operationId string, error)
QueueArchiveDelete(ctx, archiveId int) (operationId string, error)
QueueArchiveRename(ctx, archiveId int, prefix, name string) (operationId string, error)
```

### Immediate Operations
```go
// Backup control
AbortBackup(ctx, backupId types.BackupId) error
AbortBackups(ctx, backupIds []types.BackupId) error

// Mount operations
Mount(ctx, repoId int) (*MountState, error)
MountArchive(ctx, archiveId int) (*MountState, error)
Unmount(ctx, repoId int) (*MountState, error)
UnmountArchive(ctx, archiveId int) (*MountState, error)
UnmountAllForRepos(ctx, repoIds []int) error

// Analysis
ExaminePrunes(ctx, backupProfileId int, pruningRule *ent.PruningRule) ([]ExaminePruningResult, error)

// Configuration
ChangePassword(ctx, repoId int, password string) error

// Error recovery
RegenerateSSHKey(ctx) error
BreakLock(ctx, repoId int) error
```

### Archive Methods
```go
GetArchive(ctx, id int) (*Archive, error)
GetPaginatedArchives(ctx, req *PaginatedArchivesRequest) (*PaginatedArchivesResponse, error)
GetPruningDates(ctx, archiveIds []int) (PruningDates, error)
```

### Validation
```go
ValidateRepoName(ctx, name string) (string, error)
ValidateRepoPath(ctx, path string, isLocal bool) (string, error)
ValidateArchiveName(ctx, archiveId int, prefix, name string) (string, error)
TestRepoConnection(ctx, path, password string) (TestRepoConnectionResult, error)
IsBorgRepository(path string) bool
```

## Queue Implementation Notes

- Each repository has its own queue
- Operations expire if not started before `ValidUntil`
- Worker goroutines process queues sequentially per repository
- State transitions are validated before execution

## Removed Methods
The following methods are consolidated into the Repository struct or removed:
- GetState, GetNbrOfArchives, GetLastBackupErrorMsg, GetLocked
- GetWithActiveMounts, GetRepoMountState, GetArchiveMountStates
- GetConnectedRemoteHosts, GetLastArchiveByRepoId/BackupId
- SaveIntegrityCheckSettings, RunBorgDelete
- GetQueueStatus, GetOperationStatus, CancelOperation, PrioritizeOperation
- StartBackupJob(s), StartPruneJob (replaced by Queue* methods)