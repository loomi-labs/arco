// Code generated by ent, DO NOT EDIT.

package settings

import (
	"time"

	"entgo.io/ent/dialect/sql"
)

const (
	// Label holds the string label denoting the settings type in the database.
	Label = "settings"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldShowWelcome holds the string denoting the show_welcome field in the database.
	FieldShowWelcome = "show_welcome"
	// Table holds the table name of the settings in the database.
	Table = "settings"
)

// Columns holds all SQL columns for settings fields.
var Columns = []string{
	FieldID,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldShowWelcome,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt func() time.Time
	// DefaultUpdatedAt holds the default value on creation for the "updated_at" field.
	DefaultUpdatedAt func() time.Time
	// UpdateDefaultUpdatedAt holds the default value on update for the "updated_at" field.
	UpdateDefaultUpdatedAt func() time.Time
	// DefaultShowWelcome holds the default value on creation for the "show_welcome" field.
	DefaultShowWelcome bool
)

// OrderOption defines the ordering options for the Settings queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByCreatedAt orders the results by the created_at field.
func ByCreatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreatedAt, opts...).ToFunc()
}

// ByUpdatedAt orders the results by the updated_at field.
func ByUpdatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUpdatedAt, opts...).ToFunc()
}

// ByShowWelcome orders the results by the show_welcome field.
func ByShowWelcome(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldShowWelcome, opts...).ToFunc()
}
