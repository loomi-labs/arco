package util

import (
	"github.com/charmbracelet/keygen"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
)

func GenerateKeyPair() (*keygen.KeyPair, error) {
	kp, err := keygen.New(
		"~/sshtest/id_storage_test",
		keygen.WithKeyType(keygen.Ed25519),
		keygen.WithWrite(),
	)
	if err != nil {
		return nil, err
	}
	return kp, err
}

func SearchSSHKeys(log *zap.SugaredLogger) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Warnf("Failed to get home directory: %v", err)
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")
	files, err := os.ReadDir(sshDir)
	if err != nil {
		log.Warnf("Failed to read directory: %v", err)
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
