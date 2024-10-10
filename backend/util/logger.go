package util

import (
	"github.com/wailsapp/wails/v2/pkg/logger"
	"go.uber.org/zap"
)

type zapLogWrapper struct {
	log *zap.Logger
}

func NewZapLogWrapper(logger *zap.Logger) logger.Logger {
	return &zapLogWrapper{log: logger}
}

func (z *zapLogWrapper) Print(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Info(message)
}

func (z *zapLogWrapper) Trace(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Debug(message)
}

func (z *zapLogWrapper) Debug(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Debug(message)
}

func (z *zapLogWrapper) Info(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Info(message)
}

func (z *zapLogWrapper) Warning(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Warn(message)
}

func (z *zapLogWrapper) Error(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Error(message)
}

func (z *zapLogWrapper) Fatal(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Fatal(message)
}
