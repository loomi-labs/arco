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
	Info(ctx context.Context, repository, password string) (*types.InfoResponse, *Status)
	Init(ctx context.Context, repository, password string, noPassword bool) *Status
	List(ctx context.Context, repository string, password string) (*types.ListResponse, *Status)
	Compact(ctx context.Context, repository string, password string) *Status
	Create(ctx context.Context, repository, password, prefix string, backupPaths, excludePaths []string, ch chan types.BackupProgress) (string, *Status)
	Rename(ctx context.Context, repository, archive, password, newName string) *Status
	DeleteArchive(ctx context.Context, repository string, archive string, password string) *Status
	DeleteArchives(ctx context.Context, repository, password, prefix string) *Status
	DeleteRepository(ctx context.Context, repository string, password string) *Status
	MountRepository(ctx context.Context, repository string, password string, mountPath string) *Status
	MountArchive(ctx context.Context, repository string, archive string, password string, mountPath string) *Status
	Umount(ctx context.Context, path string) *Status
	Prune(ctx context.Context, repository string, password string, prefix string, pruneOptions []string, isDryRun bool, ch chan types.PruneResult) *Status
	BreakLock(ctx context.Context, repository string, password string) *Status
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

func (z *CmdLogger) LogCmdResult(ctx context.Context, result *Status, cmd string, duration time.Duration) *Status {
	if result.HasError() {
		if ctx.Value(noErrorCtxKey) == nil {
			z.Errorf("Command `%s` failed after %s: %s", cmd, duration, result.Error)
		} else {
			z.Infof("Command `%s` failed after %s: %s", cmd, duration, result.Error)
		}
	} else if result.HasWarning() {
		z.Infof("Command `%s` finished with warning after %s: %s", cmd, duration, result.Warning)
	} else if result.HasBeenCanceled {
		z.Infof("Command `%s` cancelled after %s", cmd, duration)
	} else {
		z.Infof("Command `%s` finished after %s", cmd, duration)
	}
	return result
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
