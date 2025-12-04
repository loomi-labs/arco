package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
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
		field.Bool("show_welcome").
			StructTag(`json:"showWelcome"`).
			Default(true),
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
	}
}

// Edges of the Settings.
func (Settings) Edges() []ent.Edge {
	return nil
}
