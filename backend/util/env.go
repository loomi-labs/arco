package util

import (
	"fmt"
	"os"
	"strings"
)

type BorgEnv struct {
	password string
}

func (b BorgEnv) WithPassword(password string) BorgEnv {
	b.password = password
	return b
}

func (b BorgEnv) AsList() []string {
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
	if b.password != "" {
		env = append(env, fmt.Sprintf("BORG_PASSPHRASE=%s", b.password))
	}
	return env
}
