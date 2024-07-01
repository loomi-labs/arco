package app

import (
	"arco/backend/types"
	"slices"
	"sync"
	"time"
)

type State struct {
	StartupErr       error
	runningBackups   []types.BackupIdentifier
	runningPruneJobs []types.BackupIdentifier
	occupiedRepos    []int
	notifications    []Notification
	mu               sync.Mutex
	backupTimer      *time.Timer
}

func NewState() *State {
	return &State{
		runningBackups:   []types.BackupIdentifier{},
		runningPruneJobs: []types.BackupIdentifier{},
		occupiedRepos:    []int{},
		StartupErr:       nil,
		notifications:    []Notification{},
	}
}

func (s *State) CanRunBackup(id types.BackupIdentifier) (canRun bool, reason string) {
	if s.StartupErr != nil {
		return false, "Startup error"
	}
	if slices.Contains(s.runningBackups, id) {
		return false, "Backup is already running"
	}
	if slices.Contains(s.occupiedRepos, id.RepositoryId) {
		return false, "Repository is busy"
	}
	return true, ""
}

func (s *State) AddRunningBackup(id types.BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningBackups = append(s.runningBackups, id)
	s.occupiedRepos = append(s.occupiedRepos, id.RepositoryId)
}

func (s *State) RemoveRunningBackup(id types.BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, backup := range s.runningBackups {
		if backup == id {
			s.runningBackups = append(s.runningBackups[:i], s.runningBackups[i+1:]...)
			break
		}
	}
	for i, repo := range s.occupiedRepos {
		if repo == id.RepositoryId {
			s.occupiedRepos = append(s.occupiedRepos[:i], s.occupiedRepos[i+1:]...)
			break
		}
	}
}

func (s *State) CanRunPruneJob(id types.BackupIdentifier) (canRun bool, reason string) {
	if s.StartupErr != nil {
		return false, "Startup error"
	}
	if slices.Contains(s.runningPruneJobs, id) {
		return false, "Prune job is already running"
	}
	if slices.Contains(s.occupiedRepos, id.RepositoryId) {
		return false, "Repository is busy"
	}
	return true, ""
}

func (s *State) AddRunningPruneJob(id types.BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runningPruneJobs = append(s.runningPruneJobs, id)
	s.occupiedRepos = append(s.occupiedRepos, id.RepositoryId)
}

func (s *State) RemoveRunningPruneJob(id types.BackupIdentifier) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, prune := range s.runningPruneJobs {
		if prune == id {
			s.runningPruneJobs = append(s.runningPruneJobs[:i], s.runningPruneJobs[i+1:]...)
			break
		}
	}
	for i, repo := range s.occupiedRepos {
		if repo == id.RepositoryId {
			s.occupiedRepos = append(s.occupiedRepos[:i], s.occupiedRepos[i+1:]...)
			break
		}
	}
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

func (s *State) SetBackupTimer(t *time.Timer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.backupTimer != nil {
		s.backupTimer.Stop()
	}

	s.backupTimer = t
}
