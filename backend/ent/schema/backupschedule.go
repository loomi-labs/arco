package schema

import (
	gen "arco/backend/ent"
	"arco/backend/ent/hook"
	"context"
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"fmt"
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
		// Rule 1: when hourly is enabled, nothing else can be defined
		field.Bool("hourly").
			StructTag(`json:"hourly"`).
			Default(false).
			Immutable(),
		// Rule 2: when daily_at is defined, nothing else can be defined
		field.Time("daily_at").
			StructTag(`json:"dailyAt"`).
			Nillable().
			Optional().
			Immutable(),
		// Rule 3: when weekly_at is defined, weekday must be defined
		// Rule 4: when weekly_at and weekday are defined, nothing else can be defined
		field.Enum("weekday").
			StructTag(`json:"weekday"`).
			Values("monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday").
			Nillable().
			Optional().
			Immutable(),
		field.Time("weekly_at").
			StructTag(`json:"weeklyAt"`).
			Nillable().
			Optional().
			Immutable(),
		// Rule 5: when monthly_at is defined, monthday must be defined
		// Rule 6: when monthly_at and monthday are defined, nothing else can be defined
		field.Uint8("monthday").
			StructTag(`json:"monthday"`).
			Range(1, 30).
			Nillable().
			Optional().
			Immutable(),
		field.Time("monthly_at").
			StructTag(`json:"monthlyAt"`).
			Nillable().
			Optional().
			Immutable(),
		// Rule 7: at least one schedule must be defined

		// Not part of the rules
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

// Hooks for the BackupSchedule.
func (BackupSchedule) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.BackupScheduleFunc(func(ctx context.Context, m *gen.BackupScheduleMutation) (ent.Value, error) {
					cntDefinitions := 0
					if enabled, exists := m.Hourly(); exists && enabled {
						// Rule 1
						cntDefinitions++
					}
					if dailyAt, exists := m.DailyAt(); exists {
						// Rule 2
						cntDefinitions++
						if dailyAt.IsZero() {
							return nil, fmt.Errorf("daily_at cannot be zero")
						}
					}
					weeklyAt, existsWeeklyAt := m.WeeklyAt()
					_, existsWeekday := m.Weekday()
					if existsWeeklyAt || existsWeekday {
						// Rule 3 and Rule 4
						cntDefinitions++
						if !(existsWeeklyAt && existsWeekday) {
							return nil, fmt.Errorf("weekly_at and weekday must be defined together")
						}
						if weeklyAt.IsZero() {
							return nil, fmt.Errorf("weekly_at cannot be zero")
						}
					}
					monthlyAt, monthlyAtExists := m.MonthlyAt()
					_, monthdayExists := m.Monthday()
					if monthlyAtExists || monthdayExists {
						// Rule 5 and Rule 6
						cntDefinitions++
						if !(monthlyAtExists && monthdayExists) {
							return nil, fmt.Errorf("monthly_at and monthday must be defined together")
						}
						if monthlyAt.IsZero() {
							return nil, fmt.Errorf("monthly_at cannot be zero")
						}
					}
					if cntDefinitions == 0 {
						// Rule 7
						return nil, fmt.Errorf("schedule is not defined")
					}
					if cntDefinitions > 1 {
						// Evaluation of Rule 1, 2, 4, 6
						return nil, fmt.Errorf("only one definition is allowed")
					}
					return next.Mutate(ctx, m)
				})
			},
			// Limit the hook only for these operations.
			ent.OpCreate,
		),
	}
}
