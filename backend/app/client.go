package app

import (
	"fmt"
	"go.uber.org/zap"
)

func (a *AppClient) GetStartupError() Notification {
	var message string
	if a.state.StartupErr != nil {
		message = a.state.StartupErr.Error()
	}
	return Notification{
		Message: message,
		Level:   LevelError,
	}
}

func (a *AppClient) HandleError(msg string, fErr *FrontendError) {
	errStr := ""
	if fErr != nil {
		if fErr.Message != "" && fErr.Stack != "" {
			errStr = fmt.Sprintf("%s\n%s", fErr.Message, fErr.Stack)
		} else if fErr.Message != "" {
			errStr = fErr.Message
		}
	}

	// We don't want to show the stack trace from the go code because the error comes from the frontend
	a.log.WithOptions(zap.AddCallerSkip(9999999)).
		Errorf(fmt.Sprintf("%s: %s", msg, errStr))
}

func (a *AppClient) GetNotifications() []Notification {
	return a.state.GetAndDeleteNofications()
}