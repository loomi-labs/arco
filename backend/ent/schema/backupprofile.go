package schema

import (
	"regexp"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
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
		field.Bool("exclude_caches").
			StructTag(`json:"excludeCaches"`).
			Default(false).
			Comment("Exclude directories containing CACHEDIR.TAG file"),
		field.Enum("icon").
			StructTag(`json:"icon"`).
			Values("home", "briefcase", "book", "envelope", "camera", "fire"),
		field.Enum("compression_mode").
			StructTag(`json:"compressionMode"`).
			Values("none", "lz4", "zstd", "zlib", "lzma").
			Default("lz4").
			Comment("Compression algorithm for backups"),
		field.Int("compression_level").
			StructTag(`json:"compressionLevel"`).
			Optional().
			Nillable().
			Comment("Compression level (algorithm-specific range)").
			Min(0).
			Max(22),

		// UI States
		field.Bool("data_section_collapsed").
			StructTag(`json:"dataSectionCollapsed"`).
			Default(false),
		field.Bool("schedule_section_collapsed").
			StructTag(`json:"scheduleSectionCollapsed"`).
			Default(false),
		field.Bool("advanced_section_collapsed").
			StructTag(`json:"advancedSectionCollapsed"`).
			Default(true).
			Comment("UI state: whether advanced settings section is collapsed"),
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

// Annotations of the BackupProfile.
func (BackupProfile) Annotations() []schema.Annotation {
	return []schema.Annotation{
		&entsql.Annotation{
			Checks: map[string]string{
				"compression_level_valid": `(
					(compression_mode IN ('none', 'lz4') AND compression_level IS NULL) OR
					(compression_mode = 'zstd' AND compression_level >= 1 AND compression_level <= 22) OR
					(compression_mode = 'zlib' AND compression_level >= 0 AND compression_level <= 9) OR
					(compression_mode = 'lzma' AND compression_level >= 0 AND compression_level <= 6)
				)`,
			},
		},
	}
}
