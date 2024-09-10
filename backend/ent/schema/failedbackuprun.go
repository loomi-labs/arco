package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FailedBackupRun holds the schema definition for the FailedBackupRun entity.
type FailedBackupRun struct {
	ent.Schema
}

// Fields of the FailedBackupRun.
func (FailedBackupRun) Fields() []ent.Field {
	return []ent.Field{
		//field.Int("id").
		//	StructTag(`json:"id"`),
		//field.Int("exit_code").
		//	StructTag(`json:"exitCode"`),
		field.String("error").
			StructTag(`json:"error"`).
			Immutable(),
	}
}

// Edges of the FailedBackupRun.
func (FailedBackupRun) Edges() []ent.Edge {
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

// Indexes of the FailedBackupRun.
func (FailedBackupRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("backup_profile", "repository").
			Unique(),
	}
}
