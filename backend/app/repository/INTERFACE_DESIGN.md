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
    
    // Current state (ADT enum)
    State       RepositoryState `json:"state"`
    
    // Metadata
    ArchiveCount       int         `json:"archiveCount"`
    LastBackupTime     *time.Time  `json:"lastBackupTime,omitempty"`
    LastBackupError    string      `json:"lastBackupError,omitempty"`
    StorageUsed        int64       `json:"storageUsed"`
}

// Extended struct for when frontend needs queue information
type RepositoryWithQueue struct {
    Repository       `json:",inline"`
    QueuedOperations []QueuedOperation `json:"queuedOperations"`
    ActiveOperation  *QueuedOperation  `json:"activeOperation,omitempty"`
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
func (v BackupVariant) EnumType() Operation { return v }
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
    ID         string          // Unique operation ID (UUID) - enables idempotency and deduplication
    Operation  Operation       // ADT containing type and parameters
    Status     OperationStatus // ADT containing status, progress, error
    RepoID     int
    CreatedAt  time.Time
    ValidUntil time.Time       // Auto-expire if not started
}
```

### Queue Management
```go
type QueueManager struct {
    queues        map[int]*RepositoryQueue  // RepoID -> Queue
    mu            sync.RWMutex
    
    // Cross-repository concurrency control
    maxHeavyOps   int                       // Max heavy operations across all repositories
    activeHeavy   map[int]*QueuedOperation  // RepoID -> active heavy operation
    activeLight   map[int]*QueuedOperation  // RepoID -> active light operation
}

// Operation weight classification
type OperationWeight int

const (
    WeightLight OperationWeight = iota  // Quick operations (refresh, rename, single archive delete)
    WeightHeavy                         // Resource-intensive operations (backup, prune, repo delete)
)

// Determine operation weight for concurrency control
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

type RepositoryQueue struct {
    repoID        int
    operations    map[string]*QueuedOperation  // By operation ID
    operationList []string                     // Ordered operation IDs (FIFO)
    active        *QueuedOperation              // ONE active operation per repository
    mu            sync.Mutex
    
    // Deduplication tracking
    activeBackups map[types.BackupId]string   // BackupID -> OperationID
    activeDeletes map[int]string              // ArchiveID -> OperationID
    hasRepoDelete bool                        // Only one repo delete allowed
}

// Queue operations
func (qm *QueueManager) GetQueue(repoID int) *RepositoryQueue
func (qm *QueueManager) GetCurrentState(repoID int) RepositoryState
func (q *RepositoryQueue) AddOperation(op *QueuedOperation) error
func (q *RepositoryQueue) GetOperations() []*QueuedOperation
func (q *RepositoryQueue) GetActive() *QueuedOperation
func (q *RepositoryQueue) FindOperation(opType Operation) string  // Returns existing operation ID
```

## Operation Uniqueness and Idempotency

### Duplicate Operation Handling
Operations are identified by their content to prevent duplicates:

- **Backup Operations**: Same BackupID cannot be queued multiple times
- **Delete Operations**: Only one delete per repository/archive allowed at a time
- **Archive Refresh**: Can be queued multiple times (beneficial after operations)
- **Archive Rename**: Same ArchiveID with different name/prefix creates new operation

### Idempotent Behavior
```go
// First call - creates new operation
opId1, _ := QueueBackup(ctx, backupId)  // Returns "uuid-1234"

// Second call - returns existing operation ID
opId2, _ := QueueBackup(ctx, backupId)  // Returns "uuid-1234" (same!)

// Check operation status
op, _ := GetOperation(ctx, opId1)
```

### Queue Conflict Resolution
```go
type RepositoryQueue struct {
    operations    map[string]*QueuedOperation // By operation ID
    activeBackups map[types.BackupId]string   // BackupID -> OperationID  
    activeDeletes map[int]string              // ArchiveID -> OperationID
    hasRepoDelete bool                        // Only one repo delete allowed
}

// Example: Check for existing backup before creating new operation
func (q *RepositoryQueue) FindBackupOperation(backupId types.BackupId) string {
    if operationId, exists := q.activeBackups[backupId]; exists {
        return operationId // Return existing operation ID
    }
    return "" // No existing operation
}
```

## Service Methods

### Core Repository
```go
Get(ctx, repoId int) (*Repository, error)
GetWithQueue(ctx, repoId int) (*RepositoryWithQueue, error)
All(ctx) ([]*Repository, error)
AllWithQueue(ctx) ([]*RepositoryWithQueue, error)
GetByBackupId(ctx, bId types.BackupId) (*Repository, error)
Create(ctx, name, location, password string, noPassword bool) (*Repository, error)
CreateCloudRepository(ctx, name, password string, location arcov1.RepositoryLocation) (*Repository, error)
Update(ctx, repoId int, updates map[string]interface{}) (*Repository, error)
Remove(ctx, id int) error // DB only
```

### Queued Operations
All queue methods return operation IDs and are idempotent:
```go
QueueDelete(ctx, id int) (operationId string, error)

QueueBackup(ctx, backupId types.BackupId) (operationId string, error)
QueueBackups(ctx, backupIds []types.BackupId) (operationIds []string, error)  // Convenience method - queues multiple individual backup operations

QueuePrune(ctx, backupId types.BackupId) (operationId string, error)

QueueArchiveRefresh(ctx, repoId int) (operationId string, error)
QueueArchiveDelete(ctx, archiveId int) (operationId string, error)
QueueArchiveRename(ctx, archiveId int, prefix, name string) (operationId string, error)
```

### Operation Management
```go
// Get operation by ID
GetOperation(ctx, operationId string) (*QueuedOperation, error)

// Cancel queued or running operation
CancelOperation(ctx, operationId string) error

// Get all operations for a repository
GetQueuedOperations(ctx, repoId int) ([]*QueuedOperation, error)

// Get operations by status
GetOperationsByStatus(ctx, repoId int, status OperationStatus) ([]*QueuedOperation, error)
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
FixStoredPassword(ctx, repoId int, password string) (FixStoredPasswordResult, error)

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

- **Separate QueueManager**: Manages all repository queues independently from repository data
- **Per-Repository Sequential**: Each repository processes operations one at a time (FIFO)
- **Cross-Repository Parallel**: Multiple repositories can run operations simultaneously
- **Operation Weight Control**: Heavy operations (backup, prune, delete) limited by `maxHeavyOps`
- **Light Operations**: Quick operations (refresh, rename) can always run
- **Operation Expiry**: Operations expire if not started before `ValidUntil`
- **State Coordination**: QueueManager coordinates with StateMachine for state transitions
- **Idempotent Operations**: Duplicate operations return existing operation ID
- **Deduplication**: Active operation tracking prevents conflicts (e.g., only one backup per BackupID)

### Concurrency Examples

**Low-end system (maxHeavyOps = 1)**:
- Repo A starts backup (heavy) → runs
- Repo B wants backup → waits in queue
- Repo C wants refresh (light) → runs immediately
- When Repo A backup finishes, Repo B backup starts

**Powerful system (maxHeavyOps = 4)**:
- Repo A, B, C, D start backups → all run in parallel
- Repo E wants backup → waits (limit reached)
- Repo F wants refresh → runs immediately

### Future Enhancements
- **Dynamic Limits**: Adjust `maxHeavyOps` based on system load

## Removed Methods
The following methods are consolidated into the Repository struct or removed:
- GetState, GetNbrOfArchives, GetLastBackupErrorMsg, GetLocked
- GetWithActiveMounts, GetRepoMountState, GetArchiveMountStates
- GetConnectedRemoteHosts, GetLastArchiveByRepoId/BackupId
- SaveIntegrityCheckSettings, RunBorgDelete
- GetQueueStatus, GetOperationStatus, CancelOperation, PrioritizeOperation
- StartBackupJob(s), StartPruneJob (replaced by Queue* methods)