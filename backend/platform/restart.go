package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// buildRestartArgs returns the arguments for the restarted process: the given
// args with any existing --restart-delay stripped (to prevent accumulation)
// and a fresh --restart-delay appended. The delay ensures the old process has
// time to exit and release resources (e.g., single-instance lock) before the
// new process initializes.
func buildRestartArgs(args []string) []string {
	result := make([]string, 0, len(args)+2)
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--restart-delay" {
			// Skip this flag and its value
			if i+1 < len(args) {
				skipNext = true
			}
			continue
		}
		if strings.HasPrefix(arg, "--restart-delay=") {
			continue
		}
		result = append(result, arg)
	}
	return append(result, "--restart-delay", "1s")
}

// spawnRestart starts a new instance of the current process and then exits the
// current process.
func spawnRestart(execPath string, args []string) error {
	cmd := exec.Command(execPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot restart: start child process: %w", err)
	}

	os.Exit(0)
	return nil // unreachable
}
