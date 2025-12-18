package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
	"regexp"
)

type User struct {
	ent.Schema
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

var (
	emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("email").
			StructTag(`json:"email"`).
			Match(emailPattern).
			Unique(),
		field.Time("last_logged_in").
			StructTag(`json:"lastLoggedIn"`).
			Nillable().
			Optional(),
		field.Time("access_token_expires_at").
			Nillable().
			Optional(),
		field.Time("refresh_token_expires_at").
			Nillable().
			Optional(),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{}
}
