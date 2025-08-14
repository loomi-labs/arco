package state

import (
	"context"
	"fmt"
	"sync"

	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/types"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/negrel/assert"
	"go.uber.org/zap"
)

type State struct {
	log           *zap.SugaredLogger
	mu            sync.RWMutex
	eventEmitter  types.EventEmitter
	notifications []types.Notification

	startupState    *StartupState
	authState       *AuthState
	checkoutSession *arcov1.CreateCheckoutSessionResponse
	checkoutResult  *CheckoutResult
	repoStates      map[int]*RepoState
	backupStates    map[types.BackupId]*BackupState
	pruneStates     map[types.BackupId]*PruneState

	repoMounts    map[int]*platform.MountState         // map of repository ID to mount state
	archiveMounts map[int]map[int]*platform.MountState // maps of [repository ID][archive ID] to mount state
}

type StartupStatus string

const (
	StartupStatusUnknown                StartupStatus = "unknown"
	StartupStatusCheckingForUpdates     StartupStatus = "checkingForUpdates"
	StartupStatusApplyingUpdates        StartupStatus = "applyingUpdates"
	StartupStatusRestartingArco         StartupStatus = "restartingArco"
	StartupStatusInitializingDatabase   StartupStatus = "initializingDatabase"
	StartupStatusCheckingForBorgUpdates StartupStatus = "checkingForBorgUpdates"
	StartupStatusUpdatingBorg           StartupStatus = "updatingBorg"
	StartupStatusInitializingApp        StartupStatus = "initializingApp"
	StartupStatusReady                  StartupStatus = "ready"
)

var AvailableStartupStatuses = []StartupStatus{
	StartupStatusUnknown,
	StartupStatusCheckingForUpdates,
	StartupStatusApplyingUpdates,
	StartupStatusRestartingArco,
	StartupStatusInitializingDatabase,
	StartupStatusCheckingForBorgUpdates,
	StartupStatusUpdatingBorg,
	StartupStatusInitializingApp,
	StartupStatusReady,
}

func (ss StartupStatus) String() string {
	return string(ss)
}

type StartupState struct {
	Error  string        `json:"error"`
	Status StartupStatus `json:"status"`
}

type AuthState struct {
	IsAuthenticated bool `json:"isAuthenticated"`
}

type CheckoutResultStatus string

const (
	CheckoutStatusPending   CheckoutResultStatus = "pending"
	CheckoutStatusCompleted CheckoutResultStatus = "completed"
	CheckoutStatusFailed    CheckoutResultStatus = "failed"
	CheckoutStatusTimeout   CheckoutResultStatus = "timeout"
)

type CheckoutResult struct {
	Status         CheckoutResultStatus `json:"status"`
	ErrorMessage   string               `json:"errorMessage,omitempty"`
	SubscriptionID string               `json:"subscriptionId,omitempty"`
}

type cancelCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type KeepArchive struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type PruneJobResult struct {
	PruneArchives []int         `json:"prune_archives"`
	KeepArchives  []KeepArchive `json:"keep_archives"`
}

type RepoStatus string

const (
	RepoStatusIdle                RepoStatus = "idle"
	RepoStatusBackingUp           RepoStatus = "backingUp"
	RepoStatusPruning             RepoStatus = "pruning"
	RepoStatusDeleting            RepoStatus = "deleting"
	RepoStatusMounted             RepoStatus = "mounted"
	RepoStatusPerformingOperation RepoStatus = "performingOperation"
	RepoStatusError               RepoStatus = "error"
)

var AvailableRepoStatuses = []RepoStatus{
	RepoStatusIdle,
	RepoStatusBackingUp,
	RepoStatusPruning,
	RepoStatusDeleting,
	RepoStatusMounted,
	RepoStatusPerformingOperation,
	RepoStatusError,
}

func (rs RepoStatus) String() string {
	return string(rs)
}

type RepoErrorType string

const (
	RepoErrorTypeNone        RepoErrorType = "none"
	RepoErrorTypeSSHKey      RepoErrorType = "sshKey"
	RepoErrorTypePassphrase  RepoErrorType = "passphrase"
	RepoErrorTypeLockTimeout RepoErrorType = "lockTimeout"
)

func (ret RepoErrorType) String() string {
	return string(ret)
}

type RepoErrorAction string

const (
	RepoErrorActionNone             RepoErrorAction = "none"
	RepoErrorActionRegenerateSSH    RepoErrorAction = "regenerateSSH"
	RepoErrorActionUnlockRepository RepoErrorAction = "unlockRepository"
)

func (rea RepoErrorAction) String() string {
	return string(rea)
}

type RepoState struct {
	mutex          *sync.Mutex
	Status         RepoStatus      `json:"status"`
	ErrorType      RepoErrorType   `json:"errorType"`
	ErrorMessage   string          `json:"errorMessage"`
	ErrorAction    RepoErrorAction `json:"errorAction"`
	HasWarning     bool            `json:"hasWarning"`
	WarningMessage string          `json:"warningMessage"`
}

func newRepoState() *RepoState {
	return &RepoState{
		mutex:          &sync.Mutex{},
		Status:         RepoStatusIdle,
		ErrorType:      RepoErrorTypeNone,
		ErrorMessage:   "",
		ErrorAction:    RepoErrorActionNone,
		HasWarning:     false,
		WarningMessage: "",
	}
}

type BackupStatus string

const (
	BackupStatusIdle      BackupStatus = "idle"
	BackupStatusWaiting   BackupStatus = "waiting"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusCancelled BackupStatus = "cancelled"
	BackupStatusFailed    BackupStatus = "failed"
)

var AvailableBackupStatuses = []BackupStatus{
	BackupStatusIdle,
	BackupStatusWaiting,
	BackupStatusRunning,
	BackupStatusCompleted,
	BackupStatusCancelled,
	BackupStatusFailed,
}

func (bs BackupStatus) String() string {
	return string(bs)
}

type BackupState struct {
	*cancelCtx
	Status   BackupStatus              `json:"status"`
	Progress *borgtypes.BackupProgress `json:"progress,omitempty"`
	Error    string                    `json:"error,omitempty"`
}

func newBackupState() *BackupState {
	return &BackupState{
		cancelCtx: nil,
		Status:    BackupStatusIdle,
		Progress:  nil,
		Error:     "",
	}
}

type BackupButtonStatus string

const (
	BackupButtonStatusRunBackup BackupButtonStatus = "runBackup"
	BackupButtonStatusWaiting   BackupButtonStatus = "waiting"
	BackupButtonStatusAbort     BackupButtonStatus = "abort"
	BackupButtonStatusLocked    BackupButtonStatus = "locked"
	BackupButtonStatusUnmount   BackupButtonStatus = "unmount"
	BackupButtonStatusBusy      BackupButtonStatus = "busy"
)

var AvailableBackupButtonStatuses = []BackupButtonStatus{
	BackupButtonStatusRunBackup,
	BackupButtonStatusWaiting,
	BackupButtonStatusAbort,
	BackupButtonStatusLocked,
	BackupButtonStatusUnmount,
	BackupButtonStatusBusy,
}

func (b BackupButtonStatus) String() string {
	return string(b)
}

type PruningStatus string

const (
	PruningStatusIdle      PruningStatus = "idle"
	PruningStatusWaiting   PruningStatus = "waiting"
	PruningStatusRunning   PruningStatus = "running"
	PruningStatusCompleted PruningStatus = "completed"
	PruningStatusCancelled PruningStatus = "cancelled"
	PruningStatusFailed    PruningStatus = "failed"
)

var AvailablePruningStatuses = []PruningStatus{
	PruningStatusIdle,
	PruningStatusWaiting,
	PruningStatusRunning,
	PruningStatusCompleted,
	PruningStatusCancelled,
	PruningStatusFailed,
}

func (ps PruningStatus) String() string {
	return string(ps)
}

type PruneState struct {
	*cancelCtx
	Status PruningStatus   `json:"status"`
	Result *PruneJobResult `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

func newPruneState() *PruneState {
	return &PruneState{
		cancelCtx: nil,
		Status:    PruningStatusIdle,
		Result:    nil,
		Error:     "",
	}
}

func NewState(log *zap.SugaredLogger, eventEmitter types.EventEmitter) *State {
	return &State{
		log:           log,
		mu:            sync.RWMutex{},
		eventEmitter:  eventEmitter,
		notifications: []types.Notification{},

		startupState: &StartupState{
			Status: StartupStatusUnknown,
		},
		authState: &AuthState{
			IsAuthenticated: false,
		},
		repoStates:   make(map[int]*RepoState),
		backupStates: map[types.BackupId]*BackupState{},
		pruneStates:  map[types.BackupId]*PruneState{},

		repoMounts:    make(map[int]*platform.MountState),
		archiveMounts: make(map[int]map[int]*platform.MountState),
	}
}

func newCancelCtx(ctx context.Context) *cancelCtx {
	nCtx, cancel := context.WithCancel(ctx)
	return &cancelCtx{
		ctx:    nCtx,
		cancel: cancel,
	}
}

/***********************************/
/********** Startup Error **********/
/***********************************/

func (s *State) SetStartupStatus(ctx context.Context, status StartupStatus, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventStartupStateChanged.String())

	s.startupState.Status = status
	if err != nil {
		s.startupState.Error = err.Error()
	}
	// We never clear the error, it's only set once since the app should not recover from a startup error
}

func (s *State) GetStartupState() StartupState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *s.startupState
}

/***********************************/
/********** Repo States ************/
/***********************************/

func (s *State) CanPerformRepoOperation(repositoryId int) (canRun bool, reason string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rs, ok := s.repoStates[repositoryId]; ok {
		if rs.Status != RepoStatusIdle {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

// GetRepoLock returns the mutex for the given repository ID.
// The mutex has to be acquired before performing any operations on the repository.
//
// Usage:
// mutex := state.GetRepoLock(repoId)
// mutex.Lock() // Wait to acquire the mutex
// state.SetRepoStatus(repoId, &state.RepoStatePerformingOperation{})	// Set the repo state
// defer b.state.SetRepoStatus(bId.RepositoryId, &state.RepoStateIdle{}) // Set the repo state back to idle
func (s *State) GetRepoLock(repoId int) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repoStates[repoId]; !ok {
		s.setRepoState(repoId, RepoStatusIdle)
	}
	return s.repoStates[repoId].mutex
}

func (s *State) SetRepoStatus(ctx context.Context, repoId int, state RepoStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChanged.String()+fmt.Sprintf(":%d", repoId))

	s.setRepoState(repoId, state)
}

func (s *State) setRepoState(repoId int, state RepoStatus) {
	if _, ok := s.repoStates[repoId]; ok {
		s.repoStates[repoId].Status = state
	} else {
		// If the repository state doesn't exist, we create it
		s.repoStates[repoId] = newRepoState()
	}
}

func (s *State) GetRepoState(repoId int) RepoState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rs, ok := s.repoStates[repoId]; ok {
		return *rs
	}
	return *newRepoState()
}

func (s *State) SetRepoErrorState(ctx context.Context, repoId int, errorType RepoErrorType, errorMessage string, errorAction RepoErrorAction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChanged.String()+fmt.Sprintf(":%d", repoId))

	if _, ok := s.repoStates[repoId]; !ok {
		s.repoStates[repoId] = newRepoState()
	}

	s.repoStates[repoId].Status = RepoStatusError
	s.repoStates[repoId].ErrorType = errorType
	s.repoStates[repoId].ErrorMessage = errorMessage
	s.repoStates[repoId].ErrorAction = errorAction
}

func (s *State) ClearRepoErrorState(ctx context.Context, repoId int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChanged.String()+fmt.Sprintf(":%d", repoId))

	if _, ok := s.repoStates[repoId]; !ok {
		s.repoStates[repoId] = newRepoState()
	}

	s.repoStates[repoId].ErrorType = RepoErrorTypeNone
	s.repoStates[repoId].ErrorMessage = ""
	s.repoStates[repoId].ErrorAction = RepoErrorActionNone
	s.repoStates[repoId].Status = RepoStatusIdle
}

func (s *State) GetRepoErrorState(repoId int) (RepoErrorType, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rs, ok := s.repoStates[repoId]; ok {
		return rs.ErrorType, rs.ErrorMessage
	}
	return RepoErrorTypeNone, ""
}

func (s *State) SetRepoWarningState(ctx context.Context, repoId int, warningMessage string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChanged.String()+fmt.Sprintf(":%d", repoId))

	if _, ok := s.repoStates[repoId]; !ok {
		s.repoStates[repoId] = newRepoState()
	}

	s.repoStates[repoId].HasWarning = true
	s.repoStates[repoId].WarningMessage = warningMessage
}

func (s *State) ClearRepoWarningState(ctx context.Context, repoId int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChanged.String()+fmt.Sprintf(":%d", repoId))

	if _, ok := s.repoStates[repoId]; !ok {
		s.repoStates[repoId] = newRepoState()
	}

	s.repoStates[repoId].HasWarning = false
	s.repoStates[repoId].WarningMessage = ""
}

/***********************************/
/********** Backup States **********/
/***********************************/

func (s *State) CanRunBackup(id types.BackupId) (canRun bool, reason string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if bs, ok := s.backupStates[id]; ok {
		if bs.Status == BackupStatusRunning {
			return false, "Backup is already running"
		}
		if bs.Status == BackupStatusWaiting {
			return false, "Backup is already queued to run"
		}
	}
	if rs, ok := s.repoStates[id.RepositoryId]; ok {
		if rs.Status != RepoStatusIdle {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

func (s *State) SetBackupWaiting(ctx context.Context, bId types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventBackupStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusWaiting)
}

func (s *State) SetBackupRunning(ctx context.Context, bId types.BackupId) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentState, ok := s.backupStates[bId]
	if ok {
		if currentState.Status == BackupStatusRunning {
			// If the state is already running, we don't do anything
			return currentState.ctx
		}
	}

	defer s.eventEmitter.EmitEvent(ctx, types.EventBackupStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusRunning)
	s.backupStates[bId].cancelCtx = newCancelCtx(ctx)
	s.backupStates[bId].Progress = &borgtypes.BackupProgress{}
	s.backupStates[bId].Error = ""

	s.setRepoState(bId.RepositoryId, RepoStatusBackingUp)

	return s.backupStates[bId].ctx
}

func (s *State) SetBackupCompleted(ctx context.Context, bId types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventBackupStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusCompleted)
}

func (s *State) SetBackupCancelled(ctx context.Context, bId types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventBackupStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusCancelled)
}

func (s *State) SetBackupError(ctx context.Context, bId types.BackupId, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventBackupStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusFailed)
	s.backupStates[bId].Error = err.Error()
}

func (s *State) changeBackupState(bId types.BackupId, newState BackupStatus) {
	currentState, ok := s.backupStates[bId]
	if ok {
		if currentState.Status == BackupStatusRunning && newState != BackupStatusRunning {
			// If we are here it means:
			// - the current state is running
			// - the new state is not running (it's either completed, cancelled or errored)
			// Therefore we cancel the context to stop eventual running borg operations
			currentState.cancel()
			currentState.ctx = nil
			currentState.Progress = nil
		}
	} else {
		// If the backup state doesn't exist, we create it
		s.backupStates[bId] = newBackupState()
	}

	s.backupStates[bId].Status = newState
}

func (s *State) UpdateBackupProgress(ctx context.Context, bId types.BackupId, progress borgtypes.BackupProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventBackupStateChangedString(bId))

	if currentState, ok := s.backupStates[bId]; ok {
		if currentState.Status == BackupStatusRunning {
			currentState.Progress = &progress
		}
	}
}

func (s *State) getBackupState(id types.BackupId) BackupState {
	if bs, ok := s.backupStates[id]; ok {
		return *bs
	}
	return *newBackupState()
}

func (s *State) GetBackupState(id types.BackupId) BackupState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getBackupState(id)
}

func (s *State) GetCombinedBackupProgress(ids []types.BackupId) *borgtypes.BackupProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalFiles, processedFiles int
	found := false
	for _, id := range ids {
		if bs, ok := s.backupStates[id]; ok {
			if bs.Progress != nil {
				found = true
				totalFiles += bs.Progress.TotalFiles
				processedFiles += bs.Progress.ProcessedFiles
			}
		}
	}
	if !found {
		return nil
	}
	return &borgtypes.BackupProgress{
		TotalFiles:     totalFiles,
		ProcessedFiles: processedFiles,
	}
}

/***********************************/
/********** Prune Jobs ************/
/***********************************/

func (s *State) CanRunPrune(id types.BackupId) (canRun bool, reason string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if ps, ok := s.pruneStates[id]; ok {
		if ps.Status == PruningStatusRunning {
			return false, "Prune job is already running"
		}
		if ps.Status == PruningStatusWaiting {
			return false, "Prune job is already queued to run"
		}
	}
	if rs, ok := s.repoStates[id.RepositoryId]; ok {
		if rs.Status != RepoStatusIdle {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

func (s *State) SetPruneWaiting(ctx context.Context, bId types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventPruneStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changePruneState(bId, PruningStatusWaiting)
}

func (s *State) SetPruneRunning(ctx context.Context, bId types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentState, ok := s.pruneStates[bId]
	if ok {
		if currentState.Status == PruningStatusRunning {
			// If the state is already running, we don't do anything
			return
		}
	}

	defer s.eventEmitter.EmitEvent(ctx, types.EventPruneStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changePruneState(bId, PruningStatusRunning)
	s.pruneStates[bId].cancelCtx = newCancelCtx(ctx)
	s.pruneStates[bId].Error = ""

	s.setRepoState(bId.RepositoryId, RepoStatusPruning)
}

func (s *State) SetPruneCompleted(ctx context.Context, bId types.BackupId, result PruneJobResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventPruneStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changePruneState(bId, PruningStatusCompleted)
	s.pruneStates[bId].Result = &result
	s.setRepoState(bId.RepositoryId, RepoStatusIdle)
}

func (s *State) SetPruneCancelled(ctx context.Context, bId types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventPruneStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changePruneState(bId, PruningStatusCancelled)
	s.setRepoState(bId.RepositoryId, RepoStatusIdle)
}

func (s *State) SetPruneError(ctx context.Context, bId types.BackupId, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventPruneStateChangedString(bId))
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(bId.RepositoryId))
	assert.True(err != nil, "error must not be nil")

	s.changePruneState(bId, PruningStatusFailed)
	s.pruneStates[bId].Error = err.Error()
}

func (s *State) changePruneState(bId types.BackupId, newState PruningStatus) {
	currentState, ok := s.pruneStates[bId]
	if ok {
		if currentState.Status == PruningStatusRunning && newState != PruningStatusRunning {
			// If we are here it means:
			// - the current state is running
			// - the new state is not running (it's either completed, cancelled or errored)
			// Therefore we cancel the context to stop eventual running borg operations
			currentState.cancel()
			currentState.ctx = nil
		}
	} else {
		// If the prune state doesn't exist, we create it
		s.pruneStates[bId] = newPruneState()
	}

	s.pruneStates[bId].Status = newState
}

/***********************************/
/********** Delete Jobs ************/
/***********************************/

func (s *State) CanRunDeleteJob(repoId int) (canRun bool, reason string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rs, ok := s.repoStates[repoId]; ok {
		if rs.Status != RepoStatusIdle {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

/***********************************/
/********** Notifications **********/
/***********************************/

func (s *State) AddNotification(ctx context.Context, msg string, level types.NotificationLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventNotificationAvailable.String())

	s.notifications = append(s.notifications, types.Notification{
		Message: msg,
		Level:   level,
	})
}

func (s *State) GetAndDeleteNotifications() []types.Notification {
	s.mu.Lock()
	defer s.mu.Unlock()

	notifications := s.notifications
	s.notifications = []types.Notification{}
	return notifications
}

/***********************************/
/************* Mounts **************/
/***********************************/

func (s *State) CanMountRepo(id int) (canMount bool, reason string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rs, ok := s.repoStates[id]; ok {
		// We can only mount a repository if it's idle or mounting
		if rs.Status != RepoStatusIdle && rs.Status != RepoStatusMounted {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

func (s *State) SetRepoMount(ctx context.Context, repoId int, state *platform.MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(repoId))

	if _, ok := s.repoMounts[repoId]; !ok {
		s.repoMounts[repoId] = &platform.MountState{}
	}

	s.repoMounts[repoId].IsMounted = state.IsMounted
	s.repoMounts[repoId].MountPath = state.MountPath

	if state.IsMounted {
		s.setRepoState(repoId, RepoStatusMounted)
	} else {
		hasOtherMounts := false
		for _, aState := range s.archiveMounts[repoId] {
			if aState.IsMounted {
				hasOtherMounts = true
				break
			}
		}
		if !hasOtherMounts {
			s.setRepoState(repoId, RepoStatusIdle)
		}
	}
}

func (s *State) setArchiveMount(ctx context.Context, repoId int, archiveId int, state *platform.MountState) {
	if _, ok := s.archiveMounts[repoId]; !ok {
		s.archiveMounts[repoId] = make(map[int]*platform.MountState)
	}
	if _, ok := s.archiveMounts[repoId][archiveId]; !ok {
		s.archiveMounts[repoId][archiveId] = &platform.MountState{}
	}

	s.archiveMounts[repoId][archiveId].IsMounted = state.IsMounted
	s.archiveMounts[repoId][archiveId].MountPath = state.MountPath

	if state.IsMounted {
		s.setRepoState(repoId, RepoStatusMounted)
		defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(repoId))
	} else {
		hasOtherMounts := false
		for _, aState := range s.archiveMounts[repoId] {
			if aState.IsMounted {
				hasOtherMounts = true
				break
			}
		}
		if !hasOtherMounts {
			s.setRepoState(repoId, RepoStatusIdle)
			defer s.eventEmitter.EmitEvent(ctx, types.EventRepoStateChangedString(repoId))
		}
	}
}

func (s *State) SetArchiveMount(ctx context.Context, repoId int, archiveId int, state *platform.MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setArchiveMount(ctx, repoId, archiveId, state)
}

func (s *State) SetArchiveMounts(ctx context.Context, repoId int, states map[int]*platform.MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for archiveId, state := range states {
		s.setArchiveMount(ctx, repoId, archiveId, state)
	}
}

func (s *State) getRepoMount(id int) platform.MountState {
	if state, ok := s.repoMounts[id]; ok {
		return *state
	}
	return platform.MountState{}
}

func (s *State) GetRepoMount(id int) platform.MountState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getRepoMount(id)
}

func (s *State) getArchiveMounts(repoId int) (states map[int]platform.MountState) {
	states = make(map[int]platform.MountState)
	for rId, state := range s.archiveMounts {
		if rId == repoId {
			for aId, aState := range state {
				states[aId] = *aState
			}
		}
	}
	return states
}

func (s *State) GetArchiveMounts(repoId int) (states map[int]platform.MountState) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getArchiveMounts(repoId)
}

/***********************************/
/********** Backup Button **********/
/***********************************/

// GetBackupButtonStatus returns the status of the backup with the given ID.
func (s *State) GetBackupButtonStatus(id types.BackupId) BackupButtonStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If the repository is locked, we can't do anything
	if rs, ok := s.repoStates[id.RepositoryId]; ok {
		if rs.Status == RepoStatusError {
			return BackupButtonStatusLocked
		}
		if rs.Status == RepoStatusMounted {
			return BackupButtonStatusUnmount
		}
	}

	bs := s.getBackupState(id)
	switch bs.Status {
	case BackupStatusWaiting:
		return BackupButtonStatusWaiting
	case BackupStatusRunning:
		return BackupButtonStatusAbort
	case BackupStatusIdle, BackupStatusCompleted, BackupStatusCancelled, BackupStatusFailed:
		if rs, ok := s.repoStates[id.RepositoryId]; ok {
			if rs.Status != RepoStatusIdle {
				// If the repository is busy from another backup profile, we can't do anything
				return BackupButtonStatusBusy
			}
		}

		// If the repository or any of it's archives is mounted, the user has to unmount it before doing anything
		if repoMount := s.getRepoMount(id.RepositoryId); repoMount.IsMounted {
			return BackupButtonStatusUnmount
		}
		if archiveMounts := s.getArchiveMounts(id.RepositoryId); len(archiveMounts) > 0 {
			for _, state := range archiveMounts {
				if state.IsMounted {
					return BackupButtonStatusUnmount
				}
			}
		}

		return BackupButtonStatusRunBackup
	default:
		// If we are here, we probably missed a case or introduced a horrible bug
		return BackupButtonStatusRunBackup
	}
}

// GetCombinedBackupButtonStatus returns the status of all backups in the given list of backup IDs combined.
func (s *State) GetCombinedBackupButtonStatus(bIds []types.BackupId) BackupButtonStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, bId := range bIds {
		if rs, ok := s.repoStates[bId.RepositoryId]; ok {
			if rs.Status == RepoStatusError {
				// If any repository is locked, we can't do anything
				return BackupButtonStatusLocked
			}
			if rs.Status == RepoStatusMounted {
				// If any repository is mounted, we can't do anything
				return BackupButtonStatusUnmount
			}

			// We have to check all non-idle repositories
			if rs.Status != RepoStatusIdle {
				bs := s.getBackupState(bId).Status

				// If any of the backups is waiting or running, we show the waiting or abort button
				if bs == BackupStatusWaiting {
					return BackupButtonStatusWaiting
				}

				if bs == BackupStatusRunning {
					return BackupButtonStatusAbort
				}

				// If we are here it means the repository is busy from another backup profile
				return BackupButtonStatusBusy
			}
		}
	}

	for _, bId := range bIds {
		// If any of the repositories or any of it's archives is mounted, the user has to unmount it before doing anything
		if repoMount := s.getRepoMount(bId.RepositoryId); repoMount.IsMounted {
			return BackupButtonStatusUnmount
		}
		if archiveMounts := s.getArchiveMounts(bId.RepositoryId); len(archiveMounts) > 0 {
			for _, state := range archiveMounts {
				if state.IsMounted {
					return BackupButtonStatusUnmount
				}
			}
		}
	}

	// Being here means all backups are idle, completed, cancelled or failed
	// and all repositories are idle
	return BackupButtonStatusRunBackup
}

/***********************************/
/********** Auth State *************/
/***********************************/

func (s *State) SetAuthenticated(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.authState.IsAuthenticated {
		defer s.eventEmitter.EmitEvent(ctx, types.EventAuthStateChanged.String())
		s.authState.IsAuthenticated = true
	}
}

func (s *State) SetNotAuthenticated(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.authState.IsAuthenticated {
		defer s.eventEmitter.EmitEvent(ctx, types.EventAuthStateChanged.String())
		s.authState.IsAuthenticated = false
	}
}

func (s *State) GetAuthState() AuthState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *s.authState
}

/***********************************/
/******* Checkout & Subscription ***/
/***********************************/

// SetCheckoutSession stores the current checkout session and emits events
func (s *State) SetCheckoutSession(ctx context.Context, session *arcov1.CreateCheckoutSessionResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventCheckoutStateChangedString())
	s.checkoutSession = session
}

// GetCheckoutSession returns the current checkout session
func (s *State) GetCheckoutSession() *arcov1.CreateCheckoutSessionResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.checkoutSession
}

// ClearCheckoutSession clears the current checkout session and emits events
func (s *State) ClearCheckoutSession(ctx context.Context, result *CheckoutResult, emitSubscriptionEvent bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventCheckoutStateChangedString())
	if emitSubscriptionEvent {
		defer s.eventEmitter.EmitEvent(ctx, types.EventSubscriptionAddedString())
	}
	s.checkoutSession = nil
	s.checkoutResult = result
}

// GetCheckoutResult returns the current checkout result
func (s *State) GetCheckoutResult() *CheckoutResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.checkoutResult
}

// ClearCheckoutResult clears the current checkout result
func (s *State) ClearCheckoutResult() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkoutResult = nil
}

// EmitSubscriptionCancelled emits a subscription cancelled event
func (s *State) EmitSubscriptionCancelled(ctx context.Context) {
	s.eventEmitter.EmitEvent(ctx, types.EventSubscriptionCancelledString())
}
