package borg

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTeamKeeper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "borg test suite")
}
