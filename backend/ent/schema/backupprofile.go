package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
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
		field.Strings("backup_paths").
			StructTag(`json:"backupPaths"`).Default([]string{}),
		field.Strings("exclude_paths").
			StructTag(`json:"excludePaths"`).Optional().Default([]string{}),
		field.Bool("is_setup_complete").
			StructTag(`json:"isSetupComplete"`).
			Default(false),
	}
}

// Edges of the BackupProfile.
func (BackupProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("repositories", Repository.Type),
		edge.To("archives", Archive.Type),
		edge.To("backup_schedule", BackupSchedule.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"backupSchedule,omitempty"`).
			Unique(),
		edge.From("failed_backup_runs", FailedBackupRun.Type).
			Ref("backup_profile"),
	}
}
