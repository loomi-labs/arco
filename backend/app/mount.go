package app

import (
	"arco/backend/app/state"
	"arco/backend/app/types"
	"arco/backend/ent"
	"github.com/prometheus/procfs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
)

func (r *RepositoryClient) MountRepository(repoId int) (state state.MountState, err error) {
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

	if err = r.borg.MountRepository(repo.URL, repo.Password, path); err != nil {
		return
	}

	// Update the mount state
	state, err = getMountState(path)
	if err != nil {
		return
	}
	r.state.SetRepoMount(repoId, &state)

	// Open the file manager and forget about it
	go r.openFileManager(path)
	return
}

func (r *RepositoryClient) MountArchive(archiveId int) (state state.MountState, err error) {
	archive, err := r.getArchive(archiveId)
	if err != nil {
		return
	}

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
		if err = r.borg.MountArchive(archive.Edges.Repository.URL, archive.Name, archive.Edges.Repository.Password, path); err != nil {
			return
		}

		// Update the mount state
		state, err = getMountState(path)
		if err != nil {
			return
		}
		r.state.SetArchiveMount(archive.Edges.Repository.ID, archiveId, &state)
	}

	// Open the file manager and forget about it
	go r.openFileManager(path)
	return
}

func (r *RepositoryClient) UnmountRepository(repoId int) (state state.MountState, err error) {
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
	r.state.SetRepoMount(repoId, &mountState)
	return
}

func (r *RepositoryClient) UnmountArchive(archiveId int) (state state.MountState, err error) {
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
	r.state.SetArchiveMount(archive.Edges.Repository.ID, archiveId, &mountState)
	return
}

func (r *RepositoryClient) GetRepoMountState(repoId int) state.MountState {
	return r.state.GetRepoMount(repoId)
}

func (r *RepositoryClient) GetArchiveMountStates(repoId int) (states map[int]state.MountState, err error) {
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
		r.state.SetRepoMount(repo.ID, &mountState)

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

		states, err := getMountStates(paths)
		if err != nil {
			r.log.Error("Error getting mount states: ", err)
			continue
		}
		r.state.SetArchiveMounts(repo.ID, states)
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
	return filepath.Join("/run/user", currentUser.Uid, "arco", name), nil
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

func getMountState(path string) (state state.MountState, err error) {
	states, err := getMountStates(map[int]string{0: path})
	if err != nil {
		return
	}
	if len(states) == 0 {
		return
	}
	return *states[0], nil
}

func getMountStates(paths map[int]string) (states map[int]*state.MountState, err error) {
	states = make(map[int]*state.MountState)

	mounts, err := procfs.GetMounts()
	if err != nil {
		return
	}

	for _, mount := range mounts {
		for id, path := range paths {
			if mount.MountPoint == path {
				states[id] = &state.MountState{
					IsMounted: true,
					MountPath: mount.MountPoint,
				}
			}
		}
	}
	return
}
