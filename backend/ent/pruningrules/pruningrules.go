// Code generated by ent, DO NOT EDIT.

package pruningrules

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the pruningrules type in the database.
	Label = "pruning_rules"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldKeepHourly holds the string denoting the keep_hourly field in the database.
	FieldKeepHourly = "keep_hourly"
	// FieldKeepDaily holds the string denoting the keep_daily field in the database.
	FieldKeepDaily = "keep_daily"
	// FieldKeepWeekly holds the string denoting the keep_weekly field in the database.
	FieldKeepWeekly = "keep_weekly"
	// FieldKeepMonthly holds the string denoting the keep_monthly field in the database.
	FieldKeepMonthly = "keep_monthly"
	// FieldKeepYearly holds the string denoting the keep_yearly field in the database.
	FieldKeepYearly = "keep_yearly"
	// FieldKeepWithinDays holds the string denoting the keep_within_days field in the database.
	FieldKeepWithinDays = "keep_within_days"
	// EdgeBackupProfile holds the string denoting the backup_profile edge name in mutations.
	EdgeBackupProfile = "backup_profile"
	// Table holds the table name of the pruningrules in the database.
	Table = "pruning_rules"
	// BackupProfileTable is the table that holds the backup_profile relation/edge.
	BackupProfileTable = "pruning_rules"
	// BackupProfileInverseTable is the table name for the BackupProfile entity.
	// It exists in this package in order to avoid circular dependency with the "backupprofile" package.
	BackupProfileInverseTable = "backup_profiles"
	// BackupProfileColumn is the table column denoting the backup_profile relation/edge.
	BackupProfileColumn = "backup_profile_pruning_rules"
)

// Columns holds all SQL columns for pruningrules fields.
var Columns = []string{
	FieldID,
	FieldKeepHourly,
	FieldKeepDaily,
	FieldKeepWeekly,
	FieldKeepMonthly,
	FieldKeepYearly,
	FieldKeepWithinDays,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "pruning_rules"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"backup_profile_pruning_rules",
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	for i := range ForeignKeys {
		if column == ForeignKeys[i] {
			return true
		}
	}
	return false
}

// OrderOption defines the ordering options for the PruningRules queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByKeepHourly orders the results by the keep_hourly field.
func ByKeepHourly(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKeepHourly, opts...).ToFunc()
}

// ByKeepDaily orders the results by the keep_daily field.
func ByKeepDaily(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKeepDaily, opts...).ToFunc()
}

// ByKeepWeekly orders the results by the keep_weekly field.
func ByKeepWeekly(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKeepWeekly, opts...).ToFunc()
}

// ByKeepMonthly orders the results by the keep_monthly field.
func ByKeepMonthly(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKeepMonthly, opts...).ToFunc()
}

// ByKeepYearly orders the results by the keep_yearly field.
func ByKeepYearly(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKeepYearly, opts...).ToFunc()
}

// ByKeepWithinDays orders the results by the keep_within_days field.
func ByKeepWithinDays(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKeepWithinDays, opts...).ToFunc()
}

// ByBackupProfileField orders the results by backup_profile field.
func ByBackupProfileField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBackupProfileStep(), sql.OrderByField(field, opts...))
	}
}
func newBackupProfileStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BackupProfileInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, BackupProfileTable, BackupProfileColumn),
	)
}