package utils

import (
	"bytes"
	"go.uber.org/zap"
)

// SanitizeOutput removes all lines before the first line that starts with '{'
// This is needed because borg commands with SSH can output warnings to stderr
// that get mixed with the JSON output when using CombinedOutput()
// The removed content is logged as a warning for debugging purposes
func SanitizeOutput(out []byte, logger *zap.SugaredLogger) []byte {
	out = bytes.TrimSpace(out)

	// Nothing to sanitize
	if bytes.HasPrefix(out, []byte("{")) {
		return out
	}

	// Split the output into lines and find the first line that starts with '{'
	lines := bytes.Split(out, []byte("\n"))
	for i, line := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("{")) {
			// Log the content that will be removed
			if i > 0 {
				removedContent := bytes.Join(lines[:i], []byte("\n"))
				logger.Warnf("Sanitized SSH output before JSON parsing: %s", string(removedContent))
			}
			return bytes.Join(lines[i:], []byte("\n"))
		}
	}
	return out
}