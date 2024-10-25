package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// BackupSchedule holds the schema definition for the BackupSchedule entity.
type BackupSchedule struct {
	ent.Schema
}

// Fields of the BackupSchedule.
// Rules are enforced via hooks.
// Fields for the rules are immutable to simplify the rules. To change the schedule, a new schedule must be created.
func (BackupSchedule) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			StructTag(`json:"id"`),
		field.Time("updated_at").
			StructTag(`json:"updatedAt"`).
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Enum("mode").
			StructTag(`json:"mode"`).
			Values("disabled", "hourly", "daily", "weekly", "monthly").
			Default("disabled"),

		// Schedule fields
		field.Bool("hourly").
			StructTag(`json:"hourly"`).
			Default(false),
		field.Time("daily_at").
			StructTag(`json:"dailyAt"`),
		field.Enum("weekday").
			StructTag(`json:"weekday"`).
			Values("monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"),
		field.Time("weekly_at").
			StructTag(`json:"weeklyAt"`),
		field.Uint8("monthday").
			StructTag(`json:"monthday"`).
			Range(1, 30),
		field.Time("monthly_at").
			StructTag(`json:"monthlyAt"`),

		// Runtime fields
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

// Edges of the BackupSchedule.
func (BackupSchedule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("backup_profile", BackupProfile.Type).
			Ref("backup_schedule").
			StructTag(`json:"backupProfile,omitempty"`).
			Unique().
			Required(),
	}
}

// Indexes of the BackupSchedule.
func (BackupSchedule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("next_run"),
	}
}
