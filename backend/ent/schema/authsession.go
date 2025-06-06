package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
	"regexp"
)

type AuthSession struct {
	ent.Schema
}

func (AuthSession) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

var (
	sessionEmailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func (AuthSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			StructTag(`json:"id"`).
			Unique(),
		field.String("user_email").
			StructTag(`json:"userEmail"`).
			Match(sessionEmailPattern),
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
