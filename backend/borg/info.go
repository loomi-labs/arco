package borg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os/exec"
)

func (cr *commandRunner) Info(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

// sanitizeOutput removes all lines before the first line that starts with '{'
func sanitizeOutput(out []byte) []byte {
	out = bytes.TrimSpace(out)

	// Nothing to sanitize
	if bytes.HasPrefix(out, []byte("{")) {
		return out
	}

	// Split the output into lines and find the first line that starts with '{'
	lines := bytes.Split(out, []byte("\n"))
	for i, line := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("{")) {
			return bytes.Join(lines[i:], []byte("\n"))
		}
	}
	return out
}

func (b *borg) Info(ctx context.Context, repository, password string) (*types.InfoResponse, error) {
	cmd := exec.CommandContext(ctx, b.path, "info", "--json", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Check if we can connect to the repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := b.commandRunner.Info(cmd)
	if err != nil {
		return nil, b.log.LogCmdError(ctx, cmd.String(), startTime, err)
	}

	var info types.InfoResponse
	err = json.Unmarshal(sanitizeOutput(out), &info)
	if err != nil {
		return nil, b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("failed to parse borg info output: %w", err))
	}

	b.log.LogCmdEnd(cmd.String(), startTime)
	return &info, nil
}
