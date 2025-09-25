//go:build !integration

package types

import "context"

//go:generate mockgen -destination=mocks/event_emitter.go -package=mocks . EventEmitter

type EventEmitter interface {
	EmitEvent(ctx context.Context, event string, data ...string)
}
