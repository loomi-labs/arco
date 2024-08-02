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

	repoLocks map[int]*RepoLock

	runningBackupJobs      map[types.BackupId]*BackupJob
	runningPruneJobs       map[types.BackupId]*PruneJob
	runningDryRunPruneJobs map[types.BackupId]*PruneJob
	runningDeleteJobs      map[types.BackupId]*CancelCtx

	repoMounts    map[int]*MountState         // map of repository ID to mount state
	archiveMounts map[int]map[int]*MountState // maps of [repository ID][archive ID] to mount state
}

type RepoLock struct {
	IsLocked bool
	*sync.Mutex
}

type CancelCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type BackupJob struct {
	*CancelCtx
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
	*CancelCtx
	result PruneJobResult
}

type MountState struct {
	IsMounted bool   `json:"is_mounted"`
	MountPath string `json:"mount_path"`
}

func NewCancelCtx(ctx context.Context) *CancelCtx {
	nCtx, cancel := context.WithCancel(ctx)
	return &CancelCtx{
		ctx:    nCtx,
		cancel: cancel,
	}
}

func NewState(log *zap.SugaredLogger) *State {
	return &State{
		log:           log,
		mu:            sync.Mutex{},
		notifications: []types.Notification{},
		startupError:  nil,

		repoLocks:              make(map[int]*RepoLock),
		runningBackupJobs:      make(map[types.BackupId]*BackupJob),
		runningPruneJobs:       make(map[types.BackupId]*PruneJob),
		runningDryRunPruneJobs: make(map[types.BackupId]*PruneJob),
		runningDeleteJobs:      make(map[types.BackupId]*CancelCtx),

		repoMounts:    make(map[int]*MountState),
		archiveMounts: make(map[int]map[int]*MountState),
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
/********** Repo Locks *************/
/***********************************/

// GetRepoLock returns the lock for the given repository ID.
// The lock has to be acquired before performing any operations on the repository.
//
// Usage:
// lock := state.GetRepoLock(repoId)
// lock.Lock() // Wait to acquire the lock
// state.SetRepoLocked(repoId)	// Set the repo as locked
// defer state.UnlockRepo(repoId)	// Unlock the repo when done
func (s *State) GetRepoLock(repoId int) *RepoLock {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repoLocks[repoId]; !ok {
		s.repoLocks[repoId] = &RepoLock{
			IsLocked: false,
			Mutex:    &sync.Mutex{},
		}
	}
	return s.repoLocks[repoId]
}

func (s *State) SetRepoLocked(repoId int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if lock, ok := s.repoLocks[repoId]; ok {
		lock.IsLocked = true
	}
}

func (s *State) UnlockRepo(repoId int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if lock, ok := s.repoLocks[repoId]; ok {
		lock.IsLocked = false
		lock.Unlock()
	}
}

/***********************************/
/********** Backup Jobs ************/
/***********************************/

func (s *State) CanRunBackup(id types.BackupId) (canRun bool, reason string) {
	if s.startupError != nil {
		return false, "Startup error"
	}
	if _, ok := s.runningBackupJobs[id]; ok {
		return false, "Backup is already running"
	}
	if lock, ok := s.repoLocks[id.RepositoryId]; ok {
		if lock.IsLocked {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

func (s *State) AddRunningBackup(ctx context.Context, id types.BackupId) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancelCtx := NewCancelCtx(ctx)
	s.runningBackupJobs[id] = &BackupJob{
		CancelCtx: cancelCtx,
		progress:  borg.BackupProgress{},
	}
	return cancelCtx.ctx
}

func (s *State) RemoveRunningBackup(id types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningBackupJobs[id]; ok {
		s.log.Debugf("Cancelling context of backup job %v", id)
		ctx.cancel()
	}

	delete(s.runningBackupJobs, id)
}

func (s *State) UpdateBackupProgress(id types.BackupId, progress borg.BackupProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bj := s.runningBackupJobs[id]; bj != nil {
		bj.progress = progress
	}
}

func (s *State) GetBackupProgress(id types.BackupId) (progress borg.BackupProgress, found bool) {
	if bj := s.runningBackupJobs[id]; bj != nil {
		return bj.progress, true
	}
	return borg.BackupProgress{}, false
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
	if lock, ok := s.repoLocks[id.RepositoryId]; ok {
		if lock.IsLocked {
			return false, "Repository is busy"
		}
	}
	return true, ""
}

func (s *State) AddRunningPruneJob(ctx context.Context, id types.BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningPruneJobs[id] = &PruneJob{
		CancelCtx: NewCancelCtx(ctx),
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
		CancelCtx: NewCancelCtx(ctx),
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

	s.runningDeleteJobs[id] = NewCancelCtx(ctx)
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
