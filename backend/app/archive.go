package app

import (
	"arco/backend/app/state"
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
			createdAt, err := time.Parse("2006-01-02T15:04:05.000000", arch.Start)
			if err != nil {
				return nil, err
			}
			duration, err := time.Parse("2006-01-02T15:04:05.000000", arch.Time)
			if err != nil {
				return nil, err
			}
			newArchive, err := r.db.Archive.
				Create().
				SetBorgID(arch.ID).
				SetName(arch.Name).
				SetCreatedAt(createdAt).
				SetDuration(duration).
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

	return r.borg.DeleteArchive(r.ctx, arch.Edges.Repository.URL, arch.Name, arch.Edges.Repository.Password)
}

func (r *RepositoryClient) getArchive(id int) (*ent.Archive, error) {
	return r.db.Archive.
		Query().
		WithRepository().
		Where(archive.ID(id)).
		Only(r.ctx)
}
