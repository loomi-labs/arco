package util

import (
	"go.uber.org/zap"
)

type ZapLogWrapper struct {
	log *zap.SugaredLogger
}

func NewZapLogWrapper(logger *zap.SugaredLogger) *ZapLogWrapper {
	return &ZapLogWrapper{log: logger}
}

func (z *ZapLogWrapper) Print(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Info(message)
}

func (z *ZapLogWrapper) Trace(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Debug(message)
}

func (z *ZapLogWrapper) Debug(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Debug(message)
}

func (z *ZapLogWrapper) Info(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Info(message)
}

func (z *ZapLogWrapper) Warning(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Warn(message)
}

func (z *ZapLogWrapper) Error(message string) {
	z.log.WithOptions(zap.AddCallerSkip(1)).Errorw(message)
}

func (z *ZapLogWrapper) Fatal(message string) {
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
