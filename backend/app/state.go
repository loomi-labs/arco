package app

import (
	"arco/backend/borg"
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

	runningBackupJobs map[BackupId]*BackupJob
	runningPruneJobs  map[BackupId]*CancelCtx
	runningDeleteJobs map[BackupId]*CancelCtx
}

type CancelCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type BackupJob struct {
	*CancelCtx
	progress borg.BackupProgress
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
		notifications: []Notification{},
		StartupErr:    nil,

		repoLocks:         map[int]*sync.Mutex{},
		runningBackupJobs: make(map[BackupId]*BackupJob),
		runningPruneJobs:  make(map[BackupId]*CancelCtx),
		runningDeleteJobs: make(map[BackupId]*CancelCtx),
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

func (s *State) CanRunBackup(id BackupId) (canRun bool, reason string) {
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

func (s *State) AddRunningBackup(ctx context.Context, id BackupId) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancelCtx := NewCancelCtx(ctx)
	s.runningBackupJobs[id] = &BackupJob{
		CancelCtx: cancelCtx,
		progress:  borg.BackupProgress{},
	}
	return cancelCtx.ctx
}

func (s *State) RemoveRunningBackup(id BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningBackupJobs[id]; ok {
		s.log.Debugf("Cancelling context of backup job %v", id)
		ctx.cancel()
	}

	delete(s.runningBackupJobs, id)
}

func (s *State) CanRunPruneJob(id BackupId) (canRun bool, reason string) {
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

func (s *State) AddRunningPruneJob(ctx context.Context, id BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningPruneJobs[id] = NewCancelCtx(ctx)
}

func (s *State) RemoveRunningPruneJob(id BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.runningPruneJobs[id]; ok {
		s.log.Debugf("Cancelling context of prune job %v", id)
		ctx.cancel()
	}

	delete(s.runningPruneJobs, id)
}

func (s *State) AddRunningDeleteJob(ctx context.Context, id BackupId) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningDeleteJobs[id] = NewCancelCtx(ctx)
}

func (s *State) RemoveRunningDeleteJob(id BackupId) {
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

func (s *State) UpdateBackupProgress(id BackupId, progress borg.BackupProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bj := s.runningBackupJobs[id]; bj != nil {
		bj.progress = progress
	}
}

func (s *State) GetBackupProgress(id BackupId) (progress borg.BackupProgress, found bool) {
	if bj := s.runningBackupJobs[id]; bj != nil {
		return bj.progress, true
	}
	return borg.BackupProgress{}, false
}
