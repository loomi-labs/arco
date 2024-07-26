package borg

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBorg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "borg test suite")
}
