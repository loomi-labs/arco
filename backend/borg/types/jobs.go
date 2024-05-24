package types

import "time"

type InputChannels struct {
	StartBackup chan BackupJob
}

type OutputChannels struct {
	FinishBackup chan FinishBackupJob
}

type BackupJob struct {
	BackupProfileId int
	RepositoryId    int
	RepoUrl         string
	RepoPassword    string
	Hostname        string
	Directories     []string
	BinaryPath      string
}

type FinishBackupJob struct {
	BackupProfileId int
	RepositoryId    int
	StartTime       time.Time
	EndTime         time.Time
	Cmd             string
	Err             error
}
