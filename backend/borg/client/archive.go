package client

import (
	"arco/backend/borg/util"
	"arco/backend/ent"
	"arco/backend/ent/archive"
	"arco/backend/ent/repository"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"time"
)

func (b *BorgClient) RefreshArchives(repoId int) ([]*ent.Archive, error) {
	repo, err := b.GetRepository(repoId)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(b.binaryPath, "list", "--json", repo.URL)
	cmd.Env = util.CreateEnv(repo.Password)

	// Get the list from the borg repository
	startTime := time.Now()
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", out, err)
	}
	b.log.Info(fmt.Sprintf("Command took %s", time.Since(startTime)))

	var listResponse ListResponse
	err = json.Unmarshal(out, &listResponse)
	if err != nil {
		return nil, err
	}

	// Get all the borg ids
	borgIds := make([]string, len(listResponse.Archives))
	for i, arch := range listResponse.Archives {
		borgIds[i] = arch.ID
	}

	// Delete the archives that don't exist anymore
	cnt, err := b.db.Archive.
		Delete().
		Where(
			archive.And(
				archive.HasRepositoryWith(repository.ID(repoId)),
				archive.BorgIDNotIn(borgIds...),
			)).
		Exec(b.ctx)
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		b.log.Info(fmt.Sprintf("Deleted %d archives", cnt))
	}

	// Check which archives are already saved
	archives, err := b.db.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(repoId))).
		All(b.ctx)
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
			newArchive, err := b.db.Archive.
				Create().
				SetBorgID(arch.ID).
				SetName(arch.Name).
				SetCreatedAt(createdAt).
				SetDuration(duration).
				SetRepositoryID(repoId).
				Save(b.ctx)
			if err != nil {
				return nil, err
			}
			archives = append(archives, newArchive)
			cntNewArchives++
		}
	}
	if cntNewArchives > 0 {
		b.log.Info(fmt.Sprintf("Saved %d new archives", cntNewArchives))
	}

	return archives, nil
}

func (b *BorgClient) DeleteArchive(id int) error {
	arch, err := b.db.Archive.
		Query().
		WithRepository().
		Where(archive.ID(id)).
		Only(b.ctx)
	if err != nil {
		return err
	}

	cmd := exec.Command(b.binaryPath, "delete", fmt.Sprintf("%s::%s", arch.Edges.Repository.URL, arch.Name))
	cmd.Env = util.CreateEnv(arch.Edges.Repository.Password)

	startTime := time.Now()
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))
	defer b.log.Info(fmt.Sprintf("Command took %s", time.Since(startTime)))

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}
	return nil
}
