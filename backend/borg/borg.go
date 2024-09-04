package borg

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"
)

type Borg struct {
	path string
	log  *CmdLogger
}

func NewBorg(path string, log *zap.SugaredLogger) *Borg {
	return &Borg{
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
