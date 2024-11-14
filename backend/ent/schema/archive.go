package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Archive holds the schema definition for the Archive entity.
type Archive struct {
	ent.Schema
}

// Fields of the Archive.
func (Archive) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("name").
			StructTag(`json:"name"`),
		field.Time("createdAt").
			StructTag(`json:"createdAt"`),
		// Duration is the time it took to create the archive in nanoseconds.
		field.Float("duration").
			StructTag(`json:"duration"`),
		field.String("borg_id").
			StructTag(`json:"borgId"`),
		field.Bool("will_be_pruned").
			StructTag(`json:"willBePruned"`).
			Default(false),
	}
}

// Edges of the Archive.
func (Archive) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("repository", Repository.Type).
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("backup_profile", BackupProfile.Type).
			StructTag(`json:"backupProfile,omitempty"`).
			Unique(),
	}
}
