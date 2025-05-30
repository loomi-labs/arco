// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/backupschedule"
)

// BackupSchedule is the model entity for the BackupSchedule schema.
type BackupSchedule struct {
	config `json:"-"`
	// ID of the ent.
	ID int `json:"id"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updatedAt"`
	// Mode holds the value of the "mode" field.
	Mode backupschedule.Mode `json:"mode"`
	// DailyAt holds the value of the "daily_at" field.
	DailyAt time.Time `json:"dailyAt"`
	// Weekday holds the value of the "weekday" field.
	Weekday backupschedule.Weekday `json:"weekday"`
	// WeeklyAt holds the value of the "weekly_at" field.
	WeeklyAt time.Time `json:"weeklyAt"`
	// Monthday holds the value of the "monthday" field.
	Monthday uint8 `json:"monthday"`
	// MonthlyAt holds the value of the "monthly_at" field.
	MonthlyAt time.Time `json:"monthlyAt"`
	// NextRun holds the value of the "next_run" field.
	NextRun time.Time `json:"nextRun"`
	// LastRun holds the value of the "last_run" field.
	LastRun *time.Time `json:"lastRun"`
	// LastRunStatus holds the value of the "last_run_status" field.
	LastRunStatus *string `json:"lastRunStatus"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the BackupScheduleQuery when eager-loading is set.
	Edges                          BackupScheduleEdges `json:"edges"`
	backup_profile_backup_schedule *int
	selectValues                   sql.SelectValues
}

// BackupScheduleEdges holds the relations/edges for other nodes in the graph.
type BackupScheduleEdges struct {
	// BackupProfile holds the value of the backup_profile edge.
	BackupProfile *BackupProfile `json:"backupProfile,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// BackupProfileOrErr returns the BackupProfile value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BackupScheduleEdges) BackupProfileOrErr() (*BackupProfile, error) {
	if e.BackupProfile != nil {
		return e.BackupProfile, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: backupprofile.Label}
	}
	return nil, &NotLoadedError{edge: "backup_profile"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*BackupSchedule) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case backupschedule.FieldID, backupschedule.FieldMonthday:
			values[i] = new(sql.NullInt64)
		case backupschedule.FieldMode, backupschedule.FieldWeekday, backupschedule.FieldLastRunStatus:
			values[i] = new(sql.NullString)
		case backupschedule.FieldCreatedAt, backupschedule.FieldUpdatedAt, backupschedule.FieldDailyAt, backupschedule.FieldWeeklyAt, backupschedule.FieldMonthlyAt, backupschedule.FieldNextRun, backupschedule.FieldLastRun:
			values[i] = new(sql.NullTime)
		case backupschedule.ForeignKeys[0]: // backup_profile_backup_schedule
			values[i] = new(sql.NullInt64)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the BackupSchedule fields.
func (bs *BackupSchedule) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case backupschedule.FieldID:
			value, ok := values[i].(*sql.NullInt64)
			if !ok {
				return fmt.Errorf("unexpected type %T for field id", value)
			}
			bs.ID = int(value.Int64)
		case backupschedule.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				bs.CreatedAt = value.Time
			}
		case backupschedule.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				bs.UpdatedAt = value.Time
			}
		case backupschedule.FieldMode:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field mode", values[i])
			} else if value.Valid {
				bs.Mode = backupschedule.Mode(value.String)
			}
		case backupschedule.FieldDailyAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field daily_at", values[i])
			} else if value.Valid {
				bs.DailyAt = value.Time
			}
		case backupschedule.FieldWeekday:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field weekday", values[i])
			} else if value.Valid {
				bs.Weekday = backupschedule.Weekday(value.String)
			}
		case backupschedule.FieldWeeklyAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field weekly_at", values[i])
			} else if value.Valid {
				bs.WeeklyAt = value.Time
			}
		case backupschedule.FieldMonthday:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field monthday", values[i])
			} else if value.Valid {
				bs.Monthday = uint8(value.Int64)
			}
		case backupschedule.FieldMonthlyAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field monthly_at", values[i])
			} else if value.Valid {
				bs.MonthlyAt = value.Time
			}
		case backupschedule.FieldNextRun:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field next_run", values[i])
			} else if value.Valid {
				bs.NextRun = value.Time
			}
		case backupschedule.FieldLastRun:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field last_run", values[i])
			} else if value.Valid {
				bs.LastRun = new(time.Time)
				*bs.LastRun = value.Time
			}
		case backupschedule.FieldLastRunStatus:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field last_run_status", values[i])
			} else if value.Valid {
				bs.LastRunStatus = new(string)
				*bs.LastRunStatus = value.String
			}
		case backupschedule.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for edge-field backup_profile_backup_schedule", value)
			} else if value.Valid {
				bs.backup_profile_backup_schedule = new(int)
				*bs.backup_profile_backup_schedule = int(value.Int64)
			}
		default:
			bs.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the BackupSchedule.
// This includes values selected through modifiers, order, etc.
func (bs *BackupSchedule) Value(name string) (ent.Value, error) {
	return bs.selectValues.Get(name)
}

// QueryBackupProfile queries the "backup_profile" edge of the BackupSchedule entity.
func (bs *BackupSchedule) QueryBackupProfile() *BackupProfileQuery {
	return NewBackupScheduleClient(bs.config).QueryBackupProfile(bs)
}

// Update returns a builder for updating this BackupSchedule.
// Note that you need to call BackupSchedule.Unwrap() before calling this method if this BackupSchedule
// was returned from a transaction, and the transaction was committed or rolled back.
func (bs *BackupSchedule) Update() *BackupScheduleUpdateOne {
	return NewBackupScheduleClient(bs.config).UpdateOne(bs)
}

// Unwrap unwraps the BackupSchedule entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (bs *BackupSchedule) Unwrap() *BackupSchedule {
	_tx, ok := bs.config.driver.(*txDriver)
	if !ok {
		panic("ent: BackupSchedule is not a transactional entity")
	}
	bs.config.driver = _tx.drv
	return bs
}

// String implements the fmt.Stringer.
func (bs *BackupSchedule) String() string {
	var builder strings.Builder
	builder.WriteString("BackupSchedule(")
	builder.WriteString(fmt.Sprintf("id=%v, ", bs.ID))
	builder.WriteString("created_at=")
	builder.WriteString(bs.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(bs.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("mode=")
	builder.WriteString(fmt.Sprintf("%v", bs.Mode))
	builder.WriteString(", ")
	builder.WriteString("daily_at=")
	builder.WriteString(bs.DailyAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("weekday=")
	builder.WriteString(fmt.Sprintf("%v", bs.Weekday))
	builder.WriteString(", ")
	builder.WriteString("weekly_at=")
	builder.WriteString(bs.WeeklyAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("monthday=")
	builder.WriteString(fmt.Sprintf("%v", bs.Monthday))
	builder.WriteString(", ")
	builder.WriteString("monthly_at=")
	builder.WriteString(bs.MonthlyAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("next_run=")
	builder.WriteString(bs.NextRun.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := bs.LastRun; v != nil {
		builder.WriteString("last_run=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	if v := bs.LastRunStatus; v != nil {
		builder.WriteString("last_run_status=")
		builder.WriteString(*v)
	}
	builder.WriteByte(')')
	return builder.String()
}

// BackupSchedules is a parsable slice of BackupSchedule.
type BackupSchedules []*BackupSchedule
