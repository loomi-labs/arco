// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
	"github.com/loomi-labs/arco/backend/ent/repository"
)

// ArchiveCreate is the builder for creating a Archive entity.
type ArchiveCreate struct {
	config
	mutation *ArchiveMutation
	hooks    []Hook
}

// SetCreatedAt sets the "created_at" field.
func (ac *ArchiveCreate) SetCreatedAt(t time.Time) *ArchiveCreate {
	ac.mutation.SetCreatedAt(t)
	return ac
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (ac *ArchiveCreate) SetNillableCreatedAt(t *time.Time) *ArchiveCreate {
	if t != nil {
		ac.SetCreatedAt(*t)
	}
	return ac
}

// SetUpdatedAt sets the "updated_at" field.
func (ac *ArchiveCreate) SetUpdatedAt(t time.Time) *ArchiveCreate {
	ac.mutation.SetUpdatedAt(t)
	return ac
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (ac *ArchiveCreate) SetNillableUpdatedAt(t *time.Time) *ArchiveCreate {
	if t != nil {
		ac.SetUpdatedAt(*t)
	}
	return ac
}

// SetName sets the "name" field.
func (ac *ArchiveCreate) SetName(s string) *ArchiveCreate {
	ac.mutation.SetName(s)
	return ac
}

// SetDuration sets the "duration" field.
func (ac *ArchiveCreate) SetDuration(f float64) *ArchiveCreate {
	ac.mutation.SetDuration(f)
	return ac
}

// SetBorgID sets the "borg_id" field.
func (ac *ArchiveCreate) SetBorgID(s string) *ArchiveCreate {
	ac.mutation.SetBorgID(s)
	return ac
}

// SetWillBePruned sets the "will_be_pruned" field.
func (ac *ArchiveCreate) SetWillBePruned(b bool) *ArchiveCreate {
	ac.mutation.SetWillBePruned(b)
	return ac
}

// SetNillableWillBePruned sets the "will_be_pruned" field if the given value is not nil.
func (ac *ArchiveCreate) SetNillableWillBePruned(b *bool) *ArchiveCreate {
	if b != nil {
		ac.SetWillBePruned(*b)
	}
	return ac
}

// SetID sets the "id" field.
func (ac *ArchiveCreate) SetID(i int) *ArchiveCreate {
	ac.mutation.SetID(i)
	return ac
}

// SetRepositoryID sets the "repository" edge to the Repository entity by ID.
func (ac *ArchiveCreate) SetRepositoryID(id int) *ArchiveCreate {
	ac.mutation.SetRepositoryID(id)
	return ac
}

// SetRepository sets the "repository" edge to the Repository entity.
func (ac *ArchiveCreate) SetRepository(r *Repository) *ArchiveCreate {
	return ac.SetRepositoryID(r.ID)
}

// SetBackupProfileID sets the "backup_profile" edge to the BackupProfile entity by ID.
func (ac *ArchiveCreate) SetBackupProfileID(id int) *ArchiveCreate {
	ac.mutation.SetBackupProfileID(id)
	return ac
}

// SetNillableBackupProfileID sets the "backup_profile" edge to the BackupProfile entity by ID if the given value is not nil.
func (ac *ArchiveCreate) SetNillableBackupProfileID(id *int) *ArchiveCreate {
	if id != nil {
		ac = ac.SetBackupProfileID(*id)
	}
	return ac
}

// SetBackupProfile sets the "backup_profile" edge to the BackupProfile entity.
func (ac *ArchiveCreate) SetBackupProfile(b *BackupProfile) *ArchiveCreate {
	return ac.SetBackupProfileID(b.ID)
}

// Mutation returns the ArchiveMutation object of the builder.
func (ac *ArchiveCreate) Mutation() *ArchiveMutation {
	return ac.mutation
}

// Save creates the Archive in the database.
func (ac *ArchiveCreate) Save(ctx context.Context) (*Archive, error) {
	ac.defaults()
	return withHooks(ctx, ac.sqlSave, ac.mutation, ac.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (ac *ArchiveCreate) SaveX(ctx context.Context) *Archive {
	v, err := ac.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (ac *ArchiveCreate) Exec(ctx context.Context) error {
	_, err := ac.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ac *ArchiveCreate) ExecX(ctx context.Context) {
	if err := ac.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (ac *ArchiveCreate) defaults() {
	if _, ok := ac.mutation.CreatedAt(); !ok {
		v := archive.DefaultCreatedAt()
		ac.mutation.SetCreatedAt(v)
	}
	if _, ok := ac.mutation.UpdatedAt(); !ok {
		v := archive.DefaultUpdatedAt()
		ac.mutation.SetUpdatedAt(v)
	}
	if _, ok := ac.mutation.WillBePruned(); !ok {
		v := archive.DefaultWillBePruned
		ac.mutation.SetWillBePruned(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (ac *ArchiveCreate) check() error {
	if _, ok := ac.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`ent: missing required field "Archive.created_at"`)}
	}
	if _, ok := ac.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`ent: missing required field "Archive.updated_at"`)}
	}
	if _, ok := ac.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "Archive.name"`)}
	}
	if _, ok := ac.mutation.Duration(); !ok {
		return &ValidationError{Name: "duration", err: errors.New(`ent: missing required field "Archive.duration"`)}
	}
	if _, ok := ac.mutation.BorgID(); !ok {
		return &ValidationError{Name: "borg_id", err: errors.New(`ent: missing required field "Archive.borg_id"`)}
	}
	if _, ok := ac.mutation.WillBePruned(); !ok {
		return &ValidationError{Name: "will_be_pruned", err: errors.New(`ent: missing required field "Archive.will_be_pruned"`)}
	}
	if len(ac.mutation.RepositoryIDs()) == 0 {
		return &ValidationError{Name: "repository", err: errors.New(`ent: missing required edge "Archive.repository"`)}
	}
	return nil
}

func (ac *ArchiveCreate) sqlSave(ctx context.Context) (*Archive, error) {
	if err := ac.check(); err != nil {
		return nil, err
	}
	_node, _spec := ac.createSpec()
	if err := sqlgraph.CreateNode(ctx, ac.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != _node.ID {
		id := _spec.ID.Value.(int64)
		_node.ID = int(id)
	}
	ac.mutation.id = &_node.ID
	ac.mutation.done = true
	return _node, nil
}

func (ac *ArchiveCreate) createSpec() (*Archive, *sqlgraph.CreateSpec) {
	var (
		_node = &Archive{config: ac.config}
		_spec = sqlgraph.NewCreateSpec(archive.Table, sqlgraph.NewFieldSpec(archive.FieldID, field.TypeInt))
	)
	if id, ok := ac.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := ac.mutation.CreatedAt(); ok {
		_spec.SetField(archive.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := ac.mutation.UpdatedAt(); ok {
		_spec.SetField(archive.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := ac.mutation.Name(); ok {
		_spec.SetField(archive.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := ac.mutation.Duration(); ok {
		_spec.SetField(archive.FieldDuration, field.TypeFloat64, value)
		_node.Duration = value
	}
	if value, ok := ac.mutation.BorgID(); ok {
		_spec.SetField(archive.FieldBorgID, field.TypeString, value)
		_node.BorgID = value
	}
	if value, ok := ac.mutation.WillBePruned(); ok {
		_spec.SetField(archive.FieldWillBePruned, field.TypeBool, value)
		_node.WillBePruned = value
	}
	if nodes := ac.mutation.RepositoryIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   archive.RepositoryTable,
			Columns: []string{archive.RepositoryColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(repository.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.archive_repository = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ac.mutation.BackupProfileIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   archive.BackupProfileTable,
			Columns: []string{archive.BackupProfileColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(backupprofile.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.archive_backup_profile = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// ArchiveCreateBulk is the builder for creating many Archive entities in bulk.
type ArchiveCreateBulk struct {
	config
	err      error
	builders []*ArchiveCreate
}

// Save creates the Archive entities in the database.
func (acb *ArchiveCreateBulk) Save(ctx context.Context) ([]*Archive, error) {
	if acb.err != nil {
		return nil, acb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(acb.builders))
	nodes := make([]*Archive, len(acb.builders))
	mutators := make([]Mutator, len(acb.builders))
	for i := range acb.builders {
		func(i int, root context.Context) {
			builder := acb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*ArchiveMutation)
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
					_, err = mutators[i+1].Mutate(root, acb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, acb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, acb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (acb *ArchiveCreateBulk) SaveX(ctx context.Context) []*Archive {
	v, err := acb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (acb *ArchiveCreateBulk) Exec(ctx context.Context) error {
	_, err := acb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (acb *ArchiveCreateBulk) ExecX(ctx context.Context) {
	if err := acb.Exec(ctx); err != nil {
		panic(err)
	}
}
