package app

import (
	"arco/backend/app/types"
	"arco/backend/ent"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
)

func (r *RepositoryClient) MountRepository(repoId int) (state types.MountState, err error) {
	repo, err := r.Get(repoId)
	if err != nil {
		return
	}

	path, err := getRepoMountPath(repo)
	if err != nil {
		return
	}

	err = ensurePathExists(path)
	if err != nil {
		return
	}

	if err = r.borg.MountRepository(repo.Location, repo.Password, path); err != nil {
		return
	}

	// Update the mount state
	state, err = getMountState(path)
	if err != nil {
		return
	}
	r.state.SetRepoMount(r.ctx, repoId, &state)

	// Open the file manager and forget about it
	go r.openFileManager(path)
	return
}

func (r *RepositoryClient) MountArchive(archiveId int) (state types.MountState, err error) {
	archive, err := r.getArchive(archiveId)
	if err != nil {
		return
	}

	if canMount, reason := r.state.CanMountRepo(archive.Edges.Repository.ID); !canMount {
		err = fmt.Errorf("can not mount archive: %s", reason)
		return
	}
	repoLock := r.state.GetRepoLock(archive.Edges.Repository.ID)
	repoLock.Lock()         // We might wait here for other operations to finish
	defer repoLock.Unlock() // Unlock at the end

	path, err := getArchiveMountPath(archive)
	if err != nil {
		return
	}

	err = ensurePathExists(path)
	if err != nil {
		return
	}

	// Check current mount state
	state, err = getMountState(path)
	if err != nil {
		return
	}
	if !state.IsMounted {
		// If not mounted, mount it
		if err = r.borg.MountArchive(archive.Edges.Repository.Location, archive.Name, archive.Edges.Repository.Password, path); err != nil {
			return
		}

		// Update the mount state
		state, err = getMountState(path)
		if err != nil {
			return
		}
		r.state.SetArchiveMount(r.ctx, archive.Edges.Repository.ID, archiveId, &state)
	}

	// Open the file manager and forget about it
	go r.openFileManager(path)
	return
}

func (r *RepositoryClient) UnmountAllForRepos(repoIds []int) error {
	var unmountErrors []error
	for _, repoId := range repoIds {
		mount := r.GetRepoMountState(repoId)
		if mount.IsMounted {
			if _, err := r.UnmountRepository(repoId); err != nil {
				unmountErrors = append(unmountErrors, fmt.Errorf("error unmounting repository %d: %w", repoId, err))
			}
		}
		if states, err := r.GetArchiveMountStates(repoId); err != nil {
			unmountErrors = append(unmountErrors, fmt.Errorf("error getting archive mount states for repository %d: %w", repoId, err))
		} else {
			for archiveId, state := range states {
				if state.IsMounted {
					if _, err = r.UnmountArchive(archiveId); err != nil {
						unmountErrors = append(unmountErrors, fmt.Errorf("error unmounting archive %d: %w", archiveId, err))
					}
				}
			}
		}
	}
	if len(unmountErrors) > 0 {
		return fmt.Errorf("unmount errors: %v", unmountErrors)
	}
	return nil
}

func (r *RepositoryClient) UnmountRepository(repoId int) (state types.MountState, err error) {
	repo, err := r.Get(repoId)
	if err != nil {
		return
	}

	path, err := getRepoMountPath(repo)
	if err != nil {
		return
	}

	if err = r.borg.Umount(path); err != nil {
		return
	}

	// Update the mount state
	mountState, err := getMountState(path)
	if err != nil {
		return
	}
	r.state.SetRepoMount(r.ctx, repoId, &mountState)
	return
}

func (r *RepositoryClient) UnmountArchive(archiveId int) (state types.MountState, err error) {
	archive, err := r.getArchive(archiveId)
	if err != nil {
		return
	}

	path, err := getArchiveMountPath(archive)
	if err != nil {
		return
	}

	if err = r.borg.Umount(path); err != nil {
		return
	}

	// Update the mount state
	mountState, err := getMountState(path)
	if err != nil {
		return
	}
	r.state.SetArchiveMount(r.ctx, archive.Edges.Repository.ID, archiveId, &mountState)
	return
}

func (r *RepositoryClient) GetRepoMountState(repoId int) types.MountState {
	return r.state.GetRepoMount(repoId)
}

func (r *RepositoryClient) GetArchiveMountStates(repoId int) (states map[int]types.MountState, err error) {
	repo, err := r.Get(repoId)
	if err != nil {
		return
	}
	return r.state.GetArchiveMounts(repo.ID), nil
}

// setMountStates sets the mount states of all repositories and archives to the state
func (r *RepositoryClient) setMountStates() {
	repos, err := r.All()
	if err != nil {
		r.log.Error("Error getting all repositories: ", err)
		return
	}
	for _, repo := range repos {
		// Save the mount state for the repository
		path, err := getRepoMountPath(repo)
		if err != nil {
			return
		}
		mountState, err := getMountState(path)
		if err != nil {
			r.log.Error("Error getting mount state: ", err)
			continue
		}
		r.state.SetRepoMount(r.ctx, repo.ID, &mountState)

		// Save the mount states for all archives of the repository
		archives, err := repo.QueryArchives().All(r.ctx)
		if err != nil {
			r.log.Error("Error getting all archives: ", err)
			continue
		}
		var paths = make(map[int]string)
		for _, arch := range archives {
			archivePath, err := getArchiveMountPath(arch)
			if err != nil {
				r.log.Error("Error getting archive mount path: ", err)
				continue
			}
			paths[arch.ID] = archivePath
		}

		states, err := types.GetMountStates(paths)
		if err != nil {
			r.log.Error("Error getting mount states: ", err)
			continue
		}
		r.state.SetArchiveMounts(r.ctx, repo.ID, states)
	}
}

func (r *RepositoryClient) openFileManager(path string) {
	openCmd, err := types.GetOpenFileManagerCmd()
	if err != nil {
		r.log.Error("Error getting open file manager command: ", err)
		return
	}
	cmd := exec.Command(openCmd, path)
	err = cmd.Run()
	if err != nil {
		r.log.Error("Error opening file manager: ", err)
	}
}

func getMountPath(name string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	mountPath, err := types.GetMountPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(mountPath, currentUser.Uid, "arco", name), nil
}

func getRepoMountPath(repo *ent.Repository) (string, error) {
	return getMountPath("repo-" + strconv.Itoa(repo.ID))
}

func getArchiveMountPath(archive *ent.Archive) (string, error) {
	return getMountPath("archive-" + strconv.Itoa(archive.ID))
}

func ensurePathExists(path string) error {
	// Check if the directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		//Create the directory
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func getMountState(path string) (state types.MountState, err error) {
	states, err := types.GetMountStates(map[int]string{0: path})
	if err != nil {
		return
	}
	if len(states) == 0 {
		return
	}
	return *states[0], nil
}
