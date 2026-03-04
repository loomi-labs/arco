package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RestartSelf spawns a new instance of the current process with a restart delay
// and then exits the current process. The delay ensures the old process has time
// to exit and release resources (e.g., single-instance lock) before the new
// process initializes.
func RestartSelf() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot restart: resolve executable: %w", err)
	}

	// Build args, stripping any existing --restart-delay to prevent accumulation
	args := make([]string, 0, len(os.Args))
	skipNext := false
	for i, arg := range os.Args[1:] {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--restart-delay" {
			// Skip this flag and its value
			if i+1 < len(os.Args)-1 {
				skipNext = true
			}
			continue
		}
		if strings.HasPrefix(arg, "--restart-delay=") {
			continue
		}
		args = append(args, arg)
	}
	args = append(args, "--restart-delay", "1s")

	cmd := exec.Command(execPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot restart: start child process: %w", err)
	}

	os.Exit(0)
	return nil // unreachable
}
