package app

import (
	"arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/ent"
	"arco/backend/ent/archive"
	"arco/backend/ent/predicate"
	"arco/backend/ent/repository"
	"fmt"
	"slices"
	"time"
)

func (r *RepositoryClient) RefreshArchives(repoId int) ([]*ent.Archive, error) {
	repo, err := r.Get(repoId)
	if err != nil {
		return nil, err
	}

	repoLock := r.state.GetRepoLock(repoId)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	// Wait to acquire the lock and then set the repo as fetching info
	r.state.SetRepoStatus(r.ctx, repoId, state.RepoStatusPerformingOperation)
	defer r.state.SetRepoStatus(r.ctx, repoId, state.RepoStatusIdle)

	listResponse, err := r.borg.List(repo.Location, repo.Password)
	if err != nil {
		return nil, err
	}

	// Get all the borg ids
	borgIds := make([]string, len(listResponse.Archives))
	for i, arch := range listResponse.Archives {
		borgIds[i] = arch.ID
	}

	// Delete the archives that don't exist anymore
	cnt, err := r.db.Archive.
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
	if cnt > 0 {
		r.log.Info(fmt.Sprintf("Deleted %d archives", cnt))
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
			newArchive, err := r.db.Archive.
				Create().
				SetBorgID(arch.ID).
				SetName(arch.Name).
				SetCreatedAt(time.Time(arch.Start)).
				SetDuration(time.Time(arch.Time)).
				SetRepositoryID(repoId).
				Save(r.ctx)
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
	return r.db.Archive.DeleteOneID(id).Exec(r.ctx)
}

type PaginatedArchivesRequest struct {
	RepositoryId    int    `json:"repositoryId"`
	BackupProfileId int    `json:"backupProfileId,omitempty"`
	Page            int    `json:"page"`
	PageSize        int    `json:"pageSize"`
	Search          string `json:"search,omitempty"`
}

type PaginatedArchivesResponse struct {
	Archives []*ent.Archive `json:"archives"`
	Total    int            `json:"total"`
}

func (r *RepositoryClient) GetPaginatedArchives(req *PaginatedArchivesRequest) (*PaginatedArchivesResponse, error) {
	if req.RepositoryId == 0 {
		return nil, fmt.Errorf("repositoryId is required")
	}

	// Filter by repository
	archivePredicates := []predicate.Archive{
		archive.HasRepositoryWith(repository.ID(req.RepositoryId)),
	}

	// If a backup profile is specified, filter by its prefix
	if req.BackupProfileId != 0 {
		backupProfile, err := r.backupClient().GetBackupProfile(req.BackupProfileId)
		if err != nil {
			return nil, err
		}
		archivePredicates = append(archivePredicates, archive.NameHasPrefix(backupProfile.Prefix))
	}

	// If a search term is specified, filter by it
	if req.Search != "" {
		archivePredicates = append(archivePredicates, archive.NameContains(req.Search))
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
		Where(archive.ID(id)).
		Only(r.ctx)
}
