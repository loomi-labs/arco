package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
)

// AnalyticsEvent holds the schema definition for locally buffered analytics events.
type AnalyticsEvent struct {
	ent.Schema
}

func (AnalyticsEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

// Fields of the AnalyticsEvent.
func (AnalyticsEvent) Fields() []ent.Field {
	return []ent.Field{
		field.String("event_name").
			StructTag(`json:"eventName"`).
			NotEmpty(),
		field.JSON("event_properties", map[string]string{}).
			StructTag(`json:"eventProperties"`).
			Optional(),
		field.String("app_version").
			StructTag(`json:"appVersion"`),
		field.String("os_info").
			StructTag(`json:"osInfo"`),
		field.String("locale").
			StructTag(`json:"locale"`).
			Default(""),
		field.Time("event_time").
			StructTag(`json:"eventTime"`),
		field.Bool("sent").
			Default(false),
	}
}

// Edges of the AnalyticsEvent.
func (AnalyticsEvent) Edges() []ent.Edge {
	return nil
}

// Indexes of the AnalyticsEvent.
func (AnalyticsEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sent"),
	}
}
