package borg

import (
	"context"
	"errors"
	"fmt"
	gocmd "github.com/go-cmd/cmd"
	"github.com/loomi-labs/arco/backend/borg/types"
	"github.com/negrel/assert"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strings"
	"time"
)

//go:generate mockgen -destination=mocks/borg.go -package=mocks . Borg,CommandRunner

type Borg interface {
	Info(ctx context.Context, repository, password string) (*types.InfoResponse, *types.Status)
	Init(ctx context.Context, repository, password string, noPassword bool) *types.Status
	List(ctx context.Context, repository string, password string, glob string) (*types.ListResponse, *types.Status)
	Compact(ctx context.Context, repository string, password string) *types.Status
	Create(ctx context.Context, repository, password, prefix string, backupPaths, excludePaths []string, ch chan types.BackupProgress) (string, *types.Status)
	Rename(ctx context.Context, repository, archive, password, newName string) *types.Status
	DeleteArchive(ctx context.Context, repository string, archive string, password string) *types.Status
	DeleteArchives(ctx context.Context, repository, password, prefix string) *types.Status
	DeleteRepository(ctx context.Context, repository string, password string) *types.Status
	MountRepository(ctx context.Context, repository string, password string, mountPath string) *types.Status
	MountArchive(ctx context.Context, repository string, archive string, password string, mountPath string) *types.Status
	Umount(ctx context.Context, path string) *types.Status
	Prune(ctx context.Context, repository string, password string, prefix string, pruneOptions []string, isDryRun bool, ch chan types.PruneResult) *types.Status
	BreakLock(ctx context.Context, repository string, password string) *types.Status
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

func (z *CmdLogger) LogCmdStatus(ctx context.Context, result *types.Status, cmd string, duration time.Duration) *types.Status {
	assert.NotNil(result, "LogCmdStatus received nil status")
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
	password           *string
	deleteConfirmation bool
	sshPrivateKeys     []string
}

func NewEnv(sshPrivateKeys []string) Env {
	return Env{
		sshPrivateKeys: sshPrivateKeys,
	}
}

func (e Env) WithPassword(password string) Env {
	e.password = &password
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
		sshOptions = append(sshOptions, fmt.Sprintf("-i%s", key))
	}

	env := append(
		os.Environ(),
		"BORG_EXIT_CODES=modern",
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	if e.password != nil {
		env = append(env, fmt.Sprintf("BORG_PASSPHRASE=%s", *e.password))
	}
	if e.deleteConfirmation {
		env = append(env, "BORG_DELETE_I_KNOW_WHAT_I_AM_DOING=YES")
	}
	return env
}

func createRuntimeError(err error) *types.BorgError {
	return &types.BorgError{
		ExitCode:   -1,
		Message:    err.Error(),
		Underlying: err,
		Category:   types.CategoryRuntime,
	}
}

func newStatusWithError(err error) *types.Status {
	return &types.Status{
		Error: createRuntimeError(err),
	}
}

func newStatusWithCanceled() *types.Status {
	return &types.Status{
		HasBeenCanceled: true,
	}
}

func toBorgResult(exitCode int, detail string) *types.Status {
	if exitCode == 0 {
		return &types.Status{}
	}
	if exitCode == 143 {
		return &types.Status{
			HasBeenCanceled: true,
		}
	}

	for _, warning := range types.AllBorgWarnings {
		if warning.ExitCode == exitCode {
			return &types.Status{Warning: warning}
		}
	}

	for _, err := range types.AllBorgErrors {
		if err.ExitCode == exitCode {
			switch err.ExitCode {
			case types.ErrorConnectionClosedWithHint.ExitCode, types.ErrorConnectionBrokenWithHint.ExitCode:
				return &types.Status{Error: &types.BorgError{
					ExitCode:   err.ExitCode,
					Message:    detail,
					Underlying: nil,
					Category:   err.Category,
				}}
			default:
				return &types.Status{Error: err}
			}
		}
	}

	return &types.Status{
		Error: &types.BorgError{
			ExitCode: exitCode,
			Message:  fmt.Sprintf("unknown borg error with exit code %d", exitCode),
			Category: types.CategoryUnknown,
		},
	}
}

// combinedOutputToStatus converts command output and error to a Status
func combinedOutputToStatus(out []byte, err error) *types.Status {
	if err == nil {
		return toBorgResult(0, "")
	}

	// Return the error if it is not an ExitError
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		// Include command output in the error message
		if len(out) > 0 {
			return newStatusWithError(fmt.Errorf("%s: %s", string(out), err))
		}
		return newStatusWithError(err)
	}

	return toBorgResult(exitError.ExitCode(), string(out))
}

// gocmdToStatus converts go-cmd status to a Status
func gocmdToStatus(status gocmd.Status, detail string) *types.Status {
	if status.Error != nil && status.Exit == 0 {
		// Execution error (command didn't run)
		return &types.Status{
			Error: createRuntimeError(status.Error),
		}
	}

	return toBorgResult(status.Exit, detail)
}
