package client

import (
	"arco/backend/borg/util"
	"arco/backend/ent"
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

func (b *BorgClient) MountRepository(repoId int) (state MountState, err error) {
	repo, err := b.GetRepository(repoId)
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

	cmd := exec.Command(b.binaryPath, "mount", repo.URL, path)
	cmd.Env = util.BorgEnv{}.WithPassword(repo.Password).AsList()
	b.log.Debug("Command: ", cmd.String())

	out, err := cmd.CombinedOutput()
	if err != nil {
		b.log.Error("Error running mount command: ", fmt.Errorf("%s: %s", out, err))
		return
	}
	b.log.Debug("Backup job finished", out)
	return b.getMountState(path)
}

func (b *BorgClient) MountArchive(archiveId int) (state MountState, err error) {
	archive, err := b.getArchive(archiveId)
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

	cmd := exec.Command(b.binaryPath, "mount", fmt.Sprintf("%s::%s", archive.Edges.Repository.URL, archive.Name), path)
	cmd.Env = util.BorgEnv{}.WithPassword(archive.Edges.Repository.Password).AsList()
	b.log.Debug("Command: ", cmd.String())

	out, err := cmd.CombinedOutput()
	if err != nil {
		b.log.Error("Error running mount command: ", fmt.Errorf("%s: %s", out, err))
		return
	}
	b.log.Debug("Command finished", out)
	return b.getMountState(path)
}

func (b *BorgClient) unmount(path string) (state MountState, err error) {
	cmd := exec.Command(b.binaryPath, "umount", path)
	b.log.Debug("Command: ", cmd.String())

	out, err := cmd.CombinedOutput()
	if err != nil {
		b.log.Error("Error running unmount command: ", fmt.Errorf("%s: %s", out, err))
		return
	}
	b.log.Debug("Unmount finished", out)
	return b.getMountState(path)
}

func (b *BorgClient) UnmountRepository(repoId int) (state MountState, err error) {
	repo, err := b.GetRepository(repoId)
	if err != nil {
		return
	}

	path, err := getRepoMountPath(repo)
	if err != nil {
		return
	}

	return b.unmount(path)
}

func (b *BorgClient) UnmountArchive(archiveId int) (state MountState, err error) {
	archive, err := b.getArchive(archiveId)
	if err != nil {
		return
	}

	path, err := getArchiveMountPath(archive)
	if err != nil {
		return
	}

	return b.unmount(path)
}

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

func (b *BorgClient) getMountState(mountPath string) (state MountState, err error) {
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

func (b *BorgClient) GetRepositoryMountState(repoId int) (state MountState, err error) {
	repo, err := b.GetRepository(repoId)
	if err != nil {
		return
	}

	path, err := getRepoMountPath(repo)
	if err != nil {
		return
	}

	return b.getMountState(path)
}

func (b *BorgClient) GetArchiveMountStates(repoId int) (states map[int]*MountState, err error) {
	states = make(map[int]*MountState)
	repo, err := b.GetRepository(repoId)
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
