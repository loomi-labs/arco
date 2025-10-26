package schema

import (
	"regexp"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/schema/mixin"
)

// Repository holds the schema definition for the Repository entity.
type Repository struct {
	ent.Schema
}

func (Repository) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimestampMixin{},
	}
}

var (
	ValRepositoryMinNameLength = 3
	ValRepositoryMaxNameLength = 30
	ValRepositoryNamePattern   = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`) // Only letters, numbers, hyphens, and underscores
)

// Fields of the Repository.
func (Repository) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.String("name").
			StructTag(`json:"name"`).
			MinLen(ValRepositoryMinNameLength).
			MaxLen(ValRepositoryMaxNameLength).
			Match(ValRepositoryNamePattern).
			Unique(),
		field.String("url").
			StructTag(`json:"url"`).
			Unique(),
		field.String("password").
			StructTag(`json:"password"`),

		// Integrity check
		field.Time("next_integrity_check").
			StructTag(`json:"nextIntegrityCheck"`).
			Nillable().
			Optional(),

		// Stats
		// Borg repository statistics from cache stats.
		// "total" metrics = aggregate capacity with reference counts (what would be restored)
		// "unique" metrics = deduplicated storage (actual disk usage after deduplication)
		// Reference: https://borgbackup.readthedocs.io/en/stable/internals/frontends.html
		field.Int("stats_total_chunks").
			Comment("Total number of all chunks across all archives (including duplicates)").
			Default(0).
			StructTag(`json:"statsTotalChunks"`),
		field.Int("stats_total_size").
			Comment("Total uncompressed size of all chunks multiplied by their reference counts").
			Default(0).
			StructTag(`json:"statsTotalSize"`),
		field.Int("stats_total_csize").
			Comment("Total compressed size of all chunks multiplied by their reference counts (on-disk footprint with references)").
			Default(0).
			StructTag(`json:"statsTotalCsize"`),
		field.Int("stats_total_unique_chunks").
			Comment("Number of unique/deduplicated chunks").
			Default(0).
			StructTag(`json:"statsTotalUniqueChunks"`),
		field.Int("stats_unique_size").
			Comment("Uncompressed size of unique chunks only (raw deduplicated data volume)").
			Default(0).
			StructTag(`json:"statsUniqueSize"`),
		field.Int("stats_unique_csize").
			Comment("Compressed size of unique chunks only (actual storage consumed on disk)").
			Default(0).
			StructTag(`json:"statsUniqueCsize"`),
	}
}

// Edges of the Repository.
func (Repository) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("backup_profiles", BackupProfile.Type).
			StructTag(`json:"backupProfiles,omitempty"`).
			Ref("repositories"),
		edge.From("archives", Archive.Type).
			Ref("repository"),
		edge.From("notifications", Notification.Type).
			StructTag(`json:"notifications,omitempty"`).
			Ref("repository"),
		edge.From("cloud_repository", CloudRepository.Type).
			StructTag(`json:"cloudRepository,omitempty"`).
			Ref("repository").
			Unique(),
	}
}
