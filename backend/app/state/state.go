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

	repoStates   map[int]RepoState
	backupStates map[types.BackupId]BackupState // TODO: make this a pointer?

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

type RepoState interface {
	getMutex() *sync.Mutex
	setMutex(*sync.Mutex)
}

type RepoStateIdle struct {
	*sync.Mutex
}

type RepoStateBackingUp struct {
	*sync.Mutex
}

type RepoStatePruning struct {
	*sync.Mutex
}

type RepoStateDeleting struct {
	*sync.Mutex
}

type RepoStatePerformingOperation struct {
	*sync.Mutex
}

type RepoStateLocked struct {
	*sync.Mutex
}

func (rs *RepoStateIdle) getMutex() *sync.Mutex      { return rs.Mutex }
func (rs *RepoStateIdle) setMutex(m *sync.Mutex)     { rs.Mutex = m }
func (rs *RepoStateBackingUp) getMutex() *sync.Mutex { return rs.Mutex }
func (rs *RepoStateBackingUp) setMutex(m *sync.Mutex) {
	rs.Mutex = m
}
func (rs *RepoStatePruning) getMutex() *sync.Mutex { return rs.Mutex }
func (rs *RepoStatePruning) setMutex(m *sync.Mutex) {
	rs.Mutex = m
}
func (rs *RepoStateDeleting) getMutex() *sync.Mutex { return rs.Mutex }
func (rs *RepoStateDeleting) setMutex(m *sync.Mutex) {
	rs.Mutex = m
}
func (rs *RepoStatePerformingOperation) getMutex() *sync.Mutex { return rs.Mutex }
func (rs *RepoStatePerformingOperation) setMutex(m *sync.Mutex) {
	rs.Mutex = m
}
func (rs *RepoStateLocked) getMutex() *sync.Mutex  { return rs.Mutex }
func (rs *RepoStateLocked) setMutex(m *sync.Mutex) { rs.Mutex = m }

type BackupState interface {
	bs() // marker method for the compiler... without this we could store any type as BackupState
}

type BackupStateWaiting struct {
}

type backupStateRunning struct {
	*cancelCtx
	Progress borg.BackupProgress
}

type BackupStateCompleted struct {
}

type BackupStateCancelled struct {
}

type BackupStateError struct {
	Error error
}

func (BackupStateWaiting) bs()   {}
func (backupStateRunning) bs()   {}
func (BackupStateCompleted) bs() {}
func (BackupStateCancelled) bs() {}
func (BackupStateError) bs()     {}

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

		repoStates:   make(map[int]RepoState),
		backupStates: map[types.BackupId]BackupState{},

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
	s.mu.Lock()
	defer s.mu.Unlock()

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
		s.repoStates[repoId] = &RepoStateIdle{Mutex: &sync.Mutex{}}
	}
	return s.repoStates[repoId].getMutex()
}

func (s *State) SetRepoState(repoId int, state RepoState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setRepoState(repoId, state)
}

func (s *State) setRepoState(repoId int, state RepoState) {
	if rs, ok := s.repoStates[repoId]; ok {
		if _, ok := rs.(*RepoStateIdle); !ok {
			if _, ok := state.(*RepoStateIdle); ok {
				// If we are here it means:
				// - the current state is not idle
				// - the new state is idle
				// Therefore we unlock the repository
				rs.getMutex().Unlock()
			}
		}

		// Copy the mutex from the old state to the new state
		mutex := rs.getMutex()
		if mutex == nil {
			mutex = &sync.Mutex{}
		}
		state.setMutex(mutex)
		s.repoStates[repoId] = state
	} else {
		// If the repository state doesn't exist, we create it
		if state.getMutex() == nil {
			state.setMutex(&sync.Mutex{})
		}
		s.repoStates[repoId] = state
	}
}

func (s *State) GetRepoState(repoId int) RepoState {
	if state, ok := s.repoStates[repoId]; ok {
		return state
	}
	return &RepoStateIdle{}
}

/***********************************/
/********** Backup States **********/
/***********************************/

func (s *State) CanRunBackup(id types.BackupId) (canRun bool, reason string) {
	if s.startupError != nil {
		return false, "Startup error"
	}
	if bs, ok := s.backupStates[id]; ok {
		if _, ok := bs.(*backupStateRunning); ok {
			return false, "Backup is already running"
		}
	}
	if rs, ok := s.repoStates[id.RepositoryId]; ok {
		if _, ok := rs.(*RepoStateIdle); !ok {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

// TODO: revove this?
func (s *State) GetBackupProgress(id types.BackupId) (progress borg.BackupProgress, found bool) {
	if currentState, ok := s.backupStates[id]; ok {
		if currentState, ok := currentState.(*backupStateRunning); ok {
			return currentState.Progress, true
		}
	}
	return borg.BackupProgress{}, false
}

func (s *State) SetBackupState(bId types.BackupId, newState BackupState, repoState RepoState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if currentState, ok := s.backupStates[bId]; ok {
		if _, ok := currentState.(*backupStateRunning); ok {
			// If we are here it means:
			// - the current state is running
			// - the new state is not running (it's either completed, cancelled or errored)
			// Therefore we cancel the context to stop eventual running borg operations
			currentState.(*backupStateRunning).cancel()
		}
	}
	if repoState != nil {
		s.setRepoState(bId.RepositoryId, repoState)
	}

	s.backupStates[bId] = newState
}

func (s *State) SetBackupRunning(ctx context.Context, bId types.BackupId, repoState RepoState) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if currentState, ok := s.backupStates[bId]; ok {
		if currentState, ok := currentState.(*backupStateRunning); ok {
			// If the state is already running, we don't do anything
			return currentState.ctx
		}
	}

	cCtx := newCancelCtx(ctx)
	s.backupStates[bId] = &backupStateRunning{
		cancelCtx: cCtx,
		Progress:  borg.BackupProgress{},
	}

	if repoState != nil {
		s.setRepoState(bId.RepositoryId, repoState)
	}

	return cCtx.ctx
}

//func (s *State) StopRunningBackup(id types.BackupId) {
//	s.mu.Lock()
//	defer s.mu.Unlock()
//
//	if currentState, ok := s.backupStates[id]; ok {
//		if currentState, ok := currentState.(*backupStateRunning); ok {
//			currentState.cancel()
//		}
//	}
//
//	delete(s.backupStates, id)
//}

func (s *State) UpdateBackupProgress(id types.BackupId, progress borg.BackupProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if currentState, ok := s.backupStates[id]; ok {
		if currentState, ok := currentState.(*backupStateRunning); ok {
			currentState.Progress = progress
		}
	}
}

func (s *State) GetBackupState(id types.BackupId) (BackupState, bool) {
	state, found := s.backupStates[id]
	return state, found
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
		if _, ok := rs.(*RepoStateIdle); !ok {
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
