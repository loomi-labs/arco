package types

import (
	"fmt"
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

type PruneJob struct {
	Id           BackupIdentifier
	RepoUrl      string
	RepoPassword string
	Prefix       string
}

type DeleteJob struct {
	Id           BackupIdentifier
	RepoUrl      string
	RepoPassword string
	Prefix       string
}
