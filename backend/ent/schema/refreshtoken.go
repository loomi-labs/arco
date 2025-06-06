package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"time"
)

type RefreshToken struct {
	ent.Schema
}

func (RefreshToken) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StructTag(`json:"id"`),
		field.UUID("user_id", uuid.UUID{}).
			StructTag(`json:"userId"`),
		field.String("token_hash").
			Sensitive(),
		field.Time("expires_at").
			StructTag(`json:"expiresAt"`),
		field.Time("created_at").
			StructTag(`json:"createdAt"`).
			Immutable().
			Default(time.Now),
		field.Time("last_used_at").
			StructTag(`json:"lastUsedAt"`).
			Nillable().
			Optional(),
	}
}

func (RefreshToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			StructTag(`json:"user,omitempty"`).
			Ref("refresh_tokens").
			Field("user_id").
			Required().
			Unique(),
	}
}
