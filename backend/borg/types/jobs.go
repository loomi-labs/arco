package types

import "time"

type Channels struct {
	StartBackup  chan BackupJob
	FinishBackup chan FinishBackupJob
	Notification chan string
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
