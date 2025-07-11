package borg

import (
	"context"
	"fmt"
	"github.com/loomi-labs/arco/backend/borg/types"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Borg interface {
	Info(ctx context.Context, repository, password string) (*types.InfoResponse, error)
	Init(ctx context.Context, repository, password string, noPassword bool) error
	List(ctx context.Context, repository string, password string) (*types.ListResponse, error)
	Compact(ctx context.Context, repository string, password string) error
	Create(ctx context.Context, repository, password, prefix string, backupPaths, excludePaths []string, ch chan types.BackupProgress) (string, *Status)
	Rename(ctx context.Context, repository, archive, password, newName string) error
	DeleteArchive(ctx context.Context, repository string, archive string, password string) error
	DeleteArchives(ctx context.Context, repository, password, prefix string) *Status
	DeleteRepository(ctx context.Context, repository string, password string) error
	MountRepository(ctx context.Context, repository string, password string, mountPath string) error
	MountArchive(ctx context.Context, repository string, archive string, password string, mountPath string) error
	Umount(ctx context.Context, path string) error
	Prune(ctx context.Context, repository string, password string, prefix string, pruneOptions []string, isDryRun bool, ch chan types.PruneResult) *Status
	BreakLock(ctx context.Context, repository string, password string) error
}

type borg struct {
	path           string
	log            *CmdLogger
	sshPrivateKeys []string
	commandRunner  CommandRunner
}

type CommandRunner interface {
	Info(cmd *exec.Cmd) ([]byte, error)
}

type commandRunner struct {
}

func NewBorg(path string, log *zap.SugaredLogger, sshPrivateKeys []string, cr CommandRunner) Borg {
	if cr == nil {
		cr = &commandRunner{}
	}
	return &borg{
		path:           path,
		log:            NewCmdLogger(log),
		sshPrivateKeys: sshPrivateKeys,
		commandRunner:  cr,
	}
}

const noErrorCtxKey = "noError"

// NoErrorCtx is a context value that can be used to suppress error logging for a specific command.
// This is useful when the error is expected and handled by the caller.
// The error will still be logged but at INFO level.
func NoErrorCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, noErrorCtxKey, true)
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
	z.LogCmdEndD(cmd, time.Since(startTime))
}

func (z *CmdLogger) LogCmdEndD(cmd string, duration time.Duration) {
	z.Infof("Finished command: `%s` in %s", cmd, duration)
}

func (z *CmdLogger) LogCmdError(ctx context.Context, cmd string, startTime time.Time, err error) error {
	return z.LogCmdErrorD(ctx, cmd, time.Since(startTime), err)
}

func (z *CmdLogger) LogCmdErrorD(ctx context.Context, cmd string, duration time.Duration, err error) error {
	//err = exitErrorToBorgResult(err)
	if ctx.Value(noErrorCtxKey) == nil {
		z.Errorf("Command `%s` failed after %s: %s", cmd, duration, err)
	} else {
		z.Infof("Command `%s` failed after %s: %s", cmd, duration, err)
	}
	return err
}

func (z *CmdLogger) LogCmdResultD(result *Status, cmd string, duration time.Duration) *Status {
	if result.HasError() {
		z.Errorf("Command `%s` failed after %s: %s", cmd, duration, result.Error)
	} else if result.HasWarning() {
		z.Infof("Command `%s` finished with warning after %s: %s", cmd, duration, result.Warning)
	} else if result.HasBeenCanceled {
		z.Infof("Command `%s` cancelled after %s", cmd, duration)
	} else {
		z.Infof("Command `%s` finished after %s", cmd, duration)
	}
	return result
}

func (z *CmdLogger) LogCmdCancelled(cmd string, startTime time.Time) {
	z.LogCmdCancelledD(cmd, time.Since(startTime))
}

func (z *CmdLogger) LogCmdCancelledD(cmd string, duration time.Duration) {
	z.Infof("Command `%s` cancelled after %s", cmd, duration)
}

type Env struct {
	password           string
	deleteConfirmation bool
	sshPrivateKeys     []string
}

func NewEnv(sshPrivateKeys []string) Env {
	return Env{
		sshPrivateKeys: sshPrivateKeys,
	}
}

func (e Env) WithPassword(password string) Env {
	e.password = password
	return e
}

func (e Env) WithDeleteConfirmation() Env {
	e.deleteConfirmation = true
	return e
}

func (e Env) AsList() []string {
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-oConnectTimeout=10",
	}
	for _, key := range e.sshPrivateKeys {
		sshOptions = append(sshOptions, fmt.Sprintf("-i %s", key))
	}

	env := append(
		os.Environ(),
		"BORG_EXIT_CODES=modern",
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	if e.password != "" {
		env = append(env, fmt.Sprintf("BORG_PASSPHRASE=%s", e.password))
	}
	if e.deleteConfirmation {
		env = append(env, "BORG_DELETE_I_KNOW_WHAT_I_AM_DOING=YES")
	}
	return env
}
