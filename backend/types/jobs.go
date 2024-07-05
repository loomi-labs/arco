package types

import (
	"fmt"
	"time"
)

type BackupIdentifier struct {
	BackupProfileId int
	RepositoryId    int
}

func (b BackupIdentifier) String() string {
	return fmt.Sprintf("BackupProfileId: %d, RepositoryId: %d", b.BackupProfileId, b.RepositoryId)
}

type BackupJob struct {
	Id           BackupIdentifier
	RepoUrl      string
	RepoPassword string
	Prefix       string
	Directories  []string
	IsQuiet      bool
}

type FinishBackupJob struct {
	Id        BackupIdentifier
	StartTime time.Time
	EndTime   time.Time
	Cmd       string
	Err       error
}

type PruneJob struct {
	Id           BackupIdentifier
	RepoUrl      string
	RepoPassword string
	Prefix       string
	BinaryPath   string
}

type FinishPruneJob struct {
	Id         BackupIdentifier
	StartTime  time.Time
	EndTime    time.Time
	PruneCmd   string
	PruneErr   error
	CompactCmd string
	CompactErr error
}
