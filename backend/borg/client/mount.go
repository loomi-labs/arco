package client

import (
	"arco/backend/borg/util"
	"arco/backend/ent"
	"fmt"
	"github.com/prometheus/procfs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
)

func getMountBasePath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join("/run/user", currentUser.Uid, "arco"), nil
}

func getMountPath(name string) (string, error) {
	basePath, err := getMountBasePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(basePath, name), nil
}

func createAndGetMountPath(name string) (string, error) {
	runUserPath, err := getMountPath(name)
	if err != nil {
		return "", err
	}

	// Check if the directory exists
	if _, err := os.Stat(runUserPath); os.IsNotExist(err) {
		//Create the directory
		return runUserPath, os.MkdirAll(runUserPath, 0755)
	}
	return runUserPath, nil
}

func (b *BorgClient) MountRepository(repoId int) (state MountState, err error) {
	repo, err := b.GetRepository(repoId)
	if err != nil {
		return
	}

	path, err := createAndGetMountPath(strconv.Itoa(repo.ID))
	if err != nil {
		return
	}

	cmd := exec.Command(b.binaryPath, "mount", repo.URL, path)
	cmd.Env = util.BorgEnv{}.WithPassword(repo.Password).AsList()
	b.log.Debug("Command: ", cmd.String())

	// Run backup command
	out, err := cmd.CombinedOutput()
	if err != nil {
		b.log.Error("Error running backup command: ", fmt.Errorf("%s: %s", out, err))
		return
	}
	b.log.Debug("Backup job finished", out)
	return b.getMountState(repo)
}

func (b *BorgClient) UnmountRepository(repoId int) (state MountState, err error) {
	repo, err := b.GetRepository(repoId)
	if err != nil {
		return
	}

	path, err := getMountPath(strconv.Itoa(repo.ID))
	if err != nil {
		return
	}

	cmd := exec.Command(b.binaryPath, "umount", path)
	b.log.Debug("Command: ", cmd.String())

	out, err := cmd.CombinedOutput()
	if err != nil {
		b.log.Error("Error running unmount command: ", fmt.Errorf("%s: %s", out, err))
		return
	}
	b.log.Debug("Unmount finished", out)
	return b.getMountState(repo)
}

func (b *BorgClient) isRepositoryMounted(repo *ent.Repository) (isMounted bool, mountPath string, err error) {
	mountPath, err = getMountPath(strconv.Itoa(repo.ID))
	if err != nil {
		return
	}

	mounts, err := procfs.GetMounts()
	if err != nil {
		return
	}

	// Check if the repository is mounted
	for _, mount := range mounts {
		if mount.MountPoint == mountPath {
			isMounted = true
			break
		}
	}
	return
}

func (b *BorgClient) getMountState(repo *ent.Repository) (state MountState, err error) {
	isMounted, mountPath, err := b.isRepositoryMounted(repo)
	if err != nil {
		return
	}
	state.IsMounted = isMounted
	state.MountPath = mountPath
	return
}

func (b *BorgClient) GetMountState(repoId int) (state MountState, err error) {
	repo, err := b.GetRepository(repoId)
	if err != nil {
		return
	}
	return b.getMountState(repo)
}
