package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
)

// Settings holds the schema definition for the Settings entity.
type Settings struct {
	ent.Schema
}

func (Settings) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

// Fields of the Settings.
func (Settings) Fields() []ent.Field {
	return []ent.Field{
		field.Bool("expert_mode").
			StructTag(`json:"expertMode"`).
			Default(false),
		field.Enum("theme").
			Values("light", "dark", "system").
			Default("system"),
		field.Bool("disable_transitions").
			StructTag(`json:"disableTransitions"`).
			Default(false),
		field.Bool("disable_shadows").
			StructTag(`json:"disableShadows"`).
			Default(false),
		field.Bool("macfuse_warning_dismissed").
			StructTag(`json:"macfuseWarningDismissed"`).
			Default(false),
		field.Bool("full_disk_access_warning_dismissed").
			StructTag(`json:"fullDiskAccessWarningDismissed"`).
			Default(false),
		field.Time("feedback_last_prompted_at").
			StructTag(`json:"feedbackLastPromptedAt"`).
			Optional().
			Nillable(),
		field.Bool("usage_logging_enabled").
			StructTag(`json:"usageLoggingEnabled"`).
			Optional().
			Nillable(),
		field.UUID("installation_id", uuid.UUID{}).
			StructTag(`json:"installationId"`).
			Default(uuid.New),
		field.Int("font_scale").
			StructTag(`json:"fontScale"`).
			Default(100).
			Min(80).
			Max(150),
		field.Bool("high_contrast").
			StructTag(`json:"highContrast"`).
			Default(false),
	}
}

// Edges of the Settings.
func (Settings) Edges() []ent.Edge {
	return nil
}
