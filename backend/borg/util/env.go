package util

import (
	"fmt"
	"os"
	"strings"
)

func CreateEnv(password string) []string {
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-i ~/sshtest/id_storage_test",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", password),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	return env
}

func GetEnv() []string {
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-i ~/sshtest/id_storage_test",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	return env
}
