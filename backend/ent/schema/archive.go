package schema

import (
	"entgo.io/ent"
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
		field.Time("duration").
			StructTag(`json:"duration"`),
		field.String("borgID").
			StructTag(`json:"borgID"`),
	}
}

// Edges of the Archive.
func (Archive) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("repository", Repository.Type).
			Required().
			Unique(),
	}
}
