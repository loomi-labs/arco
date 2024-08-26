package state

import (
	"arco/backend/app/types"
	"arco/backend/borg"
	"context"
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

	repoMounts    map[int]*MountState         // map of repository ID to mount state
	archiveMounts map[int]map[int]*MountState // maps of [repository ID][archive ID] to mount state
}

type RepoLock struct {
	IsLocked bool
	*sync.Mutex
}

type cancelCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type BackupJob struct {
	*cancelCtx
	progress borg.BackupProgress
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

type MountState struct {
	IsMounted bool   `json:"is_mounted"`
	MountPath string `json:"mount_path"`
}

type RepoStatus string

const (
	RepoStatusIdle                RepoStatus = "idle"
	RepoStatusBackingUp           RepoStatus = "backing_up"
	RepoStatusPruning             RepoStatus = "pruning"
	RepoStatusDeleting            RepoStatus = "deleting"
	RepoStatusPerformingOperation RepoStatus = "performing_operation"
	RepoStatusLocked              RepoStatus = "locked"
)

var AvailableRepoStatuses = []RepoStatus{
	RepoStatusIdle,
	RepoStatusBackingUp,
	RepoStatusPruning,
	RepoStatusDeleting,
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

		repoMounts:    make(map[int]*MountState),
		archiveMounts: make(map[int]map[int]*MountState),
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

func (s *State) SetRepoStatus(repoId int, state RepoStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setRepoState(repoId, state)
}

func (s *State) setRepoState(repoId int, state RepoStatus) {
	if rs, ok := s.repoStates[repoId]; ok {
		if rs.Status != RepoStatusIdle && state == RepoStatusIdle {
			// If we are here it means:
			// - the current state is not idle
			// - the new state is idle
			// Therefore we unlock the repository
			rs.mutex.Unlock()
		}

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

func (s *State) SetBackupWaiting(bId types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	s.changeBackupState(bId, BackupStatusRunning)
	s.backupStates[bId].cancelCtx = newCancelCtx(ctx)
	s.backupStates[bId].Progress = &borg.BackupProgress{}
	s.backupStates[bId].Error = ""

	s.setRepoState(bId.RepositoryId, RepoStatusBackingUp)

	return s.backupStates[bId].ctx
}

func (s *State) SetBackupCompleted(bId types.BackupId, setRepoStateIdle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.changeBackupState(bId, BackupStatusCompleted)
	if setRepoStateIdle {
		s.setRepoState(bId.RepositoryId, RepoStatusIdle)
	}
}

func (s *State) SetBackupCancelled(bId types.BackupId, setRepoStateIdle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.changeBackupState(bId, BackupStatusCancelled)
	if setRepoStateIdle {
		s.setRepoState(bId.RepositoryId, RepoStatusIdle)
	}
}

func (s *State) SetBackupError(bId types.BackupId, err error, setRepoStateIdle bool, setRepoLocked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *State) UpdateBackupProgress(id types.BackupId, progress borg.BackupProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if currentState, ok := s.backupStates[id]; ok {
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

func (s *State) AddNotification(msg string, level types.NotificationLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *State) SetRepoMount(id int, state *MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repoMounts[id]; !ok {
		s.repoMounts[id] = &MountState{}
	}

	s.repoMounts[id].IsMounted = state.IsMounted
	s.repoMounts[id].MountPath = state.MountPath
}

func (s *State) setArchiveMount(repoId int, archiveId int, state *MountState) {
	if _, ok := s.archiveMounts[repoId]; !ok {
		s.archiveMounts[repoId] = make(map[int]*MountState)
	}
	if _, ok := s.archiveMounts[repoId][archiveId]; !ok {
		s.archiveMounts[repoId][archiveId] = &MountState{}
	}

	s.archiveMounts[repoId][archiveId].IsMounted = state.IsMounted
	s.archiveMounts[repoId][archiveId].MountPath = state.MountPath
}

func (s *State) SetArchiveMount(repoId int, archiveId int, state *MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setArchiveMount(repoId, archiveId, state)
}

func (s *State) SetArchiveMounts(repoId int, states map[int]*MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for archiveId, state := range states {
		s.setArchiveMount(repoId, archiveId, state)
	}
}

func (s *State) GetRepoMount(id int) MountState {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state, ok := s.repoMounts[id]; ok {
		return *state
	}
	return MountState{}
}

func (s *State) GetArchiveMounts(repoId int) (states map[int]MountState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	states = make(map[int]MountState)
	for rId, state := range s.archiveMounts {
		if rId == repoId {
			for aId, aState := range state {
				states[aId] = *aState
			}
		}
	}
	return states
}
