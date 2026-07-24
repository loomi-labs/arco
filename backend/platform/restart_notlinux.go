//go:build !linux

package platform

import (
	"fmt"
	"os"
)

// RestartSelf spawns a new instance of the current process with a restart delay
// and then exits the current process. Exec is not used here because replacing
// the process image breaks the macOS systray (see #280).
func RestartSelf() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot restart: resolve executable: %w", err)
	}

	return spawnRestart(execPath, buildRestartArgs(os.Args[1:]))
}
