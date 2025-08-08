package app

import (
	"github.com/loomi-labs/arco/backend/app/backup_profile"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	borgtypes "github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/repository"
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
	app := (*App)(b)
	backupProfile, err := app.backupProfileService.GetBackupProfile(app.ctx, backupProfileId)
	if err != nil {
		return err
	}

	var deleteJobs []func()
	// If deleteArchives is true, we prepare a delete job for each repository
	if deleteArchives {
		for _, r := range backupProfile.Edges.Repositories {
			repo := r // Capture loop variable
			bId := types.BackupId{
				BackupProfileId: backupProfileId,
				RepositoryId:    repo.ID,
			}
			deleteJobs = append(deleteJobs, func() {
				go func() {
					_, err := app.repositoryService.RunBorgDelete(app.ctx, bId, repo.URL, repo.Password, backupProfile.Prefix)
					if err != nil {
						app.log.Error("Delete job failed: ", err)
					}
				}()
			})
		}
	}

	return app.backupProfileService.DeleteBackupProfile(app.ctx, backupProfileId, deleteJobs)
}

func (b *BackupClient) AddRepositoryToBackupProfile(backupProfileId int, repositoryId int) error {
	return (*App)(b).backupProfileService.AddRepositoryToBackupProfile((*App)(b).ctx, backupProfileId, repositoryId)
}

func (b *BackupClient) RemoveRepositoryFromBackupProfile(backupProfileId int, repositoryId int, deleteArchives bool) error {
	app := (*App)(b)
	
	// Get the backup profile with the repository
	backupProfile, err := app.db.BackupProfile.
		Query().
		Where(backupprofile.And(
			backupprofile.ID(backupProfileId),
			backupprofile.HasRepositoriesWith(repository.ID(repositoryId)),
		)).
		WithRepositories(func(q *ent.RepositoryQuery) {
			q.Where(repository.ID(repositoryId))
		}).
		Only(app.ctx)
	if err != nil {
		return err
	}
	
	var deleteJob func()
	// If deleteArchives is true, we run a delete job for the repository
	if deleteArchives && len(backupProfile.Edges.Repositories) > 0 {
		bId := types.BackupId{
			BackupProfileId: backupProfileId,
			RepositoryId:    repositoryId,
		}
		repo := backupProfile.Edges.Repositories[0]
		if repo.ID == repositoryId {
			location, password, prefix := repo.URL, repo.Password, backupProfile.Prefix
			deleteJob = func() {
				_, err := app.repositoryService.RunBorgDelete(app.ctx, bId, location, password, prefix)
				if err != nil {
					app.log.Error("Delete job failed: ", err)
				}
			}
		}
	}

	return app.backupProfileService.RemoveRepositoryFromBackupProfile(app.ctx, backupProfileId, repositoryId, deleteJob)
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

type BackupProgressResponse struct {
	BackupId types.BackupId           `json:"backupId"`
	Progress borgtypes.BackupProgress `json:"progress"`
	Found    bool                     `json:"found"`
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