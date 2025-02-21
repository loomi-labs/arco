package schema

import (
	"regexp"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
)

// BackupProfile holds the schema definition for the BackupProfile entity.
type BackupProfile struct {
	ent.Schema
}

func (BackupProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

// Fields of the BackupProfile.
func (BackupProfile) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("name").
			StructTag(`json:"name"`).
			MinLen(3).
			MaxLen(30),
		field.String("prefix").
			StructTag(`json:"prefix"`).
			// Match the prefix to be an alphanumeric string ending with a hyphen
			Match(regexp.MustCompile("^[a-z0-9]+-$")).
			// The prefix must be unique to ensure that archives belong to a single profile
			Unique().
			// To simplify the rules, the prefix is immutable
			Immutable(),
		field.Strings("backup_paths").
			StructTag(`json:"backupPaths"`).
			Default([]string{}),
		field.Strings("exclude_paths").
			StructTag(`json:"excludePaths"`).
			Optional().
			Default([]string{}),
		field.Enum("icon").
			StructTag(`json:"icon"`).
			Values("home", "briefcase", "book", "envelope", "camera", "fire"),
		field.Bool("data_section_collapsed").
			StructTag(`json:"dataSectionCollapsed"`).
			Default(false),
		field.Bool("schedule_section_collapsed").
			StructTag(`json:"scheduleSectionCollapsed"`).
			Default(false),
	}
}

// Edges of the BackupProfile.
func (BackupProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("repositories", Repository.Type).
			Required(),
		edge.To("archives", Archive.Type),
		edge.To("backup_schedule", BackupSchedule.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"backupSchedule,omitempty"`).
			Unique(),
		edge.To("pruning_rule", PruningRule.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"pruningRule,omitempty"`).
			Unique(),
		edge.From("notifications", Notification.Type).
			StructTag(`json:"notifications,omitempty"`).
			Ref("backup_profile"),
	}
}
