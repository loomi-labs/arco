package main

import (
	"context"
	"go.uber.org/zap"
	"timebender/backend/borg"
)

// App struct
type App struct {
	ctx  context.Context
	Borg *borg.Borg
}

// NewApp creates a new App application struct
func NewApp(log *zap.SugaredLogger) *App {
	return &App{
		Borg: borg.NewBorg(log),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
