package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
)

// CloudRepository holds the schema definition for the CloudRepository entity.
type CloudRepository struct {
	ent.Schema
}

func (CloudRepository) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

// Fields of the CloudRepository.
func (CloudRepository) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("cloud_id").
			StructTag(`json:"cloudId"`).
			Immutable(),
		field.Int64("storage_used_bytes").
			Default(0).
			StructTag(`json:"storageUsedBytes"`),
		field.Enum("location").
			Values("EU", "US").
			Immutable().
			StructTag(`json:"location"`),
	}
}

// Edges of the CloudRepository.
func (CloudRepository) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("repository", Repository.Type).
			Required().
			Unique().
			StructTag(`json:"repository"`),
	}
}
