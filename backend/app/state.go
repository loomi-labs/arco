package app

import (
	"context"
	"go.uber.org/zap"
	"sync"
)

type State struct {
	log           *zap.SugaredLogger
	mu            sync.Mutex
	notifications []Notification
	StartupErr    error

	repoLocks map[int]*sync.Mutex

	runningBackupJobs map[BackupIdentifier]*CancelCtx
	runningPruneJobs  map[BackupIdentifier]*CancelCtx
	runningDeleteJobs map[BackupIdentifier]*CancelCtx
}

type CancelCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// TODO: do we need this?
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
		notifications: []Notification{},
		StartupErr:    nil,

		repoLocks:         map[int]*sync.Mutex{},
		runningBackupJobs: make(map[BackupIdentifier]*CancelCtx),
		runningPruneJobs:  make(map[BackupIdentifier]*CancelCtx),
		runningDeleteJobs: make(map[BackupIdentifier]*CancelCtx),
	}
}

func (s *State) GetRepoLock(repoId int) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repoLocks[repoId]; !ok {
		s.repoLocks[repoId] = &sync.Mutex{}
	}
	return s.repoLocks[repoId]
}

func (s *State) DeleteRepoLock(repoId int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.repoLocks, repoId)
}

func (s *State) CanRunBackup(id BackupIdentifier) (canRun bool, reason string) {
	if s.StartupErr != nil {
		return false, "Startup error"
	}
	if _, ok := s.runningBackupJobs[id]; ok {
		return false, "Backup is already running"
	}
	if _, ok := s.repoLocks[id.RepositoryId]; ok {
		return false, "Repository is busy"
	}
	return true, ""
}

func (s *State) AddRunningBackup(ctx context.Context, id BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningBackupJobs[id] = NewCancelCtx(ctx)
}

func (s *State) RemoveRunningBackup(id BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningBackupJobs[id]; ok {
		s.log.Debugf("Cancelling context of backup job %v", id)
		ctx.cancel()
	}

	delete(s.runningBackupJobs, id)
}

func (s *State) CanRunPruneJob(id BackupIdentifier) (canRun bool, reason string) {
	if s.StartupErr != nil {
		return false, "Startup error"
	}
	if _, ok := s.runningPruneJobs[id]; ok {
		return false, "Prune job is already running"
	}
	if _, ok := s.repoLocks[id.RepositoryId]; ok {
		return false, "Repository is busy"
	}
	return true, ""
}

func (s *State) AddRunningPruneJob(ctx context.Context, id BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningPruneJobs[id] = NewCancelCtx(ctx)
}

func (s *State) RemoveRunningPruneJob(id BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningPruneJobs[id]; ok {
		s.log.Debugf("Cancelling context of prune job %v", id)
		ctx.cancel()
	}

	delete(s.runningPruneJobs, id)
}

func (s *State) AddRunningDeleteJob(ctx context.Context, id BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningDeleteJobs[id] = NewCancelCtx(ctx)
}

func (s *State) RemoveRunningDeleteJob(id BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningDeleteJobs[id]; ok {
		s.log.Debugf("Cancelling context of delete job %v", id)
		ctx.cancel()
	}

	delete(s.runningDeleteJobs, id)
}

func (s *State) AddNotification(msg string, level NotificationLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.notifications = append(s.notifications, Notification{
		Message: msg,
		Level:   level,
	})
}

func (s *State) GetAndDeleteNofications() []Notification {
	s.mu.Lock()
	defer s.mu.Unlock()

	notifications := s.notifications
	s.notifications = []Notification{}
	return notifications
}
