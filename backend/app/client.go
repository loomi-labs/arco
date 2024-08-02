package app

import (
	"arco/backend/app/types"
	"fmt"
	"go.uber.org/zap"
	"os"
)

func (a *AppClient) GetStartupError() types.Notification {
	var message string
	if a.state.StartupErr != nil {
		message = a.state.StartupErr.Error()
	}
	return types.Notification{
		Message: message,
		Level:   types.LevelError,
	}
}

func (a *AppClient) HandleError(msg string, fErr *types.FrontendError) {
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

func (a *AppClient) GetNotifications() []types.Notification {
	return a.state.GetAndDeleteNotifications()
}

type Env struct {
	Debug     bool   `json:"debug"`
	StartPage string `json:"startPage"`
}

func (a *AppClient) GetEnvVars() Env {
	return Env{
		Debug:     os.Getenv(EnvVarDebug.String()) == "true",
		StartPage: os.Getenv(EnvVarStartPage.String()),
	}
}
