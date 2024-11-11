package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Settings holds the schema definition for the Settings entity.
type Settings struct {
	ent.Schema
}

// Fields of the Settings.
func (Settings) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("theme").
			Values("system", "light", "dark").
			Default("system"),
		field.Bool("show_welcome").
			StructTag(`json:"showWelcome"`).
			Default(true),
	}
}

// Edges of the Settings.
func (Settings) Edges() []ent.Edge {
	return nil
}
