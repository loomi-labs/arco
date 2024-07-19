package util

import (
	"github.com/charmbracelet/keygen"
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
