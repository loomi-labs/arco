package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
)

// Notification holds the schema definition for the Notification entity.
type Notification struct {
	ent.Schema
}

// Fields of the Notification.
func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.Time("created_at").
			StructTag(`json:"createdAt"`).
			Default(time.Now()).
			Immutable(),
		field.String("message").
			StructTag(`json:"message"`).
			Immutable(),
		field.Enum("type").
			StructTag(`json:"type"`).
			Values("failed_backup_run", "failed_pruning_run").
			Immutable(),
		field.Bool("seen").
			StructTag(`json:"seen"`).
			Default(false),
		field.Enum("action").
			StructTag(`json:"action"`).
			Values("unlockRepository").
			Optional(),
	}
}

// Edges of the Notification.
func (Notification) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("backup_profile", BackupProfile.Type).
			StructTag(`json:"backupProfile,omitempty"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique().
			Immutable().
			Required(),
		edge.To("repository", Repository.Type).
			StructTag(`json:"repository,omitempty"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique().
			Immutable().
			Required(),
	}
}