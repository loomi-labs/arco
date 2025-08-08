package app

import (
	"github.com/loomi-labs/arco/backend/app/backup_profile"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
)

func (b *BackupClient) GetPruningOptions() backup_profile.GetPruningOptionsResponse {
	return (*App)(b).backupProfileService.GetPruningOptions()
}

func (b *BackupClient) SavePruningRule(backupId int, rule ent.PruningRule) (*ent.PruningRule, error) {
	return (*App)(b).backupProfileService.SavePruningRule((*App)(b).ctx, backupId, rule)
}

func (b *BackupClient) StartPruneJob(bId types.BackupId) error {
	return (*App)(b).repositoryService.StartPruneJob((*App)(b).ctx, bId)
}

func (b *BackupClient) ExaminePrunes(backupProfileId int, pruningRule *ent.PruningRule, saveResults bool) []types.ExaminePruningResult {
	results := (*App)(b).repositoryService.ExaminePrunes((*App)(b).ctx, backupProfileId, pruningRule, saveResults)
	
	// Convert to the expected type
	var convertedResults []types.ExaminePruningResult
	for _, r := range results {
		convertedResults = append(convertedResults, types.ExaminePruningResult{
			BackupID:               r.BackupID,
			RepositoryName:         r.RepositoryName,
			CntArchivesToBeDeleted: r.CntArchivesToBeDeleted,
			Error:                  r.Error,
		})
	}
	return convertedResults
}