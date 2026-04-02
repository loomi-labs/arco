package analytics

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	arcov1 "github.com/loomi-labs/arco/backend/api/v1"
	"github.com/loomi-labs/arco/backend/api/v1/arcov1connect"
	"github.com/loomi-labs/arco/backend/app/state"
	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/ent"
	"github.com/loomi-labs/arco/backend/ent/analyticsevent"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	flushInterval   = 24 * time.Hour
	flushBatchSize  = 500
	maxEventAge     = 14 * 24 * time.Hour
	flushRPCTimeout = 10 * time.Second
)

// Service contains the business logic and provides methods exposed to the frontend
type Service struct {
	log       *zap.SugaredLogger
	db        *ent.Client
	state     *state.State
	rpcClient arcov1connect.AnalyticsServiceClient

	// Cached opt-in state to avoid DB query on every TrackEvent call.
	// Access protected by enabledMu.
	enabledMu     sync.RWMutex
	enabledCached bool // whether enabledValue is valid
	enabledValue  *bool
}

// ServiceInternal provides backend-only methods that should not be exposed to frontend
type ServiceInternal struct {
	*Service
	flushMu  sync.Mutex
	flushWg  sync.WaitGroup
	stopFlush chan struct{}
}

// NewService creates a new analytics service
func NewService(log *zap.SugaredLogger, state *state.State) *ServiceInternal {
	return &ServiceInternal{
		Service: &Service{
			log:   log,
			state: state,
		},
	}
}

// Init initializes the service with database and RPC client
func (si *ServiceInternal) Init(db *ent.Client, rpcClient arcov1connect.AnalyticsServiceClient) {
	si.db = db
	si.rpcClient = rpcClient
}

func (s *Service) mustHaveDB() {
	if s.db == nil {
		panic("AnalyticsService: database client is nil")
	}
}

// IsUsageLoggingEnabled returns the opt-in state: nil = not yet asked, true = opted in, false = declined
func (s *Service) IsUsageLoggingEnabled(ctx context.Context) (*bool, error) {
	s.mustHaveDB()

	settings, err := s.db.Settings.Query().First(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	s.enabledMu.Lock()
	s.enabledCached = true
	s.enabledValue = settings.UsageLoggingEnabled
	s.enabledMu.Unlock()

	return settings.UsageLoggingEnabled, nil
}

// SetUsageLoggingEnabled updates the usage logging preference
func (s *Service) SetUsageLoggingEnabled(ctx context.Context, enabled bool) error {
	s.mustHaveDB()

	err := s.db.Settings.Update().
		SetUsageLoggingEnabled(enabled).
		Exec(ctx)
	if err != nil {
		s.log.Errorf("Failed to set usage logging enabled: %v", err)
		return err
	}

	if !enabled {
		if _, err := s.db.AnalyticsEvent.Delete().
			Where(analyticsevent.Sent(false)).
			Exec(ctx); err != nil {
			s.log.Errorf("Failed to clear queued analytics events: %v", err)
			return err
		}
	}

	// Update cache
	s.enabledMu.Lock()
	s.enabledCached = true
	s.enabledValue = &enabled
	s.enabledMu.Unlock()

	return nil
}

// isEnabled returns the cached opt-in state without querying the database.
// Falls back to a DB query if the cache is not yet populated.
func (s *Service) isEnabled(ctx context.Context) (bool, error) {
	s.enabledMu.RLock()
	if s.enabledCached {
		val := s.enabledValue
		s.enabledMu.RUnlock()
		return val != nil && *val, nil
	}
	s.enabledMu.RUnlock()

	// Cache miss — populate from DB
	enabled, err := s.IsUsageLoggingEnabled(ctx)
	if err != nil {
		return false, err
	}
	return enabled != nil && *enabled, nil
}

// TrackEvent records an analytics event if usage logging is enabled.
// Errors are logged internally and never returned to callers.
func (s *Service) TrackEvent(ctx context.Context, eventName EventName, properties map[string]string) {
	s.mustHaveDB()

	enabled, err := s.isEnabled(ctx)
	if err != nil {
		s.log.Errorf("Failed to check analytics opt-in: %v", err)
		return
	}
	if !enabled {
		return
	}

	builder := s.db.AnalyticsEvent.Create().
		SetEventName(string(eventName)).
		SetAppVersion(types.Version).
		SetOsInfo(runtime.GOOS).
		SetLocale(getUserLocale()).
		SetEventTime(time.Now())

	if properties != nil {
		builder = builder.SetEventProperties(properties)
	}

	if err = builder.Exec(ctx); err != nil {
		s.log.Errorf("Failed to track event %q: %v", eventName, err)
	}
}

// getInstallationID returns the installation UUID, generating one if it's the zero UUID
func (s *Service) getInstallationID(ctx context.Context) (uuid.UUID, error) {
	settings, err := s.db.Settings.Query().First(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}

	if settings.InstallationID == uuid.Nil {
		newID := uuid.New()
		err = s.db.Settings.Update().
			SetInstallationID(newID).
			Exec(ctx)
		if err != nil {
			return uuid.UUID{}, err
		}
		return newID, nil
	}

	return settings.InstallationID, nil
}

// StartFlushLoop runs a periodic flush of buffered events to the cloud.
// Call StopFlushLoop to stop the loop and wait for it to exit.
func (si *ServiceInternal) StartFlushLoop(ctx context.Context) {
	si.stopFlush = make(chan struct{})
	si.flushWg.Add(1)
	defer si.flushWg.Done()

	si.log.Debug("Starting analytics flush loop")

	// Initial flush on startup
	rpcCtx, cancel := context.WithTimeout(ctx, flushRPCTimeout)
	if err := si.flushEvents(rpcCtx); err != nil {
		si.log.Debugf("Initial analytics flush failed (will retry): %v", err)
	}
	cancel()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rpcCtx, cancel := context.WithTimeout(ctx, flushRPCTimeout)
			if err := si.flushEvents(rpcCtx); err != nil {
				si.log.Debugf("Analytics flush failed (will retry): %v", err)
			}
			cancel()
		case <-si.stopFlush:
			si.log.Debug("Analytics flush loop stopped")
			return
		case <-ctx.Done():
			si.log.Debug("Analytics flush loop stopped")
			return
		}
	}
}

// StopFlushLoop signals the flush loop to stop and waits for it to exit.
func (si *ServiceInternal) StopFlushLoop() {
	if si.stopFlush != nil {
		close(si.stopFlush)
	}
	si.flushWg.Wait()
}

// FlushAndShutdown performs a final flush before shutdown
func (si *ServiceInternal) FlushAndShutdown() {
	si.log.Debug("Flushing analytics events before shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := si.flushEvents(ctx); err != nil {
		si.log.Debugf("Final analytics flush failed: %v", err)
	}
}

// flushEvents sends unsent events to the cloud and removes them on success.
// Protected by flushMu to prevent concurrent flushes during shutdown.
func (si *ServiceInternal) flushEvents(ctx context.Context) error {
	si.flushMu.Lock()
	defer si.flushMu.Unlock()

	si.mustHaveDB()

	events, err := si.db.AnalyticsEvent.Query().
		Where(analyticsevent.Sent(false)).
		Limit(flushBatchSize).
		All(ctx)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	installationID, err := si.getInstallationID(ctx)
	if err != nil {
		return err
	}

	protoEvents := make([]*arcov1.AnalyticsEvent, len(events))
	for i, e := range events {
		protoEvents[i] = &arcov1.AnalyticsEvent{
			InstallationId:  installationID.String(),
			EventName:       e.EventName,
			EventProperties: e.EventProperties,
			AppVersion:      e.AppVersion,
			OsInfo:          e.OsInfo,
			Locale:          e.Locale,
			Timestamp:       timestamppb.New(e.EventTime),
		}
	}

	req := connect.NewRequest(&arcov1.IngestEventsRequest{
		Events: protoEvents,
	})

	_, err = si.rpcClient.IngestEvents(ctx, req)
	if err != nil {
		si.log.Debugf("Failed to send %d analytics events: %v", len(events), err)
		return err
	}

	// Delete sent events
	ids := make([]int, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}

	_, err = si.db.AnalyticsEvent.Delete().
		Where(analyticsevent.IDIn(ids...)).
		Exec(ctx)
	if err != nil {
		si.log.Errorf("Failed to delete sent analytics events: %v", err)
		return err
	}

	si.log.Debugf("Flushed %d analytics events to cloud", len(events))

	// Clean up old events that were never sent (e.g. cloud was unreachable)
	_, _ = si.db.AnalyticsEvent.Delete().
		Where(analyticsevent.EventTimeLT(time.Now().Add(-maxEventAge))).
		Exec(ctx)

	return nil
}

// getUserLocale returns the user's locale from environment variables (e.g. "en_US.UTF-8").
// Falls back to empty string if no locale is set.
func getUserLocale() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}
