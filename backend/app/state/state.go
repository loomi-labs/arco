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

type RepoStateVal string

const (
	RepoStateIdle                RepoStateVal = "idle"
	RepoStateBackingUp           RepoStateVal = "backing_up"
	RepoStatePruning             RepoStateVal = "pruning"
	RepoStateDeleting            RepoStateVal = "deleting"
	RepoStatePerformingOperation RepoStateVal = "performing_operation"
	RepoStateLocked              RepoStateVal = "locked"
)

type RepoState struct {
	mutex *sync.Mutex
	State RepoStateVal `json:"state"`
}

func newRepoState() *RepoState {
	return &RepoState{
		mutex: &sync.Mutex{},
		State: RepoStateIdle,
	}
}

type BackupStateVal string

const (
	BackupStateIdle      BackupStateVal = "idle"
	BackupStateWaiting   BackupStateVal = "waiting"
	BackupStateRunning   BackupStateVal = "running"
	BackupStateCompleted BackupStateVal = "completed"
	BackupStateCancelled BackupStateVal = "cancelled"
	BackupStateError     BackupStateVal = "error"
)

var AllBackupStates = []BackupStateVal{
	BackupStateIdle,
	BackupStateWaiting,
	BackupStateRunning,
	BackupStateCompleted,
	BackupStateCancelled,
	BackupStateError,
}

func (bs BackupStateVal) String() string {
	return string(bs)
}

type BackupState struct {
	*cancelCtx
	State    BackupStateVal       `json:"state"`
	Progress *borg.BackupProgress `json:"progress,omitempty"`
	Error    error                `json:"error,omitempty"`
}

func newBackupState() *BackupState {
	return &BackupState{
		cancelCtx: nil,
		State:     BackupStateIdle,
		Progress:  nil,
		Error:     nil,
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
// state.SetRepoState(repoId, &state.RepoStatePerformingOperation{})	// Set the repo state
// defer b.state.SetRepoState(bId.RepositoryId, &state.RepoStateIdle{}) // Set the repo state back to idle
func (s *State) GetRepoLock(repoId int) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repoStates[repoId]; !ok {
		s.setRepoState(repoId, RepoStateIdle)
	}
	return s.repoStates[repoId].mutex
}

func (s *State) SetRepoState(repoId int, state RepoStateVal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setRepoState(repoId, state)
}

func (s *State) setRepoState(repoId int, state RepoStateVal) {
	if rs, ok := s.repoStates[repoId]; ok {
		if rs.State != RepoStateIdle && state == RepoStateIdle {
			// If we are here it means:
			// - the current state is not idle
			// - the new state is idle
			// Therefore we unlock the repository
			rs.mutex.Unlock()
		}

		s.repoStates[repoId].State = state
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
		if bs.State == BackupStateRunning {
			return false, "Backup is already running"
		}
	}
	if rs, ok := s.repoStates[id.RepositoryId]; ok {
		if rs.State != RepoStateIdle {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

// TODO: remove this?
func (s *State) GetBackupProgress(id types.BackupId) (progress borg.BackupProgress, found bool) {
	if currentState, ok := s.backupStates[id]; ok {
		if currentState.State == BackupStateRunning {
			if currentState.Progress != nil {
				return *currentState.Progress, true
			}
			return borg.BackupProgress{}, true
		}
	}
	return borg.BackupProgress{}, false
}

func (s *State) SetBackupWaiting(bId types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.changeBackupState(bId, BackupStateWaiting)
}

func (s *State) SetBackupRunning(ctx context.Context, bId types.BackupId) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentState, ok := s.backupStates[bId]
	if ok {
		if currentState.State == BackupStateRunning {
			// If the state is already running, we don't do anything
			return currentState.ctx
		}
	}

	s.changeBackupState(bId, BackupStateRunning)
	s.backupStates[bId].cancelCtx = newCancelCtx(ctx)
	s.backupStates[bId].Progress = &borg.BackupProgress{}
	s.backupStates[bId].Error = nil

	s.setRepoState(bId.RepositoryId, RepoStateBackingUp)

	return s.backupStates[bId].ctx
}

func (s *State) SetBackupCompleted(bId types.BackupId, setRepoStateIdle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.changeBackupState(bId, BackupStateCompleted)
	if setRepoStateIdle {
		s.setRepoState(bId.RepositoryId, RepoStateIdle)
	}
}

func (s *State) SetBackupCancelled(bId types.BackupId, setRepoStateIdle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.changeBackupState(bId, BackupStateCancelled)
	if setRepoStateIdle {
		s.setRepoState(bId.RepositoryId, RepoStateIdle)
	}
}

func (s *State) SetBackupError(bId types.BackupId, err error, setRepoStateIdle bool, setRepoLocked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.changeBackupState(bId, BackupStateError)
	s.backupStates[bId].Error = err
	if setRepoLocked {
		s.setRepoState(bId.RepositoryId, RepoStateLocked)
	} else if setRepoStateIdle {
		s.setRepoState(bId.RepositoryId, RepoStateIdle)
	}
}

func (s *State) changeBackupState(bId types.BackupId, newState BackupStateVal) {
	currentState, ok := s.backupStates[bId]
	if ok {
		if currentState.State == BackupStateRunning && newState != BackupStateRunning {
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

	s.backupStates[bId].State = newState
}

func (s *State) UpdateBackupProgress(id types.BackupId, progress borg.BackupProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if currentState, ok := s.backupStates[id]; ok {
		if currentState.State == BackupStateRunning {
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
		if rs.State != RepoStateIdle {
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
