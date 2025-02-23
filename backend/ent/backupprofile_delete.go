// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/predicate"
)

// BackupProfileDelete is the builder for deleting a BackupProfile entity.
type BackupProfileDelete struct {
	config
	hooks    []Hook
	mutation *BackupProfileMutation
}

// Where appends a list predicates to the BackupProfileDelete builder.
func (bpd *BackupProfileDelete) Where(ps ...predicate.BackupProfile) *BackupProfileDelete {
	bpd.mutation.Where(ps...)
	return bpd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (bpd *BackupProfileDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, bpd.sqlExec, bpd.mutation, bpd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (bpd *BackupProfileDelete) ExecX(ctx context.Context) int {
	n, err := bpd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (bpd *BackupProfileDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(backupprofile.Table, sqlgraph.NewFieldSpec(backupprofile.FieldID, field.TypeInt))
	if ps := bpd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, bpd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	bpd.mutation.done = true
	return affected, err
}

// BackupProfileDeleteOne is the builder for deleting a single BackupProfile entity.
type BackupProfileDeleteOne struct {
	bpd *BackupProfileDelete
}

// Where appends a list predicates to the BackupProfileDelete builder.
func (bpdo *BackupProfileDeleteOne) Where(ps ...predicate.BackupProfile) *BackupProfileDeleteOne {
	bpdo.bpd.mutation.Where(ps...)
	return bpdo
}

// Exec executes the deletion query.
func (bpdo *BackupProfileDeleteOne) Exec(ctx context.Context) error {
	n, err := bpdo.bpd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{backupprofile.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (bpdo *BackupProfileDeleteOne) ExecX(ctx context.Context) {
	if err := bpdo.Exec(ctx); err != nil {
		panic(err)
	}
}
