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
	z.Infof("Running command: %s", cmd)
	return time.Now()
}

func (z *CmdLogger) LogCmdEnd(cmd string, startTime time.Time) {
	z.Infof("Finished command: %s in %s", cmd, time.Since(startTime))
}

func (z *CmdLogger) LogCmdError(cmd string, startTime time.Time, err error) error {
	z.Errorf("Command %s failed after %s: %s", cmd, time.Since(startTime), err)
	return err
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
		"-i ~/sshtest/id_storage_test",
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
