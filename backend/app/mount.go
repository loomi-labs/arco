package app

import (
	"arco/backend/ent"
	"arco/backend/util"
	"fmt"
	"github.com/prometheus/procfs"
	"golang.org/x/exp/maps"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
)

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

func (r *RepositoryClient) openFileManager(path string) {
	openCmd, err := util.GetOpenFileManagerCmd()
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

func (r *RepositoryClient) MountRepository(repoId int) (state MountState, err error) {
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

	cmd := exec.Command(r.config.BorgPath, "mount", repo.URL, path)
	cmd.Env = util.BorgEnv{}.WithPassword(repo.Password).AsList()

	startTime := r.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return state, r.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	r.log.LogCmdEnd(cmd.String(), startTime)

	// Open the file manager and forget about it
	go r.openFileManager(path)

	return r.getMountState(path)
}

func (r *RepositoryClient) MountArchive(archiveId int) (state MountState, err error) {
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

	cmd := exec.Command(r.config.BorgPath, "mount", fmt.Sprintf("%s::%s", archive.Edges.Repository.URL, archive.Name), path)
	cmd.Env = util.BorgEnv{}.WithPassword(archive.Edges.Repository.Password).AsList()

	startTime := r.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return state, r.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	r.log.LogCmdEnd(cmd.String(), startTime)

	// Open the file manager and forget about it
	go r.openFileManager(path)

	return r.getMountState(path)
}

func (r *RepositoryClient) unmount(path string) (state MountState, err error) {
	cmd := exec.Command(r.config.BorgPath, "umount", path)
	r.log.Debug("Command: ", cmd.String())

	out, err := cmd.CombinedOutput()
	if err != nil {
		r.log.Error("Error running unmount command: ", fmt.Errorf("%s: %s", out, err))
		return
	}
	r.log.Debug("Unmount finished", out)
	return r.getMountState(path)
}

func (r *RepositoryClient) UnmountRepository(repoId int) (state MountState, err error) {
	repo, err := r.Get(repoId)
	if err != nil {
		return
	}

	path, err := getRepoMountPath(repo)
	if err != nil {
		return
	}

	return r.unmount(path)
}

func (r *RepositoryClient) UnmountArchive(archiveId int) (state MountState, err error) {
	archive, err := r.getArchive(archiveId)
	if err != nil {
		return
	}

	path, err := getArchiveMountPath(archive)
	if err != nil {
		return
	}

	return r.unmount(path)
}

// TODO: move mount state to state
func getMounts(mountPaths ...string) (mounts []*procfs.MountInfo, err error) {
	allMounts, err := procfs.GetMounts()
	if err != nil {
		return
	}

	// Filter out the mounts we are interested in
	for _, mount := range allMounts {
		for _, mountPath := range mountPaths {
			if mount.MountPoint == mountPath {
				mounts = append(mounts, mount)
			}
		}
	}
	return
}

func (r *RepositoryClient) getMountState(mountPath string) (state MountState, err error) {
	mounts, err := getMounts(mountPath)
	if err != nil {
		return
	}
	if len(mounts) == 0 {
		return
	}
	state.IsMounted = true
	state.MountPath = mountPath
	return
}

func (r *RepositoryClient) GetRepositoryMountState(repoId int) (state MountState, err error) {
	repo, err := r.Get(repoId)
	if err != nil {
		return
	}

	path, err := getRepoMountPath(repo)
	if err != nil {
		return
	}

	return r.getMountState(path)
}

func (r *RepositoryClient) GetArchiveMountStates(repoId int) (states map[int]*MountState, err error) {
	states = make(map[int]*MountState)
	repo, err := r.Get(repoId)
	if err != nil {
		return
	}

	// Get all the archive mount paths
	pathMap := make(map[string]*ent.Archive, len(repo.Edges.Archives))
	for _, archive := range repo.Edges.Archives {
		path := ""
		path, err = getArchiveMountPath(archive)
		if err != nil {
			return
		}
		pathMap[path] = archive
	}

	// Get all the archives that are currently mounted
	mounts, err := getMounts(maps.Keys(pathMap)...)
	if err != nil {
		return
	}
	for _, mount := range mounts {
		archive := pathMap[mount.MountPoint]
		states[archive.ID] = &MountState{
			IsMounted: true,
			MountPath: mount.MountPoint,
		}
	}
	return
}