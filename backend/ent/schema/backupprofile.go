package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// BackupProfile holds the schema definition for the BackupProfile entity.
type BackupProfile struct {
	ent.Schema
}

// Fields of the BackupProfile.
func (BackupProfile) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("name").
			StructTag(`json:"name"`),
		field.String("prefix").
			StructTag(`json:"prefix"`),
		field.Strings("directories").
			StructTag(`json:"directories"`),
		field.Bool("is_setup_complete").
			StructTag(`json:"isSetupComplete"`).
			Default(false),
	}
}

// Edges of the BackupProfile.
func (BackupProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("repositories", Repository.Type),
		edge.To("backup_schedule", BackupSchedule.Type).
			Unique(),
	}
}
