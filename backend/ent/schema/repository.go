package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Repository holds the schema definition for the Repository entity.
type Repository struct {
	ent.Schema
}

// Fields of the Repository.
func (Repository) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("name").
			StructTag(`json:"name"`),
		field.String("location").
			StructTag(`json:"location"`).
			Unique(),
		field.String("password").
			StructTag(`json:"password"`),

		// Integrity check
		field.Time("next_integrity_check").
			StructTag(`json:"nextIntegrityCheck"`).
			Nillable().
			Optional(),

		// Stats
		field.Int("stats_total_chunks").
			Default(0).
			StructTag(`json:"statsTotalChunks"`),
		field.Int("stats_total_size").
			Default(0).
			StructTag(`json:"statsTotalSize"`),
		field.Int("stats_total_csize").
			Default(0).
			StructTag(`json:"statsTotalCsize"`),
		field.Int("stats_total_unique_chunks").
			Default(0).
			StructTag(`json:"statsTotalUniqueChunks"`),
		field.Int("stats_unique_size").
			Default(0).
			StructTag(`json:"statsUniqueSize"`),
		field.Int("stats_unique_csize").
			Default(0).
			StructTag(`json:"statsUniqueCsize"`),
	}
}

// Edges of the Repository.
func (Repository) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("backup_profiles", BackupProfile.Type).
			StructTag(`json:"backupProfiles,omitempty"`).
			Ref("repositories"),
		edge.From("archives", Archive.Type).
			Ref("repository"),
		edge.From("notifications", Notification.Type).
			StructTag(`json:"notifications,omitempty"`).
			Ref("repository"),
	}
}
