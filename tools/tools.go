//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/wailsapp/wails/v2/cmd/wails"
	_ "go.uber.org/mock/mockgen"
)
