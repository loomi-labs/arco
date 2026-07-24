package platform

import (
	"fmt"
	"os"
	"syscall"
)

// RestartSelf replaces the current process image with a new instance via exec.
// Keeping the same PID is required when running under systemd: with spawn+exit
// the unit's main process exits, systemd marks the service inactive and kills
// the spawned child with the rest of the cgroup, so the app never comes back
// up after a self-update.
func RestartSelf() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot restart: resolve executable: %w", err)
	}

	argv := append([]string{execPath}, buildRestartArgs(os.Args[1:])...)
	if err := syscall.Exec(execPath, argv, os.Environ()); err != nil {
		return fmt.Errorf("cannot restart: exec: %w", err)
	}
	return nil // unreachable: exec only returns on error
}
