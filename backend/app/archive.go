package app

import (
	"fmt"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/predicate"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"slices"
	"strings"
	"time"
)

func (r *RepositoryClient) RefreshArchives(repoId int) ([]*ent.Archive, error) {
	if r.state.GetRepoState(repoId).Status != state.RepoStatusIdle {
		return nil, fmt.Errorf("can not refresh archives: the repository is busy")
	}

	repoLock := r.state.GetRepoLock(repoId)
	repoLock.Lock()         // We should not have to wait here since we checked the status before
	defer repoLock.Unlock() // Unlock at the end

	return r.refreshArchives(repoId)
}

// refreshArchives fetches the archives from the borg repository and saves them to the database.
// It also deletes the archives that don't exist anymore.
// Precondition: the caller must have acquired the lock for the repository
func (r *RepositoryClient) refreshArchives(repoId int) ([]*ent.Archive, error) {
	repo, err := r.db.Repository.
		Query().
		Where(repository.ID(repoId)).
		WithBackupProfiles().
		Only(r.ctx)
	if err != nil {
		return nil, err
	}

	// Set the repo as performing operation
	r.state.SetRepoStatus(r.ctx, repoId, state.RepoStatusPerformingOperation)
	defer r.state.SetRepoStatus(r.ctx, repoId, state.RepoStatusIdle)

	listResponse, err := r.borg.List(r.ctx, repo.Location, repo.Password)
	if err != nil {
		return nil, err
	}

	// Get all the borg ids
	borgIds := make([]string, len(listResponse.Archives))
	for i, arch := range listResponse.Archives {
		borgIds[i] = arch.ID
	}

	// Delete the archives that don't exist anymore
	cntDeletedArchives, err := r.db.Archive.
		Delete().
		Where(
			archive.And(
				archive.HasRepositoryWith(repository.ID(repoId)),
				archive.BorgIDNotIn(borgIds...),
			)).
		Exec(r.ctx)
	if err != nil {
		return nil, err
	}
	if cntDeletedArchives > 0 {
		r.log.Info(fmt.Sprintf("Deleted %d archives", cntDeletedArchives))
	}

	// Check which archives are already saved
	archives, err := r.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(repoId))).
		All(r.ctx)
	if err != nil {
		return nil, err
	}
	savedBorgIds := make([]string, len(archives))
	for i, arch := range archives {
		savedBorgIds[i] = arch.BorgID
	}

	// Save the new archives
	cntNewArchives := 0
	for _, arch := range listResponse.Archives {
		if !slices.Contains(savedBorgIds, arch.ID) {
			createdAt := time.Time(arch.Start)
			duration := time.Time(arch.End).Sub(createdAt)
			createQuery := r.db.Archive.
				Create().
				SetBorgID(arch.ID).
				SetName(arch.Name).
				SetCreatedAt(createdAt).
				SetDuration(duration.Seconds()).
				SetRepositoryID(repoId)

			// Find the backup profile that has the same prefix as the archive
			for _, backupProfile := range repo.Edges.BackupProfiles {
				if strings.HasPrefix(arch.Name, backupProfile.Prefix) {
					createQuery = createQuery.SetBackupProfileID(backupProfile.ID)
					break
				}
			}

			newArchive, err := createQuery.Save(r.ctx)
			if err != nil {
				return nil, err
			}
			archives = append(archives, newArchive)
			cntNewArchives++
		}
	}
	if cntNewArchives > 0 {
		r.log.Info(fmt.Sprintf("Saved %d new archives", cntNewArchives))
	}

	if cntDeletedArchives > 0 || cntNewArchives > 0 {
		defer runtime.EventsEmit(r.ctx, types.EventArchivesChangedString(repoId))
	}

	return archives, nil
}

func (r *RepositoryClient) DeleteArchive(id int) error {
	arch, err := r.db.Archive.
		Query().
		WithRepository().
		Where(archive.ID(id)).
		Only(r.ctx)
	if err != nil {
		return err
	}
	if canRun, reason := r.state.CanRunDeleteJob(arch.Edges.Repository.ID); !canRun {
		return fmt.Errorf("can not delete archive: %s", reason)
	}

	repoLock := r.state.GetRepoLock(arch.Edges.Repository.ID)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the repo as locked
	r.state.SetRepoStatus(r.ctx, arch.Edges.Repository.ID, state.RepoStatusPerformingOperation)
	defer r.state.SetRepoStatus(r.ctx, arch.Edges.Repository.ID, state.RepoStatusIdle)

	err = r.borg.DeleteArchive(r.ctx, arch.Edges.Repository.Location, arch.Name, arch.Edges.Repository.Password)
	if err != nil {
		return err
	}
	err = r.db.Archive.DeleteOneID(id).Exec(r.ctx)
	if err != nil {
		return err
	}
	runtime.EventsEmit(r.ctx, types.EventArchivesChangedString(arch.Edges.Repository.ID))
	return nil
}

type PaginatedArchivesRequest struct {
	// Required
	RepositoryId int `json:"repositoryId"`
	Page         int `json:"page"`
	PageSize     int `json:"pageSize"`
	// Optional
	BackupProfileFilter *BackupProfileFilter `json:"backupProfileFilter,omitempty"`
	Search              string               `json:"search,omitempty"`
	StartDate           time.Time            `json:"startDate,omitempty"`
	EndDate             time.Time            `json:"endDate,omitempty"`
}

type PaginatedArchivesResponse struct {
	Archives []*ent.Archive `json:"archives"`
	Total    int            `json:"total"`
}

func (r *RepositoryClient) GetPaginatedArchives(req *PaginatedArchivesRequest) (*PaginatedArchivesResponse, error) {
	if req.RepositoryId <= 0 {
		return nil, fmt.Errorf("repositoryId is required")
	}
	if req.Page <= 0 {
		return nil, fmt.Errorf("page is required")
	}
	if req.PageSize <= 0 {
		return nil, fmt.Errorf("pageSize is required")
	}

	// Filter by repository
	archivePredicates := []predicate.Archive{
		archive.HasRepositoryWith(repository.ID(req.RepositoryId)),
	}

	// If a backup profile filter is specified, filter by it
	if req.BackupProfileFilter != nil {
		if req.BackupProfileFilter.Id != 0 {
			// First filter by BackupProfile.ID
			archivePredicates = append(archivePredicates, archive.HasBackupProfileWith(backupprofile.ID(req.BackupProfileFilter.Id)))
		} else if req.BackupProfileFilter.IsUnknownFilter {
			// If the unknown filter is specified, filter by archives that don't have a backup profile
			archivePredicates = append(archivePredicates, archive.Not(archive.HasBackupProfile()))
		}
		// Filter by BackupProfile.Name does not have to be supported
		// Filter all is implicit
	}

	// If a search term is specified, filter by it
	if req.Search != "" {
		archivePredicates = append(archivePredicates, archive.NameContains(req.Search))
	}

	// If start date is specified, filter by it
	if !req.StartDate.IsZero() {
		archivePredicates = append(archivePredicates, archive.CreatedAtGTE(req.StartDate))
	}

	// If end date is specified, filter by it
	if !req.EndDate.IsZero() {
		archivePredicates = append(archivePredicates, archive.CreatedAtLTE(req.EndDate))
	}

	total, err := r.db.Archive.
		Query().
		Where(archive.And(archivePredicates...)).
		Count(r.ctx)
	if err != nil {
		return nil, err
	}

	archives, err := r.db.Archive.
		Query().
		WithBackupProfile(func(q *ent.BackupProfileQuery) {
			q.Select(backupprofile.FieldName)
			q.Select(backupprofile.FieldPrefix)
		}).
		Where(archive.And(archivePredicates...)).
		Order(ent.Desc(archive.FieldCreatedAt)).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		All(r.ctx)
	if err != nil {
		return nil, err
	}

	return &PaginatedArchivesResponse{
		Archives: archives,
		Total:    total,
	}, nil
}

type PruningDates struct {
	Dates []PruningDate `json:"dates"`
}

type PruningDate struct {
	ArchiveId int       `json:"archiveId"`
	NextRun   time.Time `json:"nextRun"`
}

func (r *RepositoryClient) GetPruningDates(archiveIds []int) (response PruningDates, err error) {
	archives, err := r.db.Archive.
		Query().
		Where(archive.And(
			archive.IDIn(archiveIds...),
			archive.HasBackupProfile(),
			archive.WillBePruned(true),
		)).
		WithBackupProfile(func(q *ent.BackupProfileQuery) {
			q.WithPruningRule()
		}).
		All(r.ctx)
	if err != nil {
		return
	}
	for _, arch := range archives {
		if arch.Edges.BackupProfile.Edges.PruningRule != nil {
			response.Dates = append(response.Dates, PruningDate{
				ArchiveId: arch.ID,
				NextRun:   arch.Edges.BackupProfile.Edges.PruningRule.NextRun,
			})
		}
	}
	return
}

func (r *RepositoryClient) GetLastArchiveByBackupId(backupId types.BackupId) (*ent.Archive, error) {
	backupProfile, err := r.backupClient().GetBackupProfile(backupId.BackupProfileId)
	if err != nil {
		return nil, err
	}

	first, err := r.db.Archive.
		Query().
		Where(archive.And(
			archive.HasRepositoryWith(repository.ID(backupId.RepositoryId)),
			archive.NameHasPrefix(backupProfile.Prefix),
		)).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(r.ctx)
	if err != nil && ent.IsNotFound(err) {
		return nil, nil
	}
	return first, err
}

func (r *RepositoryClient) GetLastArchiveByRepoId(repoId int) (*ent.Archive, error) {
	first, err := r.db.Archive.
		Query().
		Where(archive.And(
			archive.HasRepositoryWith(repository.ID(repoId)),
		)).
		Order(ent.Desc(archive.FieldCreatedAt)).
		First(r.ctx)
	if err != nil && ent.IsNotFound(err) {
		return nil, nil
	}
	return first, err
}

func (r *RepositoryClient) getArchive(id int) (*ent.Archive, error) {
	return r.db.Archive.
		Query().
		WithRepository().
		WithBackupProfile().
		Where(archive.ID(id)).
		Only(r.ctx)
}

func (r *RepositoryClient) RenameArchive(id int, prefix, name string) error {
	r.log.Debugf("Renaming archive %d to %s", id, name)
	validationError, err := r.validationClient().ArchiveName(id, prefix, name)
	if err != nil {
		return err
	}
	if validationError != "" {
		return fmt.Errorf("can not rename archive: %s", validationError)
	}

	newName := prefix + name

	arch, err := r.getArchive(id)
	if err != nil {
		return err
	}

	if r.state.GetRepoState(arch.Edges.Repository.ID).Status != state.RepoStatusIdle {
		return fmt.Errorf("can not rename archive: the repository is busy")
	}

	repoLock := r.state.GetRepoLock(arch.Edges.Repository.ID)
	repoLock.Lock()         // We should not have to wait here since we checked the status before
	defer repoLock.Unlock() // Unlock at the end

	err = r.borg.Rename(r.ctx, arch.Edges.Repository.Location, arch.Name, arch.Edges.Repository.Password, newName)
	if err != nil {
		return fmt.Errorf("failed to rename archive: %w", err)
	}

	return r.db.Archive.
		UpdateOneID(id).
		SetName(newName).
		Exec(r.ctx)
}
