package state

import (
	"arco/backend/app/types"
	"arco/backend/borg"
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
	"sync"
)

type State struct {
	log           *zap.SugaredLogger
	mu            sync.Mutex
	notifications []types.Notification
	startupError  error

	repoStates   map[int]*RepoState
	backupStates map[types.BackupId]*BackupState

	runningPruneJobs       map[types.BackupId]*PruneJob
	runningDryRunPruneJobs map[types.BackupId]*PruneJob
	runningDeleteJobs      map[types.BackupId]*cancelCtx

	repoMounts    map[int]*types.MountState         // map of repository ID to mount state
	archiveMounts map[int]map[int]*types.MountState // maps of [repository ID][archive ID] to mount state
}

type RepoLock struct {
	IsLocked bool
	*sync.Mutex
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

type PruneJob struct {
	*cancelCtx
	result PruneJobResult
}

type RepoStatus string

const (
	RepoStatusIdle                RepoStatus = "idle"
	RepoStatusBackingUp           RepoStatus = "backingUp"
	RepoStatusPruning             RepoStatus = "pruning"
	RepoStatusDeleting            RepoStatus = "deleting"
	RepoStatusMounted             RepoStatus = "mounted"
	RepoStatusPerformingOperation RepoStatus = "performingOperation"
	RepoStatusLocked              RepoStatus = "locked"
)

var AvailableRepoStatuses = []RepoStatus{
	RepoStatusIdle,
	RepoStatusBackingUp,
	RepoStatusPruning,
	RepoStatusDeleting,
	RepoStatusMounted,
	RepoStatusPerformingOperation,
	RepoStatusLocked,
}

func (rs RepoStatus) String() string {
	return string(rs)
}

type RepoState struct {
	mutex  *sync.Mutex
	Status RepoStatus `json:"status"`
}

func newRepoState() *RepoState {
	return &RepoState{
		mutex:  &sync.Mutex{},
		Status: RepoStatusIdle,
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
	Status   BackupStatus         `json:"status"`
	Progress *borg.BackupProgress `json:"progress,omitempty"`
	Error    string               `json:"error,omitempty"`
}

func newBackupState() *BackupState {
	return &BackupState{
		cancelCtx: nil,
		Status:    BackupStatusIdle,
		Progress:  nil,
		Error:     "",
	}
}

type BackupResult string

const (
	BackupResultSuccess   BackupResult = "success"
	BackupResultCancelled BackupResult = "cancelled"
	BackupResultError     BackupResult = "error"
)

func (b BackupResult) String() string {
	return string(b)
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

func NewState(log *zap.SugaredLogger) *State {
	return &State{
		log:           log,
		mu:            sync.Mutex{},
		notifications: []types.Notification{},
		startupError:  nil,

		repoStates:   make(map[int]*RepoState),
		backupStates: map[types.BackupId]*BackupState{},

		runningPruneJobs:       make(map[types.BackupId]*PruneJob),
		runningDryRunPruneJobs: make(map[types.BackupId]*PruneJob),
		runningDeleteJobs:      make(map[types.BackupId]*cancelCtx),

		repoMounts:    make(map[int]*types.MountState),
		archiveMounts: make(map[int]map[int]*types.MountState),
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

func (s *State) SetStartupError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startupError = err
}

func (s *State) GetStartupError() error {
	return s.startupError
}

/***********************************/
/********** Repo States ************/
/***********************************/

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
	defer runtime.EventsEmit(ctx, types.EventRepoStateChanged.String()+fmt.Sprintf(":%d", repoId))

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
	if rs, ok := s.repoStates[repoId]; ok {
		return *rs
	}
	return *newRepoState()
}

/***********************************/
/********** Backup States **********/
/***********************************/

func (s *State) CanRunBackup(id types.BackupId) (canRun bool, reason string) {
	if s.startupError != nil {
		return false, "Startup error"
	}
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
	defer runtime.EventsEmit(ctx, types.EventBackupStateChangedString(bId))
	defer runtime.EventsEmit(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusWaiting)
}

func (s *State) SetBackupRunning(ctx context.Context, bId types.BackupId) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer runtime.EventsEmit(ctx, types.EventBackupStateChangedString(bId))
	defer runtime.EventsEmit(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	currentState, ok := s.backupStates[bId]
	if ok {
		if currentState.Status == BackupStatusRunning {
			// If the state is already running, we don't do anything
			return currentState.ctx
		}
	}

	s.changeBackupState(bId, BackupStatusRunning)
	s.backupStates[bId].cancelCtx = newCancelCtx(ctx)
	s.backupStates[bId].Progress = &borg.BackupProgress{}
	s.backupStates[bId].Error = ""

	s.setRepoState(bId.RepositoryId, RepoStatusBackingUp)

	return s.backupStates[bId].ctx
}

func (s *State) SetBackupCompleted(ctx context.Context, bId types.BackupId, setRepoStateIdle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer runtime.EventsEmit(ctx, types.EventBackupStateChangedString(bId))
	defer runtime.EventsEmit(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusCompleted)
	if setRepoStateIdle {
		s.setRepoState(bId.RepositoryId, RepoStatusIdle)
	}
}

func (s *State) SetBackupCancelled(ctx context.Context, bId types.BackupId, setRepoStateIdle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer runtime.EventsEmit(ctx, types.EventBackupStateChangedString(bId))
	defer runtime.EventsEmit(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusCancelled)
	if setRepoStateIdle {
		s.setRepoState(bId.RepositoryId, RepoStatusIdle)
	}
}

func (s *State) SetBackupError(ctx context.Context, bId types.BackupId, err error, setRepoStateIdle bool, setRepoLocked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer runtime.EventsEmit(ctx, types.EventBackupStateChangedString(bId))
	defer runtime.EventsEmit(ctx, types.EventRepoStateChangedString(bId.RepositoryId))

	s.changeBackupState(bId, BackupStatusFailed)
	s.backupStates[bId].Error = err.Error()
	if setRepoLocked {
		s.setRepoState(bId.RepositoryId, RepoStatusLocked)
	} else if setRepoStateIdle {
		s.setRepoState(bId.RepositoryId, RepoStatusIdle)
	}
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

func (s *State) UpdateBackupProgress(ctx context.Context, bId types.BackupId, progress borg.BackupProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer runtime.EventsEmit(ctx, types.EventBackupStateChangedString(bId))

	if currentState, ok := s.backupStates[bId]; ok {
		if currentState.Status == BackupStatusRunning {
			currentState.Progress = &progress
		}
	}
}

func (s *State) GetBackupState(id types.BackupId) BackupState {
	if bs, ok := s.backupStates[id]; ok {
		return *bs
	}
	return *newBackupState()
}

func (s *State) GetCombinedBackupProgress(ids []types.BackupId) *borg.BackupProgress {
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
	return &borg.BackupProgress{
		TotalFiles:     totalFiles,
		ProcessedFiles: processedFiles,
	}
}

/***********************************/
/********** Prune Jobs ************/
/***********************************/

func (s *State) CanRunPruneJob(id types.BackupId) (canRun bool, reason string) {
	if s.startupError != nil {
		return false, "Startup error"
	}
	if _, ok := s.runningPruneJobs[id]; ok {
		return false, "Prune job is already running"
	}
	if rs, ok := s.repoStates[id.RepositoryId]; ok {
		if rs.Status != RepoStatusIdle {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

func (s *State) AddRunningPruneJob(ctx context.Context, id types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningPruneJobs[id] = &PruneJob{
		cancelCtx: newCancelCtx(ctx),
		result:    PruneJobResult{},
	}
}

func (s *State) RemoveRunningPruneJob(id types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningPruneJobs[id]; ok {
		s.log.Debugf("Cancelling context of prune job %v", id)
		ctx.cancel()
	}

	delete(s.runningPruneJobs, id)
}

func (s *State) SetPruneResult(id types.BackupId, result PruneJobResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bj := s.runningPruneJobs[id]; bj != nil {
		bj.result = result
	}
}

/***********************************/
/********** Dry Run Prune Jobs ****/
/***********************************/

func (s *State) AddRunningDryRunPruneJob(ctx context.Context, id types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningDryRunPruneJobs[id] = &PruneJob{
		cancelCtx: newCancelCtx(ctx),
		result:    PruneJobResult{},
	}
}

func (s *State) RemoveRunningDryRunPruneJob(id types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningDryRunPruneJobs[id]; ok {
		s.log.Debugf("Cancelling context of dry-run prune job %v", id)
		ctx.cancel()
	}

	delete(s.runningDryRunPruneJobs, id)
}

func (s *State) SetDryRunPruneResult(id types.BackupId, result PruneJobResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bj := s.runningDryRunPruneJobs[id]; bj != nil {
		bj.result = result
	}
}

/***********************************/
/********** Delete Jobs ************/
/***********************************/

func (s *State) CanRunDeleteJob(repoId int) (canRun bool, reason string) {
	if s.startupError != nil {
		return false, "Startup error"
	}
	if rs, ok := s.repoStates[repoId]; ok {
		if rs.Status != RepoStatusIdle {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

func (s *State) AddRunningDeleteJob(ctx context.Context, id types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningDeleteJobs[id] = newCancelCtx(ctx)
}

func (s *State) RemoveRunningDeleteJob(id types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningDeleteJobs[id]; ok {
		s.log.Debugf("Cancelling context of delete job %v", id)
		ctx.cancel()
	}

	delete(s.runningDeleteJobs, id)
}

/***********************************/
/********** Notifications **********/
/***********************************/

func (s *State) AddNotification(ctx context.Context, msg string, level types.NotificationLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer runtime.EventsEmit(ctx, types.EventNotificationAvailable.String())

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
	if s.startupError != nil {
		return false, "Startup error"
	}
	if rs, ok := s.repoStates[id]; ok {
		// We can only mount a repository if it's idle or mounting
		if rs.Status != RepoStatusIdle && rs.Status != RepoStatusMounted {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

func (s *State) SetRepoMount(ctx context.Context, repoId int, state *types.MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer runtime.EventsEmit(ctx, types.EventRepoStateChangedString(repoId))

	if _, ok := s.repoMounts[repoId]; !ok {
		s.repoMounts[repoId] = &types.MountState{}
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

func (s *State) setArchiveMount(ctx context.Context, repoId int, archiveId int, state *types.MountState) {
	if _, ok := s.archiveMounts[repoId]; !ok {
		s.archiveMounts[repoId] = make(map[int]*types.MountState)
	}
	if _, ok := s.archiveMounts[repoId][archiveId]; !ok {
		s.archiveMounts[repoId][archiveId] = &types.MountState{}
	}

	s.archiveMounts[repoId][archiveId].IsMounted = state.IsMounted
	s.archiveMounts[repoId][archiveId].MountPath = state.MountPath

	if state.IsMounted {
		s.setRepoState(repoId, RepoStatusMounted)
		defer runtime.EventsEmit(ctx, types.EventRepoStateChangedString(repoId))
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
			defer runtime.EventsEmit(ctx, types.EventRepoStateChangedString(repoId))
		}
	}
}

func (s *State) SetArchiveMount(ctx context.Context, repoId int, archiveId int, state *types.MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setArchiveMount(ctx, repoId, archiveId, state)
}

func (s *State) SetArchiveMounts(ctx context.Context, repoId int, states map[int]*types.MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for archiveId, state := range states {
		s.setArchiveMount(ctx, repoId, archiveId, state)
	}
}

func (s *State) GetRepoMount(id int) types.MountState {
	if state, ok := s.repoMounts[id]; ok {
		return *state
	}
	return types.MountState{}
}

func (s *State) GetArchiveMounts(repoId int) (states map[int]types.MountState) {
	states = make(map[int]types.MountState)
	for rId, state := range s.archiveMounts {
		if rId == repoId {
			for aId, aState := range state {
				states[aId] = *aState
			}
		}
	}
	return states
}

/***********************************/
/********** Backup Button **********/
/***********************************/

// GetBackupButtonStatus returns the status of the backup with the given ID.
func (s *State) GetBackupButtonStatus(id types.BackupId) BackupButtonStatus {
	// If the repository is locked, we can't do anything
	if rs, ok := s.repoStates[id.RepositoryId]; ok {
		if rs.Status == RepoStatusLocked {
			return BackupButtonStatusLocked
		}
		if rs.Status == RepoStatusMounted {
			return BackupButtonStatusUnmount
		}
	}

	bs := s.GetBackupState(id)
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
		if repoMount := s.GetRepoMount(id.RepositoryId); repoMount.IsMounted {
			return BackupButtonStatusUnmount
		}
		if archiveMounts := s.GetArchiveMounts(id.RepositoryId); len(archiveMounts) > 0 {
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
	for _, bId := range bIds {
		if rs, ok := s.repoStates[bId.RepositoryId]; ok {
			if rs.Status == RepoStatusLocked {
				// If any repository is locked, we can't do anything
				return BackupButtonStatusLocked
			}
			if rs.Status == RepoStatusMounted {
				// If any repository is mounted, we can't do anything
				return BackupButtonStatusUnmount
			}

			// We have to check all non-idle repositories
			if rs.Status != RepoStatusIdle {
				bs := s.GetBackupState(bId).Status

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
		if repoMount := s.GetRepoMount(bId.RepositoryId); repoMount.IsMounted {
			return BackupButtonStatusUnmount
		}
		if archiveMounts := s.GetArchiveMounts(bId.RepositoryId); len(archiveMounts) > 0 {
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
