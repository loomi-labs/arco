package app

import (
	"fmt"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"go.uber.org/zap"
)

func (a *AppClient) GetStartupState() state.StartupState {
	return a.state.GetStartupState()
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
		Debug:     EnvVarDebug.Bool(),
		StartPage: EnvVarStartPage.String(),
	}
}

func (a *AppClient) GetSettings() (*ent.Settings, error) {
	return a.db.Settings.Query().First(a.ctx)
}

func (a *AppClient) SaveSettings(settings *ent.Settings) error {
	a.log.Debugf("Saving settings: %s", settings)
	return a.db.Settings.
		Update().
		SetShowWelcome(settings.ShowWelcome).
		Exec(a.ctx)
}
