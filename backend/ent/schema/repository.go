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
		field.String("url").
			StructTag(`json:"url"`),
		field.String("password").
			StructTag(`json:"password"`),
	}
}

// Edges of the Repository.
func (Repository) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("backup_profiles", BackupProfile.Type).
			Ref("repositories"),
		edge.From("archives", Archive.Type).
			Ref("repository"),
	}
}
