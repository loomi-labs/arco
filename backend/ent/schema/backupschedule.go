package schema

import (
	gen "arco/backend/ent"
	"arco/backend/ent/hook"
	"context"
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"fmt"
)

// BackupSchedule holds the schema definition for the BackupSchedule entity.
type BackupSchedule struct {
	ent.Schema
}

// Fields of the BackupSchedule.
// Rules are enforced via hooks.
func (BackupSchedule) Fields() []ent.Field {
	return []ent.Field{
		// Rule 1: when hourly is enabled, nothing else can be defined
		field.Bool("hourly").
			StructTag(`json:"hourly"`).
			Default(false),
		// Rule 2: when daily_at is defined, nothing else can be defined
		field.Time("daily_at").
			StructTag(`json:"dailyAt"`).
			Nillable().
			Optional(),
		// Rule 3: when weekly_at is defined, weekday must be defined
		// Rule 4: when weekly_at and weekday are defined, nothing else can be defined
		field.Enum("weekday").
			StructTag(`json:"weekday"`).
			Values("monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday").
			Nillable().
			Optional(),
		field.Time("weekly_at").
			StructTag(`json:"weeklyAt"`).
			Nillable().
			Optional(),
		// Rule 5: when monthly_at is defined, monthday must be defined
		// Rule 6: when monthly_at and monthday are defined, nothing else can be defined
		field.Uint8("monthday").
			StructTag(`json:"monthday"`).
			Range(1, 30).
			Nillable().
			Optional(),
		field.Time("monthly_at").
			StructTag(`json:"monthlyAt"`).
			Nillable().
			Optional(),
	}
}

// Edges of the BackupSchedule.
func (BackupSchedule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("backup_profile", BackupProfile.Type).
			Ref("backup_schedule").
			Unique().
			Required(),
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
						cntDefinitions++
					}
					if dailyAt, exists := m.DailyAt(); exists {
						cntDefinitions++
						if dailyAt.IsZero() {
							return nil, fmt.Errorf("daily_at cannot be zero")
						}
					}
					weeklyAt, existsWeeklyAt := m.WeeklyAt()
					_, existsWeekday := m.Weekday()
					if existsWeeklyAt || existsWeekday {
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
						cntDefinitions++
						if !(monthlyAtExists && monthdayExists) {
							return nil, fmt.Errorf("monthly_at and monthday must be defined together")
						}
						if monthlyAt.IsZero() {
							return nil, fmt.Errorf("monthly_at cannot be zero")
						}
					}
					if cntDefinitions == 0 {
						return nil, fmt.Errorf("schedule is not defined")
					}
					if cntDefinitions > 1 {
						return nil, fmt.Errorf("only one definition is allowed")
					}
					return next.Mutate(ctx, m)
				})
			},
			// Limit the hook only for these operations.
			ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne,
		),
	}
}
