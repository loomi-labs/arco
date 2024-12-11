package util

import (
	"github.com/wailsapp/wails/v2/pkg/logger"
	"go.uber.org/zap"
)

type zapLogWrapper struct {
	log *zap.SugaredLogger
}

func NewZapLogWrapper(logger *zap.SugaredLogger) logger.Logger {
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
	z.log.WithOptions(zap.AddCallerSkip(1)).Errorw(message)
}

func (z *zapLogWrapper) Fatal(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Fatalw(message)
}

type GooseLogger struct {
	log *zap.SugaredLogger
}

func NewGooseLogger(log *zap.SugaredLogger) *GooseLogger {
	return &GooseLogger{log: log}
}

func (g *GooseLogger) Fatalf(format string, v ...interface{}) {
	g.log.Fatalf(format, v...)
}

func (g *GooseLogger) Printf(format string, v ...interface{}) {
	g.log.Infof(format, v...)
}
