# State Machine Package

Type-safe state machine for repository operations using ADT (Algebraic Data Type) pattern for exhaustive type checking and safe state transitions.

## Core Concepts

### State Machine
- **Validates transitions** between repository states before they occur
- **Guard functions** enable conditional transitions (e.g., "can transition to Queued only if operations exist")
- **Reflection-based** transition lookup using state types
- **Thread-safe** with RWMutex protection

### ADT Pattern
- **RepositoryState** - 9 possible states as type-safe variants
- **Operation** - 11 operation types as type-safe variants
- **Exhaustive checking** - compiler ensures all cases are handled
- **Generated code** - Union types and constructors auto-generated (see backend/CLAUDE.md)

### Cancellable Operations
- Active states (BackingUp, Pruning, Deleting, Refreshing) embed `cancelCtx`
- Cancel function accessible via `GetCancel(state)`
- Context extractable via `GetCancelCtxOrDefault(defaultCtx, state)`
- Enables graceful operation abortion

## Repository States

### Active States (Cancellable)
- **BackingUp** - Running backup operation with progress tracking
- **Pruning** - Running prune operation for backup profile
- **Deleting** - Deleting archive or entire repository
- **Refreshing** - Refreshing archive list from borg
- **Mounting** - Mounting repository or archive (transitions to Mounted)

### Non-Active States
- **Idle** - No operations running or queued
- **Queued** - Operations waiting (contains next operation and queue length)
- **Mounted** - Repository/archive is mounted (contains mount info array)
- **Error** - Operation failed (contains error type, message, and recovery action)

## Operation Types

### Heavy Operations (Limited Concurrency)
- **Backup** - Create new archive
- **Prune** - Delete old archives based on retention rules
- **Delete** - Delete entire repository

### Light Operations (No Concurrency Limits)
- **ArchiveRefresh** - Refresh archive list
- **ArchiveDelete** - Delete single archive
- **ArchiveRename** - Rename archive
- **Mount/MountArchive** - Mount repository or archive
- **Unmount/UnmountArchive** - Unmount repository or archive
- **ExaminePrune** - Dry-run prune to preview deletions

## State Transitions

### Transition Rules
Defined in `initializeTransitions()` with guard functions:

```
From Idle → BackingUp, Pruning, Deleting, Refreshing, Mounting, Queued, Error
From Queued → BackingUp, Pruning, Deleting, Refreshing, Mounting, Idle, Error
From BackingUp → Idle, Queued (if queue not empty), Error
From Pruning → Idle, Queued (if queue not empty), Error
From Deleting → Idle, Queued (if queue not empty), Error
From Refreshing → Idle, Queued (if queue not empty), Error
From Mounting → Mounted, Queued (if queue not empty), Error
From Mounted → Refreshing (unmount), Error
From Error → Idle (error resolved)
```

### Guard Functions
- `nop` - Always allow transition
- `hasQueuedOps` - Allow only if repository has queued operations

### Validation
- `CanTransition(repoId, from, to)` - Check if transition is valid
- `Transition(repoId, from, to)` - Validate and perform transition
- `GetValidTransitions(repoId, currentState)` - List all valid next states

## Error Handling

### Error Types
- **ErrorTypeGeneral** - Unspecified error
- **ErrorTypeSSHKey** - SSH authentication failed
- **ErrorTypePassphrase** - Repository password incorrect
- **ErrorTypeLocked** - Repository is locked by another process

### Error Actions
- **ErrorActionNone** - No automatic recovery available
- **ErrorActionRegenerateSSH** - Regenerate SSH key (ArcoCloud repos)
- **ErrorActionChangePassphrase** - Update stored password
- **ErrorActionBreakLock** - Break repository lock

## Factory Methods

State creation with automatic context management:
- `CreateIdleState()` - No parameters
- `CreateQueuedState(nextOp, queueLen)` - Queue info
- `CreateBackingUpState(ctx, backupData)` - With cancel context
- `CreatePruningState(ctx, backupId)` - With cancel context
- `CreateDeletingState(ctx, archiveId)` - With cancel context
- `CreateRefreshingState(ctx)` - With cancel context
- `CreateMountingState(archiveId)` - Optional archive ID
- `CreateMountedState(mounts)` - Mount info array
- `CreateErrorState(errorType, msg, action)` - Error details

## Utility Functions

- `GetStateTypeName(state)` - Human-readable state name
- `IsActiveState(state)` - Check if operation is running
- `IsIdleState(state)` - Check if repository is idle
- `IsQueuedState(state)` - Check if operations are queued
- `IsMountedState(state)` - Check if mounted
- `IsErrorState(state)` - Check if in error
- `GetOperationWeight(op)` - Get Heavy or Light weight
