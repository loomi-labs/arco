package borg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gocmd "github.com/go-cmd/cmd"
	"github.com/loomi-labs/arco/backend/borg/types"
)

// Check runs the borg check command to verify repository integrity
func (b *borg) Check(ctx context.Context, repository, password string, quick bool) *types.CheckResult {
	cmdStr := []string{"check"}

	if quick {
		cmdStr = append(cmdStr, "--repository-only")
	} else {
		cmdStr = append(cmdStr, "--verify-data")
	}

	cmdStr = append(cmdStr, "--log-json", repository)

	// Standard execution pattern (same as other borg commands)
	options := gocmd.Options{Buffered: false, Streaming: true}
	cmd := gocmd.NewCmdOptions(options, b.path, cmdStr...)
	cmdLog := fmt.Sprintf("%s %s", b.path, strings.Join(cmdStr, " "))
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	b.log.LogCmdStart(cmdLog)
	statusChan := cmd.Start()

	// Start log decoder goroutine
	logsChan := make(chan types.LogMessage)
	go decodeCheckLogs(cmd, logsChan)

	// Collect log messages during execution
	var logs []types.LogMessage

	select {
	case <-ctx.Done():
		err := cmd.Stop()
		if err != nil {
			b.log.Errorf("error stopping command: %v", err)
		}
		_ = <-statusChan
		// Drain remaining error logs
		for log := range logsChan {
			logs = append(logs, log)
		}
		borgStatus := newStatusWithCanceled()
		return &types.CheckResult{
			Status:    b.log.LogCmdStatus(ctx, borgStatus, cmdLog, time.Duration(cmd.Status().Runtime)),
			ErrorLogs: logs,
		}
	case _ = <-statusChan:
		// Drain remaining error logs after command completes
		for log := range logsChan {
			logs = append(logs, log)
		}
		break
	}

	status := cmd.Status()

	// If we have error logs AND exit code is 1, hack it to 0
	// This treats "check found issues" as success (command ran successfully)
	// The actual errors are captured in ErrorLogs for later processing
	if len(logs) > 0 && status.Exit == 1 {
		status.Exit = 0
	}

	borgStatus := gocmdToStatus(status, "")
	return &types.CheckResult{
		Status:    b.log.LogCmdStatus(ctx, borgStatus, cmdLog, time.Duration(status.Runtime)),
		ErrorLogs: logs,
	}
}

// decodeCheckLogs parses JSON log messages from borg check stderr output
// It filters for ERROR level messages only (no warnings)
func decodeCheckLogs(cmd *gocmd.Cmd, ch chan<- types.LogMessage) {
	defer close(ch)
	for {
		select {
		case _ = <-cmd.Stdout:
			// Ignore stdout (borg outputs JSON to stderr)
		case data := <-cmd.Stderr:
			// Parse type discriminator first
			var typeMsg types.Type
			if err := json.Unmarshal([]byte(data), &typeMsg); err != nil {
				continue // Skip unparseable messages
			}

			// Filter to LogMessageType only
			if types.JSONType(typeMsg.Type) != types.LogMessageType {
				continue
			}

			// Parse to LogMessage struct
			var logMsg types.LogMessage
			if err := json.Unmarshal([]byte(data), &logMsg); err != nil {
				continue
			}

			// Filter ERROR/CRITICAL level messages only (no warnings)
			level := logMsg.LevelName
			if level == "ERROR" {
				ch <- logMsg
			}
		case <-cmd.Done():
			return
		}
	}
}
