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
