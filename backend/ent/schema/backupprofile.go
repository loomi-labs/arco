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
		field.String("name"),
		field.String("prefix"),
		field.String("directories"),
		field.Bool("hasPeriodicBackups").
			Default(false),
		field.Time("periodicBackupTime").
			Optional(),
	}
}

// Edges of the BackupProfile.
func (BackupProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("repositories", Repository.Type),
	}
}
