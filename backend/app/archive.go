package app

import (
	"arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/ent"
	"arco/backend/ent/archive"
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
	repoLock.Lock()

	// Wait to acquire the lock and then set the repo as fetching info
	r.state.SetRepoState(repoId, state.RepoStatePerformingOperation)
	defer r.state.SetRepoState(repoId, state.RepoStateIdle)

	listResponse, err := r.borg.List(repo.URL, repo.Password)
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

	repoLock := r.state.GetRepoLock(arch.Edges.Repository.ID)
	repoLock.Lock()

	// Wait to acquire the lock and then set the repo as locked
	r.state.SetRepoState(arch.Edges.Repository.ID, state.RepoStatePerformingOperation)
	defer r.state.SetRepoState(arch.Edges.Repository.ID, state.RepoStateIdle)

	err = r.borg.DeleteArchive(r.ctx, arch.Edges.Repository.URL, arch.Name, arch.Edges.Repository.Password)
	if err != nil {
		return err
	}
	return r.db.Archive.DeleteOneID(id).Exec(r.ctx)
}

type PaginatedArchivesResponse struct {
	Archives []*ent.Archive `json:"archives"`
	Total    int            `json:"total"`
}

func (r *RepositoryClient) GetPaginatedArchives(backupId types.BackupId, page, pageSize int) (*PaginatedArchivesResponse, error) {
	backupProfile, err := r.backupClient().GetBackupProfile(backupId.BackupProfileId)
	if err != nil {
		return nil, err
	}

	archiveQuery := r.db.Archive.Query().
		Where(archive.And(
			archive.HasRepositoryWith(repository.ID(backupId.RepositoryId)),
			archive.NameHasPrefix(backupProfile.Prefix),
		))

	total, err := archiveQuery.Count(r.ctx)
	if err != nil {
		return nil, err
	}

	archives, err := r.db.Archive.
		Query().
		Where(archive.And(
			archive.HasRepositoryWith(repository.ID(backupId.RepositoryId)),
			archive.NameHasPrefix(backupProfile.Prefix),
		)).
		Order(ent.Desc(archive.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(r.ctx)
	if err != nil {
		return nil, err
	}

	return &PaginatedArchivesResponse{
		Archives: archives,
		Total:    total,
	}, nil
}

func (r *RepositoryClient) getArchive(id int) (*ent.Archive, error) {
	return r.db.Archive.
		Query().
		WithRepository().
		Where(archive.ID(id)).
		Only(r.ctx)
}
