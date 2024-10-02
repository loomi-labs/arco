package borg

import (
	"fmt"
	"os/exec"
)

func (b *borg) MountRepository(repoUrl string, password string, mountPath string) error {
	return b.mount(repoUrl, nil, password, mountPath)
}

func (b *borg) MountArchive(repoUrl string, archive string, password string, mountPath string) error {
	return b.mount(repoUrl, &archive, password, mountPath)
}

func (b *borg) mount(repoUrl string, archive *string, password string, mountPath string) error {
	archiveOrRepo := repoUrl
	if archive != nil {
		archiveOrRepo = fmt.Sprintf("%s::%s", repoUrl, *archive)
	}

	cmd := exec.Command(b.path, "mount", archiveOrRepo, mountPath)
	cmd.Env = Env{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}

func (b *borg) Umount(path string) error {
	cmd := exec.Command(b.path, "umount", path)

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
