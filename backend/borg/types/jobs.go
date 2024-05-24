package types

import "time"

type InputChannels struct {
	StartBackup chan BackupJob
}

type OutputChannels struct {
	FinishBackup chan FinishBackupJob
}

type BackupIdentifier struct {
	BackupProfileId int
	RepositoryId    int
}

type BackupJob struct {
	Id           BackupIdentifier
	RepoUrl      string
	RepoPassword string
	Hostname     string
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
