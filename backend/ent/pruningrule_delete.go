// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/predicate"
	"github.com/loomi-labs/arco/backend/ent/pruningrule"
)

// PruningRuleDelete is the builder for deleting a PruningRule entity.
type PruningRuleDelete struct {
	config
	hooks    []Hook
	mutation *PruningRuleMutation
}

// Where appends a list predicates to the PruningRuleDelete builder.
func (prd *PruningRuleDelete) Where(ps ...predicate.PruningRule) *PruningRuleDelete {
	prd.mutation.Where(ps...)
	return prd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (prd *PruningRuleDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, prd.sqlExec, prd.mutation, prd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (prd *PruningRuleDelete) ExecX(ctx context.Context) int {
	n, err := prd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (prd *PruningRuleDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(pruningrule.Table, sqlgraph.NewFieldSpec(pruningrule.FieldID, field.TypeInt))
	if ps := prd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, prd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	prd.mutation.done = true
	return affected, err
}

// PruningRuleDeleteOne is the builder for deleting a single PruningRule entity.
type PruningRuleDeleteOne struct {
	prd *PruningRuleDelete
}

// Where appends a list predicates to the PruningRuleDelete builder.
func (prdo *PruningRuleDeleteOne) Where(ps ...predicate.PruningRule) *PruningRuleDeleteOne {
	prdo.prd.mutation.Where(ps...)
	return prdo
}

// Exec executes the deletion query.
func (prdo *PruningRuleDeleteOne) Exec(ctx context.Context) error {
	n, err := prdo.prd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{pruningrule.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (prdo *PruningRuleDeleteOne) ExecX(ctx context.Context) {
	if err := prdo.Exec(ctx); err != nil {
		panic(err)
	}
}
