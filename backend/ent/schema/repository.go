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

		// Stats
		field.Int("stats_total_chunks").
			Default(0).
			StructTag(`json:"stats_total_chunks"`),
		field.Int("stats_total_size").
			Default(0).
			StructTag(`json:"stats_total_size"`),
		field.Int("stats_total_csize").
			Default(0).
			StructTag(`json:"stats_total_csize"`),
		field.Int("stats_total_unique_chunks").
			Default(0).
			StructTag(`json:"stats_total_unique_chunks"`),
		field.Int("stats_unique_size").
			Default(0).
			StructTag(`json:"stats_unique_size"`),
		field.Int("stats_unique_csize").
			Default(0).
			StructTag(`json:"stats_unique_csize"`),
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
		edge.From("failed_backup_runs", FailedBackupRun.Type).
			StructTag(`json:"failedBackupRuns,omitempty"`).
			Ref("repository"),
	}
}
