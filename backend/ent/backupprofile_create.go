// Code generated by ent, DO NOT EDIT.

package ent

import (
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	"arco/backend/ent/repository"
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
)

// BackupProfileCreate is the builder for creating a BackupProfile entity.
type BackupProfileCreate struct {
	config
	mutation *BackupProfileMutation
	hooks    []Hook
}

// SetName sets the "name" field.
func (bpc *BackupProfileCreate) SetName(s string) *BackupProfileCreate {
	bpc.mutation.SetName(s)
	return bpc
}

// SetPrefix sets the "prefix" field.
func (bpc *BackupProfileCreate) SetPrefix(s string) *BackupProfileCreate {
	bpc.mutation.SetPrefix(s)
	return bpc
}

// SetDirectories sets the "directories" field.
func (bpc *BackupProfileCreate) SetDirectories(s []string) *BackupProfileCreate {
	bpc.mutation.SetDirectories(s)
	return bpc
}

// SetIsSetupComplete sets the "is_setup_complete" field.
func (bpc *BackupProfileCreate) SetIsSetupComplete(b bool) *BackupProfileCreate {
	bpc.mutation.SetIsSetupComplete(b)
	return bpc
}

// SetNillableIsSetupComplete sets the "is_setup_complete" field if the given value is not nil.
func (bpc *BackupProfileCreate) SetNillableIsSetupComplete(b *bool) *BackupProfileCreate {
	if b != nil {
		bpc.SetIsSetupComplete(*b)
	}
	return bpc
}

// SetID sets the "id" field.
func (bpc *BackupProfileCreate) SetID(i int) *BackupProfileCreate {
	bpc.mutation.SetID(i)
	return bpc
}

// AddRepositoryIDs adds the "repositories" edge to the Repository entity by IDs.
func (bpc *BackupProfileCreate) AddRepositoryIDs(ids ...int) *BackupProfileCreate {
	bpc.mutation.AddRepositoryIDs(ids...)
	return bpc
}

// AddRepositories adds the "repositories" edges to the Repository entity.
func (bpc *BackupProfileCreate) AddRepositories(r ...*Repository) *BackupProfileCreate {
	ids := make([]int, len(r))
	for i := range r {
		ids[i] = r[i].ID
	}
	return bpc.AddRepositoryIDs(ids...)
}

// SetBackupScheduleID sets the "backup_schedule" edge to the BackupSchedule entity by ID.
func (bpc *BackupProfileCreate) SetBackupScheduleID(id int) *BackupProfileCreate {
	bpc.mutation.SetBackupScheduleID(id)
	return bpc
}

// SetNillableBackupScheduleID sets the "backup_schedule" edge to the BackupSchedule entity by ID if the given value is not nil.
func (bpc *BackupProfileCreate) SetNillableBackupScheduleID(id *int) *BackupProfileCreate {
	if id != nil {
		bpc = bpc.SetBackupScheduleID(*id)
	}
	return bpc
}

// SetBackupSchedule sets the "backup_schedule" edge to the BackupSchedule entity.
func (bpc *BackupProfileCreate) SetBackupSchedule(b *BackupSchedule) *BackupProfileCreate {
	return bpc.SetBackupScheduleID(b.ID)
}

// Mutation returns the BackupProfileMutation object of the builder.
func (bpc *BackupProfileCreate) Mutation() *BackupProfileMutation {
	return bpc.mutation
}

// Save creates the BackupProfile in the database.
func (bpc *BackupProfileCreate) Save(ctx context.Context) (*BackupProfile, error) {
	bpc.defaults()
	return withHooks(ctx, bpc.sqlSave, bpc.mutation, bpc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (bpc *BackupProfileCreate) SaveX(ctx context.Context) *BackupProfile {
	v, err := bpc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (bpc *BackupProfileCreate) Exec(ctx context.Context) error {
	_, err := bpc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bpc *BackupProfileCreate) ExecX(ctx context.Context) {
	if err := bpc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bpc *BackupProfileCreate) defaults() {
	if _, ok := bpc.mutation.IsSetupComplete(); !ok {
		v := backupprofile.DefaultIsSetupComplete
		bpc.mutation.SetIsSetupComplete(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bpc *BackupProfileCreate) check() error {
	if _, ok := bpc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "BackupProfile.name"`)}
	}
	if _, ok := bpc.mutation.Prefix(); !ok {
		return &ValidationError{Name: "prefix", err: errors.New(`ent: missing required field "BackupProfile.prefix"`)}
	}
	if _, ok := bpc.mutation.Directories(); !ok {
		return &ValidationError{Name: "directories", err: errors.New(`ent: missing required field "BackupProfile.directories"`)}
	}
	if _, ok := bpc.mutation.IsSetupComplete(); !ok {
		return &ValidationError{Name: "is_setup_complete", err: errors.New(`ent: missing required field "BackupProfile.is_setup_complete"`)}
	}
	return nil
}

func (bpc *BackupProfileCreate) sqlSave(ctx context.Context) (*BackupProfile, error) {
	if err := bpc.check(); err != nil {
		return nil, err
	}
	_node, _spec := bpc.createSpec()
	if err := sqlgraph.CreateNode(ctx, bpc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != _node.ID {
		id := _spec.ID.Value.(int64)
		_node.ID = int(id)
	}
	bpc.mutation.id = &_node.ID
	bpc.mutation.done = true
	return _node, nil
}

func (bpc *BackupProfileCreate) createSpec() (*BackupProfile, *sqlgraph.CreateSpec) {
	var (
		_node = &BackupProfile{config: bpc.config}
		_spec = sqlgraph.NewCreateSpec(backupprofile.Table, sqlgraph.NewFieldSpec(backupprofile.FieldID, field.TypeInt))
	)
	if id, ok := bpc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := bpc.mutation.Name(); ok {
		_spec.SetField(backupprofile.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := bpc.mutation.Prefix(); ok {
		_spec.SetField(backupprofile.FieldPrefix, field.TypeString, value)
		_node.Prefix = value
	}
	if value, ok := bpc.mutation.Directories(); ok {
		_spec.SetField(backupprofile.FieldDirectories, field.TypeJSON, value)
		_node.Directories = value
	}
	if value, ok := bpc.mutation.IsSetupComplete(); ok {
		_spec.SetField(backupprofile.FieldIsSetupComplete, field.TypeBool, value)
		_node.IsSetupComplete = value
	}
	if nodes := bpc.mutation.RepositoriesIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   backupprofile.RepositoriesTable,
			Columns: backupprofile.RepositoriesPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(repository.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bpc.mutation.BackupScheduleIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   backupprofile.BackupScheduleTable,
			Columns: []string{backupprofile.BackupScheduleColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(backupschedule.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// BackupProfileCreateBulk is the builder for creating many BackupProfile entities in bulk.
type BackupProfileCreateBulk struct {
	config
	err      error
	builders []*BackupProfileCreate
}

// Save creates the BackupProfile entities in the database.
func (bpcb *BackupProfileCreateBulk) Save(ctx context.Context) ([]*BackupProfile, error) {
	if bpcb.err != nil {
		return nil, bpcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(bpcb.builders))
	nodes := make([]*BackupProfile, len(bpcb.builders))
	mutators := make([]Mutator, len(bpcb.builders))
	for i := range bpcb.builders {
		func(i int, root context.Context) {
			builder := bpcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*BackupProfileMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, bpcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, bpcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				if specs[i].ID.Value != nil && nodes[i].ID == 0 {
					id := specs[i].ID.Value.(int64)
					nodes[i].ID = int(id)
				}
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, bpcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (bpcb *BackupProfileCreateBulk) SaveX(ctx context.Context) []*BackupProfile {
	v, err := bpcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (bpcb *BackupProfileCreateBulk) Exec(ctx context.Context) error {
	_, err := bpcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bpcb *BackupProfileCreateBulk) ExecX(ctx context.Context) {
	if err := bpcb.Exec(ctx); err != nil {
		panic(err)
	}
}
