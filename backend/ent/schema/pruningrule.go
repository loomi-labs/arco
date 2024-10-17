package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// PruningRule holds the schema definition for the PruningRule entity.
type PruningRule struct {
	ent.Schema
}

// Fields of the PruningRule.
func (PruningRule) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.Int("keep_hourly").
			StructTag(`json:"keepHourly"`),
		field.Int("keep_daily").
			StructTag(`json:"keepDaily"`),
		field.Int("keep_weekly").
			StructTag(`json:"keepWeekly"`),
		field.Int("keep_monthly").
			StructTag(`json:"keepMonthly"`),
		field.Int("keep_yearly").
			StructTag(`json:"keepYearly"`),
		field.Int("keep_within_days").
			StructTag(`json:"keepWithinDays"`),
	}
}

// Edges of the PruningRule.
func (PruningRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("backup_profile", BackupProfile.Type).
			Ref("pruning_rule").
			StructTag(`json:"backupProfile,omitempty"`).
			Unique().
			Required(),
	}
}
