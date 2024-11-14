package borg

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"
)

type Borg interface {
	Compact(ctx context.Context, repoUrl string, repoPassword string) error
	Create(ctx context.Context, repoUrl, password, prefix string, backupPaths, excludePaths []string, ch chan BackupProgress) (string, error)
	DeleteArchive(ctx context.Context, repository string, archive string, password string) error
	DeleteArchives(ctx context.Context, repoUrl, password, prefix string) error
	Info(url, password string) (*InfoResponse, error)
	Init(url, password string, noPassword bool) error
	List(repoUrl string, password string) (*ListResponse, error)
	MountRepository(repoUrl string, password string, mountPath string) error
	MountArchive(repoUrl string, archive string, password string, mountPath string) error
	Prune(ctx context.Context, repoUrl string, password string, prefix string, pruneOptions []string, isDryRun bool, ch chan PruneResult) error
	BreakLock(ctx context.Context, repository string, password string) error
	Umount(path string) error
	Rename(ctx context.Context, repository, archive, password, newName string) error
}

type borg struct {
	path string
	log  *CmdLogger
}

func NewBorg(path string, log *zap.SugaredLogger) Borg {
	return &borg{
		path: path,
		log:  NewCmdLogger(log),
	}
}

type CmdLogger struct {
	*zap.SugaredLogger
}

func NewCmdLogger(log *zap.SugaredLogger) *CmdLogger {
	return &CmdLogger{log}
}

func (z *CmdLogger) LogCmdStart(cmd string) time.Time {
	z.Infof("Running command: `%s`", cmd)
	return time.Now()
}

func (z *CmdLogger) LogCmdEnd(cmd string, startTime time.Time) {
	z.Infof("Finished command: `%s` in %s", cmd, time.Since(startTime))
}

func (z *CmdLogger) LogCmdError(cmd string, startTime time.Time, err error) error {
	err = exitCodesToError(err)
	z.Errorf("Command `%s` failed after %s: %s", cmd, time.Since(startTime), err)
	return err
}

func (z *CmdLogger) LogCmdCancelled(cmd string, startTime time.Time) {
	z.Infof("Command `%s` cancelled after %s", cmd, time.Since(startTime))
}

type Env struct {
	password string
}

func (e Env) WithPassword(password string) Env {
	e.password = password
	return e
}

func (e Env) AsList() []string {
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-oConnectTimeout=10",
		"-i ~/.config/arco/id_rsa",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
		"BORG_EXIT_CODES=modern",
	)
	if e.password != "" {
		env = append(env, fmt.Sprintf("BORG_PASSPHRASE=%s", e.password))
	}
	return env
}
