package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
)

type AuthSession struct {
	ent.Schema
}

func (AuthSession) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

func (AuthSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("session_id").
			StructTag(`json:"session_id"`).
			Unique(),
		field.Enum("status").
			StructTag(`json:"status"`).
			Values("PENDING", "AUTHENTICATED", "EXPIRED", "CANCELLED").
			Default("PENDING"),
		field.Time("expires_at").
			StructTag(`json:"expiresAt"`),
	}
}

func (AuthSession) Edges() []ent.Edge {
	return []ent.Edge{}
}
