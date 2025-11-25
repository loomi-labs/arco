package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
)

// Notification holds the schema definition for the Notification entity.
type Notification struct {
	ent.Schema
}

func (Notification) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

// Fields of the Notification.
func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("message").
			StructTag(`json:"message"`).
			Immutable(),
		field.Enum("type").
			StructTag(`json:"type"`).
			Values("failed_backup_run", "failed_pruning_run", "warning_backup_run", "warning_pruning_run", "failed_quick_check", "failed_full_check", "warning_quick_check", "warning_full_check").
			Immutable(),
		field.Bool("seen").
			StructTag(`json:"seen"`).
			Default(false),

		// TODO: This field can most likely be removed since it's never considered
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
