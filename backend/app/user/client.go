package user

import (
	"context"
	"fmt"

	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
)

// Service contains the business logic and provides methods exposed to the frontend
type Service struct {
	log          *zap.SugaredLogger
	db           *ent.Client
	state        *state.State
	eventEmitter types.EventEmitter
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
}

// NewService creates a new user service
func NewService(log *zap.SugaredLogger, state *state.State) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:   log,
			state: state,
		},
	}
}

// Init initializes the service with remaining dependencies
func (si *ServiceInternal) Init(db *ent.Client, eventEmitter types.EventEmitter) {
	si.db = db
	si.eventEmitter = eventEmitter
}

// mustHaveDB panics if db is nil. This is a programming error guard.
func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("UserService: database client is nil")
	}
}

func (s *Service) GetStartupState(ctx context.Context) state.StartupState {
	return s.state.GetStartupState()
}

func (s *Service) HandleError(ctx context.Context, msg string, fErr *types.FrontendError) {
	errStr := ""
	if fErr != nil {
		if fErr.Message != "" && fErr.Stack != "" {
			errStr = fmt.Sprintf("%s\n%s", fErr.Message, fErr.Stack)
		} else if fErr.Message != "" {
			errStr = fErr.Message
		}
	}

	// We don't want to show the stack trace from the go code because the error comes from the frontend
	s.log.WithOptions(zap.AddCallerSkip(9999999)).
		Errorf("%s: %s", msg, errStr)
}

func (s *Service) GetNotifications(ctx context.Context) []types.Notification {
	return s.state.GetAndDeleteNotifications()
}

type Env struct {
	Debug            bool   `json:"debug"`
	StartPage        string `json:"startPage"`
	LoginBetaEnabled bool   `json:"loginBetaEnabled"`
}

func (s *Service) GetEnvVars(ctx context.Context) Env {
	return Env{
		Debug:            types.EnvVarDebug.Bool(),
		StartPage:        types.EnvVarStartPage.String(),
		LoginBetaEnabled: types.EnvVarEnableLoginBeta.Bool(),
	}
}

func (s *Service) GetSettings(ctx context.Context) (*ent.Settings, error) {
	s.mustHaveDB()
	return s.db.Settings.Query().First(ctx)
}

func (s *Service) SaveSettings(ctx context.Context, settings *ent.Settings) error {
	s.mustHaveDB()
	s.log.Debugf("Saving settings: %s", settings)
	err := s.db.Settings.
		Update().
		SetShowWelcome(settings.ShowWelcome).
		SetExpertMode(settings.ExpertMode).
		SetTheme(settings.Theme).
		Exec(ctx)
	if err != nil {
		return err
	}

	s.eventEmitter.EmitEvent(application.Get().Context(), types.EventSettingsChangedString())
	return nil
}

type AppInfo struct {
	Version     string `json:"version"`
	WebsiteURL  string `json:"websiteUrl"`
	GithubURL   string `json:"githubUrl"`
	Description string `json:"description"`
}

func (s *Service) GetAppInfo(ctx context.Context) AppInfo {
	return AppInfo{
		Version:     types.Version,
		WebsiteURL:  "https://arco-backup.com",
		GithubURL:   "https://github.com/loomi-labs/arco",
		Description: "Arco is a modern, user-friendly backup tool powered by Borg Backup.",
	}
}

func (s *Service) GetAllEvents(ctx context.Context) []types.Event {
	return types.AllEvents
}

type User struct {
	Email string `json:"email"`
}

func (s *Service) GetUser(ctx context.Context) (*User, error) {
	s.mustHaveDB()
	entUser, err := s.db.User.Query().First(ctx)
	if err != nil {
		return nil, err
	}

	return &User{
		Email: entUser.Email,
	}, nil
}

func (s *Service) LogDebug(ctx context.Context, message string) {
	s.log.Debug(message)
}
