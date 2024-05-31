package types

import "time"

type InputChannels struct {
	StartBackup chan BackupJob
	StartPrune  chan PruneJob
}

func NewInputChannels() *InputChannels {
	return &InputChannels{
		StartBackup: make(chan BackupJob),
		StartPrune:  make(chan PruneJob),
	}
}

type OutputChannels struct {
	FinishBackup chan FinishBackupJob
	FinishPrune  chan FinishPruneJob
}

func NewOutputChannels() *OutputChannels {
	return &OutputChannels{
		FinishBackup: make(chan FinishBackupJob),
		FinishPrune:  make(chan FinishPruneJob),
	}
}

type BackupIdentifier struct {
	BackupProfileId int
	RepositoryId    int
}

type BackupJob struct {
	Id           BackupIdentifier
	RepoUrl      string
	RepoPassword string
	Prefix       string
	Directories  []string
	BinaryPath   string
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
