package app

import (
	"github.com/loomi-labs/arco/backend/app/backup_profile"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
)

/***********************************/
/********** Backup Profile *********/
/***********************************/

func (b *BackupClient) GetBackupProfile(id int) (*ent.BackupProfile, error) {
	return (*App)(b).backupProfileService.GetBackupProfile((*App)(b).ctx, id)
}

func (b *BackupClient) GetBackupProfiles() ([]*ent.BackupProfile, error) {
	return (*App)(b).backupProfileService.GetBackupProfiles((*App)(b).ctx)
}

func (b *BackupClient) GetBackupProfileFilterOptions(repoId int) ([]backup_profile.BackupProfileFilter, error) {
	return (*App)(b).backupProfileService.GetBackupProfileFilterOptions((*App)(b).ctx, repoId)
}

func (b *BackupClient) NewBackupProfile() (*ent.BackupProfile, error) {
	return (*App)(b).backupProfileService.NewBackupProfile((*App)(b).ctx)
}

func (b *BackupClient) GetDirectorySuggestions() []string {
	return (*App)(b).backupProfileService.GetDirectorySuggestions()
}

func (b *BackupClient) DoesPathExist(path string) bool {
	return (*App)(b).backupProfileService.DoesPathExist(path)
}

func (b *BackupClient) IsDirectory(path string) bool {
	return (*App)(b).backupProfileService.IsDirectory(path)
}

func (b *BackupClient) IsDirectoryEmpty(path string) bool {
	return (*App)(b).backupProfileService.IsDirectoryEmpty(path)
}

func (b *BackupClient) CreateDirectory(path string) error {
	return (*App)(b).backupProfileService.CreateDirectory(path)
}

func (b *BackupClient) GetPrefixSuggestion(name string) (string, error) {
	return (*App)(b).backupProfileService.GetPrefixSuggestion((*App)(b).ctx, name)
}

func (b *BackupClient) CreateBackupProfile(backup ent.BackupProfile, repositoryIds []int) (*ent.BackupProfile, error) {
	return (*App)(b).backupProfileService.CreateBackupProfile((*App)(b).ctx, backup, repositoryIds)
}

func (b *BackupClient) UpdateBackupProfile(backup ent.BackupProfile) (*ent.BackupProfile, error) {
	return (*App)(b).backupProfileService.UpdateBackupProfile((*App)(b).ctx, backup)
}

func (b *BackupClient) DeleteBackupProfile(backupProfileId int, deleteArchives bool) error {
	return (*App)(b).backupProfileService.DeleteBackupProfile((*App)(b).ctx, backupProfileId, deleteArchives)
}

func (b *BackupClient) AddRepositoryToBackupProfile(backupProfileId int, repositoryId int) error {
	return (*App)(b).backupProfileService.AddRepositoryToBackupProfile((*App)(b).ctx, backupProfileId, repositoryId)
}

func (b *BackupClient) RemoveRepositoryFromBackupProfile(backupProfileId int, repositoryId int, deleteArchives bool) error {
	return (*App)(b).backupProfileService.RemoveRepositoryFromBackupProfile((*App)(b).ctx, backupProfileId, repositoryId, deleteArchives)
}

func (b *BackupClient) SelectDirectory(data backup_profile.SelectDirectoryData) (string, error) {
	return (*App)(b).backupProfileService.SelectDirectory(data)
}

/***********************************/
/********** Backup Functions *******/
/***********************************/

func (b *BackupClient) StartBackupJobs(bIds []types.BackupId) error {
	return (*App)(b).repositoryService.StartBackupJobs((*App)(b).ctx, bIds)
}

func (b *BackupClient) AbortBackupJobs(bIds []types.BackupId) error {
	return (*App)(b).repositoryService.AbortBackupJobs((*App)(b).ctx, bIds)
}

func (b *BackupClient) GetState(bId types.BackupId) state.BackupState {
	return (*App)(b).repositoryService.GetBackupState(bId)
}

func (b *BackupClient) GetBackupButtonStatus(bIds []types.BackupId) state.BackupButtonStatus {
	return (*App)(b).repositoryService.GetBackupButtonStatus(bIds)
}

func (b *BackupClient) GetCombinedBackupProgress(bIds []types.BackupId) *borgtypes.BackupProgress {
	return (*App)(b).repositoryService.GetCombinedBackupProgress(bIds)
}

func (b *BackupClient) GetLastBackupErrorMsg(bId types.BackupId) (string, error) {
	return (*App)(b).repositoryService.GetLastBackupErrorMsgByBackupId((*App)(b).ctx, bId)
}

/***********************************/
/********** Backup Schedule ********/
/***********************************/

func (b *BackupClient) SaveBackupSchedule(backupProfileId int, schedule ent.BackupSchedule) error {
	return (*App)(b).backupProfileService.SaveBackupSchedule((*App)(b).ctx, backupProfileId, schedule)
}
