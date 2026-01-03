package util

import (
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
)

func searchSSHKeysInDir(log *zap.SugaredLogger, sshDir string) []string {
	files, err := os.ReadDir(sshDir)
	if err != nil {
		log.Infof("Failed to read directory %s: %v", sshDir, err)
		return nil
	}

	var keys []string
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(sshDir, file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			_, err = ssh.ParseRawPrivateKey(content)
			if err != nil {
				continue
			}
			keys = append(keys, filePath)
			log.Debugf("Found SSH key: %s", filePath)
		}
	}
	return keys
}

func searchSSHKeysInHomeDir(log *zap.SugaredLogger) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Warnf("Failed to get home directory: %v", err)
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")
	return searchSSHKeysInDir(log, sshDir)
}

func SearchSSHKeys(log *zap.SugaredLogger, sshDir string) []string {
	var allKeys []string

	// Search in arco ssh directory FIRST (prioritize ArcoCloud key)
	allKeys = append(allKeys, searchSSHKeysInDir(log, sshDir)...)

	// Then search in home directory
	homeKeys := searchSSHKeysInHomeDir(log)
	allKeys = append(allKeys, homeKeys...)

	// Remove duplicates (in case same key exists in both locations)
	uniqueKeys := make([]string, 0, len(allKeys))
	seen := make(map[string]bool)
	for _, key := range allKeys {
		if !seen[key] {
			seen[key] = true
			uniqueKeys = append(uniqueKeys, key)
		}
	}

	return uniqueKeys
}
