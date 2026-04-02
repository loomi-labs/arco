package analytics

import "context"

// Tracker is the interface used by other services to record analytics events.
//
//go:generate mockgen -destination=mocks/tracker.go -package=mocks . Tracker
type Tracker interface {
	TrackEvent(ctx context.Context, eventName EventName, properties map[string]string)
}

// NoopTracker is a Tracker that does nothing. Useful for tests.
type NoopTracker struct{}

func (NoopTracker) TrackEvent(_ context.Context, _ EventName, _ map[string]string) {}
