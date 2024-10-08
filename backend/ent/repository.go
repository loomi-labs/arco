// Code generated by ent, DO NOT EDIT.

package ent

import (
	"arco/backend/ent/repository"
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
)

// Repository is the model entity for the Repository schema.
type Repository struct {
	config `json:"-"`
	// ID of the ent.
	ID int `json:"id"`
	// Name holds the value of the "name" field.
	Name string `json:"name"`
	// Location holds the value of the "location" field.
	Location string `json:"location"`
	// Password holds the value of the "password" field.
	Password string `json:"password"`
	// StatsTotalChunks holds the value of the "stats_total_chunks" field.
	StatsTotalChunks int `json:"stats_total_chunks"`
	// StatsTotalSize holds the value of the "stats_total_size" field.
	StatsTotalSize int `json:"stats_total_size"`
	// StatsTotalCsize holds the value of the "stats_total_csize" field.
	StatsTotalCsize int `json:"stats_total_csize"`
	// StatsTotalUniqueChunks holds the value of the "stats_total_unique_chunks" field.
	StatsTotalUniqueChunks int `json:"stats_total_unique_chunks"`
	// StatsUniqueSize holds the value of the "stats_unique_size" field.
	StatsUniqueSize int `json:"stats_unique_size"`
	// StatsUniqueCsize holds the value of the "stats_unique_csize" field.
	StatsUniqueCsize int `json:"stats_unique_csize"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the RepositoryQuery when eager-loading is set.
	Edges        RepositoryEdges `json:"edges"`
	selectValues sql.SelectValues
}

// RepositoryEdges holds the relations/edges for other nodes in the graph.
type RepositoryEdges struct {
	// BackupProfiles holds the value of the backup_profiles edge.
	BackupProfiles []*BackupProfile `json:"backupProfiles,omitempty"`
	// Archives holds the value of the archives edge.
	Archives []*Archive `json:"archives,omitempty"`
	// FailedBackupRuns holds the value of the failed_backup_runs edge.
	FailedBackupRuns []*FailedBackupRun `json:"failedBackupRuns,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// BackupProfilesOrErr returns the BackupProfiles value or an error if the edge
// was not loaded in eager-loading.
func (e RepositoryEdges) BackupProfilesOrErr() ([]*BackupProfile, error) {
	if e.loadedTypes[0] {
		return e.BackupProfiles, nil
	}
	return nil, &NotLoadedError{edge: "backup_profiles"}
}

// ArchivesOrErr returns the Archives value or an error if the edge
// was not loaded in eager-loading.
func (e RepositoryEdges) ArchivesOrErr() ([]*Archive, error) {
	if e.loadedTypes[1] {
		return e.Archives, nil
	}
	return nil, &NotLoadedError{edge: "archives"}
}

// FailedBackupRunsOrErr returns the FailedBackupRuns value or an error if the edge
// was not loaded in eager-loading.
func (e RepositoryEdges) FailedBackupRunsOrErr() ([]*FailedBackupRun, error) {
	if e.loadedTypes[2] {
		return e.FailedBackupRuns, nil
	}
	return nil, &NotLoadedError{edge: "failed_backup_runs"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Repository) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case repository.FieldID, repository.FieldStatsTotalChunks, repository.FieldStatsTotalSize, repository.FieldStatsTotalCsize, repository.FieldStatsTotalUniqueChunks, repository.FieldStatsUniqueSize, repository.FieldStatsUniqueCsize:
			values[i] = new(sql.NullInt64)
		case repository.FieldName, repository.FieldLocation, repository.FieldPassword:
			values[i] = new(sql.NullString)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Repository fields.
func (r *Repository) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case repository.FieldID:
			value, ok := values[i].(*sql.NullInt64)
			if !ok {
				return fmt.Errorf("unexpected type %T for field id", value)
			}
			r.ID = int(value.Int64)
		case repository.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				r.Name = value.String
			}
		case repository.FieldLocation:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field location", values[i])
			} else if value.Valid {
				r.Location = value.String
			}
		case repository.FieldPassword:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field password", values[i])
			} else if value.Valid {
				r.Password = value.String
			}
		case repository.FieldStatsTotalChunks:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field stats_total_chunks", values[i])
			} else if value.Valid {
				r.StatsTotalChunks = int(value.Int64)
			}
		case repository.FieldStatsTotalSize:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field stats_total_size", values[i])
			} else if value.Valid {
				r.StatsTotalSize = int(value.Int64)
			}
		case repository.FieldStatsTotalCsize:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field stats_total_csize", values[i])
			} else if value.Valid {
				r.StatsTotalCsize = int(value.Int64)
			}
		case repository.FieldStatsTotalUniqueChunks:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field stats_total_unique_chunks", values[i])
			} else if value.Valid {
				r.StatsTotalUniqueChunks = int(value.Int64)
			}
		case repository.FieldStatsUniqueSize:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field stats_unique_size", values[i])
			} else if value.Valid {
				r.StatsUniqueSize = int(value.Int64)
			}
		case repository.FieldStatsUniqueCsize:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field stats_unique_csize", values[i])
			} else if value.Valid {
				r.StatsUniqueCsize = int(value.Int64)
			}
		default:
			r.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Repository.
// This includes values selected through modifiers, order, etc.
func (r *Repository) Value(name string) (ent.Value, error) {
	return r.selectValues.Get(name)
}

// QueryBackupProfiles queries the "backup_profiles" edge of the Repository entity.
func (r *Repository) QueryBackupProfiles() *BackupProfileQuery {
	return NewRepositoryClient(r.config).QueryBackupProfiles(r)
}

// QueryArchives queries the "archives" edge of the Repository entity.
func (r *Repository) QueryArchives() *ArchiveQuery {
	return NewRepositoryClient(r.config).QueryArchives(r)
}

// QueryFailedBackupRuns queries the "failed_backup_runs" edge of the Repository entity.
func (r *Repository) QueryFailedBackupRuns() *FailedBackupRunQuery {
	return NewRepositoryClient(r.config).QueryFailedBackupRuns(r)
}

// Update returns a builder for updating this Repository.
// Note that you need to call Repository.Unwrap() before calling this method if this Repository
// was returned from a transaction, and the transaction was committed or rolled back.
func (r *Repository) Update() *RepositoryUpdateOne {
	return NewRepositoryClient(r.config).UpdateOne(r)
}

// Unwrap unwraps the Repository entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (r *Repository) Unwrap() *Repository {
	_tx, ok := r.config.driver.(*txDriver)
	if !ok {
		panic("ent: Repository is not a transactional entity")
	}
	r.config.driver = _tx.drv
	return r
}

// String implements the fmt.Stringer.
func (r *Repository) String() string {
	var builder strings.Builder
	builder.WriteString("Repository(")
	builder.WriteString(fmt.Sprintf("id=%v, ", r.ID))
	builder.WriteString("name=")
	builder.WriteString(r.Name)
	builder.WriteString(", ")
	builder.WriteString("location=")
	builder.WriteString(r.Location)
	builder.WriteString(", ")
	builder.WriteString("password=")
	builder.WriteString(r.Password)
	builder.WriteString(", ")
	builder.WriteString("stats_total_chunks=")
	builder.WriteString(fmt.Sprintf("%v", r.StatsTotalChunks))
	builder.WriteString(", ")
	builder.WriteString("stats_total_size=")
	builder.WriteString(fmt.Sprintf("%v", r.StatsTotalSize))
	builder.WriteString(", ")
	builder.WriteString("stats_total_csize=")
	builder.WriteString(fmt.Sprintf("%v", r.StatsTotalCsize))
	builder.WriteString(", ")
	builder.WriteString("stats_total_unique_chunks=")
	builder.WriteString(fmt.Sprintf("%v", r.StatsTotalUniqueChunks))
	builder.WriteString(", ")
	builder.WriteString("stats_unique_size=")
	builder.WriteString(fmt.Sprintf("%v", r.StatsUniqueSize))
	builder.WriteString(", ")
	builder.WriteString("stats_unique_csize=")
	builder.WriteString(fmt.Sprintf("%v", r.StatsUniqueCsize))
	builder.WriteByte(')')
	return builder.String()
}

// Repositories is a parsable slice of Repository.
type Repositories []*Repository
