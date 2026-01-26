package state

import (
	"context"
	"sync"

	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/app/types"
	"go.uber.org/zap"
)

type State struct {
	log           *zap.SugaredLogger
	mu            sync.RWMutex
	eventEmitter  types.EventEmitter
	notifications []types.Notification

	startupState    *StartupState
	authState       *AuthState
	checkoutSession *arcov1.CreateCheckoutSessionResponse
	checkoutResult  *CheckoutResult
	dirtyPageName   string // tracks which page has unsaved changes
}

type StartupStatus string

const (
	StartupStatusUnknown                StartupStatus = "unknown"
	StartupStatusCheckingForUpdates     StartupStatus = "checkingForUpdates"
	StartupStatusApplyingUpdates        StartupStatus = "applyingUpdates"
	StartupStatusRestartingArco         StartupStatus = "restartingArco"
	StartupStatusInitializingDatabase   StartupStatus = "initializingDatabase"
	StartupStatusCheckingForBorgUpdates StartupStatus = "checkingForBorgUpdates"
	StartupStatusUpdatingBorg           StartupStatus = "updatingBorg"
	StartupStatusInitializingApp        StartupStatus = "initializingApp"
	StartupStatusReady                  StartupStatus = "ready"
)

var AvailableStartupStatuses = []StartupStatus{
	StartupStatusUnknown,
	StartupStatusCheckingForUpdates,
	StartupStatusApplyingUpdates,
	StartupStatusRestartingArco,
	StartupStatusInitializingDatabase,
	StartupStatusCheckingForBorgUpdates,
	StartupStatusUpdatingBorg,
	StartupStatusInitializingApp,
	StartupStatusReady,
}

func (ss StartupStatus) String() string {
	return string(ss)
}

type StartupState struct {
	Error  string        `json:"error"`
	Status StartupStatus `json:"status"`
}

type AuthState struct {
	IsAuthenticated bool `json:"isAuthenticated"`
}

type CheckoutResultStatus string

const (
	CheckoutStatusPending   CheckoutResultStatus = "pending"
	CheckoutStatusCompleted CheckoutResultStatus = "completed"
	CheckoutStatusFailed    CheckoutResultStatus = "failed"
	CheckoutStatusTimeout   CheckoutResultStatus = "timeout"
)

type CheckoutResult struct {
	Status         CheckoutResultStatus `json:"status"`
	ErrorMessage   string               `json:"errorMessage,omitempty"`
	SubscriptionID string               `json:"subscriptionId,omitempty"`
}

func NewState(log *zap.SugaredLogger, eventEmitter types.EventEmitter) *State {
	return &State{
		log:           log,
		mu:            sync.RWMutex{},
		eventEmitter:  eventEmitter,
		notifications: []types.Notification{},

		startupState: &StartupState{
			Status: StartupStatusUnknown,
		},
		authState: &AuthState{
			IsAuthenticated: false,
		},
	}
}

/***********************************/
/********** Startup Error **********/
/***********************************/

func (s *State) SetStartupStatus(ctx context.Context, status StartupStatus, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventStartupStateChanged.String())

	s.startupState.Status = status
	if err != nil {
		s.startupState.Error = err.Error()
	}
	// We never clear the error, it's only set once since the app should not recover from a startup error
}

func (s *State) GetStartupState() StartupState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *s.startupState
}

/***********************************/
/********** Notifications **********/
/***********************************/

func (s *State) AddNotification(ctx context.Context, msg string, level types.NotificationLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventNotificationAvailable.String())

	s.notifications = append(s.notifications, types.Notification{
		Message: msg,
		Level:   level,
	})
}

func (s *State) GetAndDeleteNotifications() []types.Notification {
	s.mu.Lock()
	defer s.mu.Unlock()

	notifications := s.notifications
	s.notifications = []types.Notification{}
	return notifications
}

/***********************************/
/********** Auth State *************/
/***********************************/

func (s *State) SetAuthenticated(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.authState.IsAuthenticated {
		defer s.eventEmitter.EmitEvent(ctx, types.EventAuthStateChanged.String())
		s.authState.IsAuthenticated = true
	}
}

func (s *State) SetNotAuthenticated(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.authState.IsAuthenticated {
		defer s.eventEmitter.EmitEvent(ctx, types.EventAuthStateChanged.String())
		s.authState.IsAuthenticated = false
	}
}

func (s *State) GetAuthState() AuthState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *s.authState
}

/***********************************/
/******* Checkout & Subscription ***/
/***********************************/

// SetCheckoutSession stores the current checkout session and emits events
func (s *State) SetCheckoutSession(ctx context.Context, session *arcov1.CreateCheckoutSessionResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventCheckoutStateChangedString())
	s.checkoutSession = session
}

// GetCheckoutSession returns the current checkout session
func (s *State) GetCheckoutSession() *arcov1.CreateCheckoutSessionResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.checkoutSession
}

// ClearCheckoutSession clears the current checkout session and emits events
func (s *State) ClearCheckoutSession(ctx context.Context, result *CheckoutResult, emitSubscriptionEvent bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.eventEmitter.EmitEvent(ctx, types.EventCheckoutStateChangedString())
	if emitSubscriptionEvent {
		defer s.eventEmitter.EmitEvent(ctx, types.EventSubscriptionAddedString())
	}
	s.checkoutSession = nil
	s.checkoutResult = result
}

// GetCheckoutResult returns the current checkout result
func (s *State) GetCheckoutResult() *CheckoutResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.checkoutResult
}

// ClearCheckoutResult clears the current checkout result
func (s *State) ClearCheckoutResult() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkoutResult = nil
}

// EmitSubscriptionCancelled emits a subscription cancelled event
func (s *State) EmitSubscriptionCancelled(ctx context.Context) {
	s.eventEmitter.EmitEvent(ctx, types.EventSubscriptionCancelledString())
}

/***********************************/
/********** Dirty Page State *******/
/***********************************/

// SetDirtyPage marks a page as having unsaved changes
func (s *State) SetDirtyPage(pageName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirtyPageName = pageName
}

// ClearDirtyPage clears the dirty page state
func (s *State) ClearDirtyPage() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirtyPageName = ""
}

// IsDirty returns true if any page has unsaved changes
func (s *State) IsDirty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirtyPageName != ""
}
