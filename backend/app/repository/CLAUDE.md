# Repository Package

This package manages Borg repositories, archives, and backup operations with a queued operation system.

## Public Service Methods

Methods on `(s *Service)` are exposed to the frontend via Wails3 bindings.

### Core Repository Operations
- `All(ctx) ([]*Repository, error)` - Get all repositories with state
- `Get(ctx, repoId) (*Repository, error)` - Get repository by ID
- `GetWithQueue(ctx, repoId) (*RepositoryWithQueue, error)` - Get repository with queue info
- `Create(ctx, name, location, password, noPassword) (*Repository, error)` - Create local/remote repository
- `CreateCloudRepository(ctx, name, password, location) (*Repository, error)` - Create ArcoCloud repository
- `Update(ctx, repoId, *UpdateRequest) (*Repository, error)` - Update repository fields
- `Remove(ctx, id) error` - Remove from database only
- `Delete(ctx, id) error` - Delete repository completely (queues operation)

### Archive Operations
- `RefreshArchives(ctx, repoId) (string, error)` - Queue archive refresh operation
- `GetPaginatedArchives(ctx, *PaginatedArchivesRequest) (*PaginatedArchivesResponse, error)` - Get archives with filters
- `GetPruningDates(ctx, archiveIds) (PruningDates, error)` - Get prune dates for archives
- `GetLastArchiveByRepoId(ctx, repoId) (*ent.Archive, error)` - Get most recent archive
- `GetLastArchiveByBackupId(ctx, backupId) (*ent.Archive, error)` - Get last archive for backup profile

### Queue Operations
- `QueueBackup(ctx, backupId) (string, error)` - Queue backup operation
- `QueueBackups(ctx, []backupId) ([]string, error)` - Queue multiple backups
- `QueuePrune(ctx, backupId) (string, error)` - Queue prune operation
- `QueueArchiveDelete(ctx, archiveId) (string, error)` - Queue archive deletion
- `QueueArchiveRename(ctx, archiveId, name) (string, error)` - Queue archive rename

### Operation Management
- `GetActiveOperation(ctx, repoId, *operationType) (*SerializableQueuedOperation, error)` - Get active operation
- `CancelOperation(ctx, repositoryId, operationId) error` - Cancel queued/running operation
- `GetQueuedOperations(ctx, repoId, *operationType) ([]*SerializableQueuedOperation, error)` - Get queued operations

### Immediate Operations
- `AbortBackup(ctx, backupId) error` - Abort running backup
- `AbortBackups(ctx, []backupId) error` - Abort multiple backups
- `Mount(ctx, repoId) (string, error)` - Mount repository (opens file manager)
- `MountArchive(ctx, archiveId) (string, error)` - Mount archive (opens file manager)
- `Unmount(ctx, repoId) (string, error)` - Unmount repository
- `UnmountArchive(ctx, archiveId) (string, error)` - Unmount archive
- `UnmountAllForRepos(ctx, []repoId) []error` - Unmount all mounts for repositories

### Prune Operations
- `ExaminePrunes(ctx, backupProfileId, *pruningRule, saveResults) ([]ExaminePruningResult, error)` - Examine what would be pruned

### Validation
- `ValidateRepoName(ctx, name) (string, error)` - Validate repository name (returns error message or "")
- `ValidateRepoPath(ctx, path, isLocal) (string, error)` - Validate repository path
- `ValidateArchiveName(ctx, archiveId, name) (string, error)` - Validate archive name
- `TestRepoConnection(ctx, path, password) (TestRepoConnectionResult, error)` - Test repository connection
- `IsBorgRepository(path) bool` - Check if path is borg repository

### Password/Key Management
- `FixStoredPassword(ctx, repoId, password) (FixStoredPasswordResult, error)` - Update stored password
- `RegenerateSSHKey(ctx) error` - Regenerate SSH key for ArcoCloud
- `BreakLock(ctx, repoId) error` - Break repository lock

### Backup State
- `GetBackupButtonStatus(ctx, []backupId) (BackupButtonStatus, error)` - Get combined backup button state
- `GetCombinedBackupProgress(ctx, []backupId) (*borgtypes.BackupProgress, error)` - Get combined progress
- `GetBackupState(ctx, backupId) (*statemachine.Backup, error)` - Get backup operation state
- `GetLastBackupErrorMsgByBackupId(ctx, backupId) (string, error)` - Get last error for backup

### Utilities
- `GetConnectedRemoteHosts(ctx) ([]string, error)` - Get connected remote SSH hosts

## Package Elements

### Services
- **Service** - Main service struct with methods exposed to frontend. Manages repositories, archives, and operations
- **ServiceInternal** - Backend-only methods (sync operations, init). Wraps Service with additional internal methods

### Operation Management
- **QueueManager** - Manages operation queues across all repositories. Handles:
  - Concurrency limits (max heavy operations across repos)
  - Operation lifecycle (queueing, starting, completing, canceling)
  - State transitions via RepositoryStateMachine
  - Repository state tracking (idle, backing up, mounted, error, etc.)
  - Progress updates and event emission
- **RepositoryQueue** - Per-repository FIFO operation queue with:
  - Deduplication (one backup per BackupId, one delete per archive, one repo delete)
  - Position tracking
  - Active/queued operation management
  - Expiration handling

### Cloud Integration
- **CloudRepositoryClient** - ArcoCloud repository management:
  - SSH key generation/management
  - Repository CRUD operations via RPC
  - Error handling with rate limiting and retry logic

### ADT Types
Type-safe algebraic data types for state modeling (see backend/CLAUDE.md for ADT system details):
- **Location** - Repository type: Local, Remote, ArcoCloud
- **OperationStatus** - Queue status: Queued, Running, Completed, Failed, Expired
- **ArchiveRenameState** - Rename state: RenameNone, RenameQueued, RenameActive
- **ArchiveDeleteState** - Delete state: DeleteNone, DeleteQueued, DeleteActive

### Data Structures
- **Repository** - Consolidated repository with state, metadata, and type info
- **RepositoryWithQueue** - Repository plus queue/active operations
- **QueuedOperation** - Operation with ID, status, expiration, and immediate flag
- **ArchiveWithPendingChanges** - Archive with rename/delete operation states
