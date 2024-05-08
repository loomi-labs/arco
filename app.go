package main

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"timebender/backend/borg"
)

// App struct
type App struct {
	ctx  context.Context
	Borg *borg.Borg
}

// NewApp creates a new App application struct
func NewApp(log logger.Logger) *App {
	return &App{
		Borg: borg.NewBorg(log),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
