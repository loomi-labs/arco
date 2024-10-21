package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
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
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Bool("is_enabled").
			StructTag(`json:"isEnabled"`),
		// https://borgbackup.readthedocs.io/en/stable/usage/prune.html
		// Fields to define the keep rules
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
		// Field to define the keep within interval
		field.Int("keep_within_days").
			StructTag(`json:"keepWithinDays"`),

		// Status fields
		field.Time("next_run").
			StructTag(`json:"nextRun"`).
			Optional(),
		field.Time("last_run").
			StructTag(`json:"lastRun"`).
			Nillable().
			Optional(),
		field.String("last_run_status").
			StructTag(`json:"lastRunStatus"`).
			Nillable().
			Optional(),
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
