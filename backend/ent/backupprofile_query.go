// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"database/sql/driver"
	"fmt"
	"math"
	"timebender/backend/ent/backupprofile"
	"timebender/backend/ent/predicate"
	"timebender/backend/ent/repository"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
)

// BackupProfileQuery is the builder for querying BackupProfile entities.
type BackupProfileQuery struct {
	config
	ctx              *QueryContext
	order            []backupprofile.OrderOption
	inters           []Interceptor
	predicates       []predicate.BackupProfile
	withRepositories *RepositoryQuery
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the BackupProfileQuery builder.
func (bpq *BackupProfileQuery) Where(ps ...predicate.BackupProfile) *BackupProfileQuery {
	bpq.predicates = append(bpq.predicates, ps...)
	return bpq
}

// Limit the number of records to be returned by this query.
func (bpq *BackupProfileQuery) Limit(limit int) *BackupProfileQuery {
	bpq.ctx.Limit = &limit
	return bpq
}

// Offset to start from.
func (bpq *BackupProfileQuery) Offset(offset int) *BackupProfileQuery {
	bpq.ctx.Offset = &offset
	return bpq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (bpq *BackupProfileQuery) Unique(unique bool) *BackupProfileQuery {
	bpq.ctx.Unique = &unique
	return bpq
}

// Order specifies how the records should be ordered.
func (bpq *BackupProfileQuery) Order(o ...backupprofile.OrderOption) *BackupProfileQuery {
	bpq.order = append(bpq.order, o...)
	return bpq
}

// QueryRepositories chains the current query on the "repositories" edge.
func (bpq *BackupProfileQuery) QueryRepositories() *RepositoryQuery {
	query := (&RepositoryClient{config: bpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := bpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := bpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(backupprofile.Table, backupprofile.FieldID, selector),
			sqlgraph.To(repository.Table, repository.FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, backupprofile.RepositoriesTable, backupprofile.RepositoriesPrimaryKey...),
		)
		fromU = sqlgraph.SetNeighbors(bpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first BackupProfile entity from the query.
// Returns a *NotFoundError when no BackupProfile was found.
func (bpq *BackupProfileQuery) First(ctx context.Context) (*BackupProfile, error) {
	nodes, err := bpq.Limit(1).All(setContextOp(ctx, bpq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{backupprofile.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (bpq *BackupProfileQuery) FirstX(ctx context.Context) *BackupProfile {
	node, err := bpq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first BackupProfile ID from the query.
// Returns a *NotFoundError when no BackupProfile ID was found.
func (bpq *BackupProfileQuery) FirstID(ctx context.Context) (id int, err error) {
	var ids []int
	if ids, err = bpq.Limit(1).IDs(setContextOp(ctx, bpq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{backupprofile.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (bpq *BackupProfileQuery) FirstIDX(ctx context.Context) int {
	id, err := bpq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single BackupProfile entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one BackupProfile entity is found.
// Returns a *NotFoundError when no BackupProfile entities are found.
func (bpq *BackupProfileQuery) Only(ctx context.Context) (*BackupProfile, error) {
	nodes, err := bpq.Limit(2).All(setContextOp(ctx, bpq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{backupprofile.Label}
	default:
		return nil, &NotSingularError{backupprofile.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (bpq *BackupProfileQuery) OnlyX(ctx context.Context) *BackupProfile {
	node, err := bpq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only BackupProfile ID in the query.
// Returns a *NotSingularError when more than one BackupProfile ID is found.
// Returns a *NotFoundError when no entities are found.
func (bpq *BackupProfileQuery) OnlyID(ctx context.Context) (id int, err error) {
	var ids []int
	if ids, err = bpq.Limit(2).IDs(setContextOp(ctx, bpq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{backupprofile.Label}
	default:
		err = &NotSingularError{backupprofile.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (bpq *BackupProfileQuery) OnlyIDX(ctx context.Context) int {
	id, err := bpq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of BackupProfiles.
func (bpq *BackupProfileQuery) All(ctx context.Context) ([]*BackupProfile, error) {
	ctx = setContextOp(ctx, bpq.ctx, "All")
	if err := bpq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*BackupProfile, *BackupProfileQuery]()
	return withInterceptors[[]*BackupProfile](ctx, bpq, qr, bpq.inters)
}

// AllX is like All, but panics if an error occurs.
func (bpq *BackupProfileQuery) AllX(ctx context.Context) []*BackupProfile {
	nodes, err := bpq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of BackupProfile IDs.
func (bpq *BackupProfileQuery) IDs(ctx context.Context) (ids []int, err error) {
	if bpq.ctx.Unique == nil && bpq.path != nil {
		bpq.Unique(true)
	}
	ctx = setContextOp(ctx, bpq.ctx, "IDs")
	if err = bpq.Select(backupprofile.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (bpq *BackupProfileQuery) IDsX(ctx context.Context) []int {
	ids, err := bpq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (bpq *BackupProfileQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, bpq.ctx, "Count")
	if err := bpq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, bpq, querierCount[*BackupProfileQuery](), bpq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (bpq *BackupProfileQuery) CountX(ctx context.Context) int {
	count, err := bpq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (bpq *BackupProfileQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, bpq.ctx, "Exist")
	switch _, err := bpq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (bpq *BackupProfileQuery) ExistX(ctx context.Context) bool {
	exist, err := bpq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the BackupProfileQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (bpq *BackupProfileQuery) Clone() *BackupProfileQuery {
	if bpq == nil {
		return nil
	}
	return &BackupProfileQuery{
		config:           bpq.config,
		ctx:              bpq.ctx.Clone(),
		order:            append([]backupprofile.OrderOption{}, bpq.order...),
		inters:           append([]Interceptor{}, bpq.inters...),
		predicates:       append([]predicate.BackupProfile{}, bpq.predicates...),
		withRepositories: bpq.withRepositories.Clone(),
		// clone intermediate query.
		sql:  bpq.sql.Clone(),
		path: bpq.path,
	}
}

// WithRepositories tells the query-builder to eager-load the nodes that are connected to
// the "repositories" edge. The optional arguments are used to configure the query builder of the edge.
func (bpq *BackupProfileQuery) WithRepositories(opts ...func(*RepositoryQuery)) *BackupProfileQuery {
	query := (&RepositoryClient{config: bpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	bpq.withRepositories = query
	return bpq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.BackupProfile.Query().
//		GroupBy(backupprofile.FieldName).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (bpq *BackupProfileQuery) GroupBy(field string, fields ...string) *BackupProfileGroupBy {
	bpq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &BackupProfileGroupBy{build: bpq}
	grbuild.flds = &bpq.ctx.Fields
	grbuild.label = backupprofile.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name"`
//	}
//
//	client.BackupProfile.Query().
//		Select(backupprofile.FieldName).
//		Scan(ctx, &v)
func (bpq *BackupProfileQuery) Select(fields ...string) *BackupProfileSelect {
	bpq.ctx.Fields = append(bpq.ctx.Fields, fields...)
	sbuild := &BackupProfileSelect{BackupProfileQuery: bpq}
	sbuild.label = backupprofile.Label
	sbuild.flds, sbuild.scan = &bpq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a BackupProfileSelect configured with the given aggregations.
func (bpq *BackupProfileQuery) Aggregate(fns ...AggregateFunc) *BackupProfileSelect {
	return bpq.Select().Aggregate(fns...)
}

func (bpq *BackupProfileQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range bpq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, bpq); err != nil {
				return err
			}
		}
	}
	for _, f := range bpq.ctx.Fields {
		if !backupprofile.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if bpq.path != nil {
		prev, err := bpq.path(ctx)
		if err != nil {
			return err
		}
		bpq.sql = prev
	}
	return nil
}

func (bpq *BackupProfileQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*BackupProfile, error) {
	var (
		nodes       = []*BackupProfile{}
		_spec       = bpq.querySpec()
		loadedTypes = [1]bool{
			bpq.withRepositories != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*BackupProfile).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &BackupProfile{config: bpq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, bpq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := bpq.withRepositories; query != nil {
		if err := bpq.loadRepositories(ctx, query, nodes,
			func(n *BackupProfile) { n.Edges.Repositories = []*Repository{} },
			func(n *BackupProfile, e *Repository) { n.Edges.Repositories = append(n.Edges.Repositories, e) }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (bpq *BackupProfileQuery) loadRepositories(ctx context.Context, query *RepositoryQuery, nodes []*BackupProfile, init func(*BackupProfile), assign func(*BackupProfile, *Repository)) error {
	edgeIDs := make([]driver.Value, len(nodes))
	byID := make(map[int]*BackupProfile)
	nids := make(map[int]map[*BackupProfile]struct{})
	for i, node := range nodes {
		edgeIDs[i] = node.ID
		byID[node.ID] = node
		if init != nil {
			init(node)
		}
	}
	query.Where(func(s *sql.Selector) {
		joinT := sql.Table(backupprofile.RepositoriesTable)
		s.Join(joinT).On(s.C(repository.FieldID), joinT.C(backupprofile.RepositoriesPrimaryKey[1]))
		s.Where(sql.InValues(joinT.C(backupprofile.RepositoriesPrimaryKey[0]), edgeIDs...))
		columns := s.SelectedColumns()
		s.Select(joinT.C(backupprofile.RepositoriesPrimaryKey[0]))
		s.AppendSelect(columns...)
		s.SetDistinct(false)
	})
	if err := query.prepareQuery(ctx); err != nil {
		return err
	}
	qr := QuerierFunc(func(ctx context.Context, q Query) (Value, error) {
		return query.sqlAll(ctx, func(_ context.Context, spec *sqlgraph.QuerySpec) {
			assign := spec.Assign
			values := spec.ScanValues
			spec.ScanValues = func(columns []string) ([]any, error) {
				values, err := values(columns[1:])
				if err != nil {
					return nil, err
				}
				return append([]any{new(sql.NullInt64)}, values...), nil
			}
			spec.Assign = func(columns []string, values []any) error {
				outValue := int(values[0].(*sql.NullInt64).Int64)
				inValue := int(values[1].(*sql.NullInt64).Int64)
				if nids[inValue] == nil {
					nids[inValue] = map[*BackupProfile]struct{}{byID[outValue]: {}}
					return assign(columns[1:], values[1:])
				}
				nids[inValue][byID[outValue]] = struct{}{}
				return nil
			}
		})
	})
	neighbors, err := withInterceptors[[]*Repository](ctx, query, qr, query.inters)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected "repositories" node returned %v`, n.ID)
		}
		for kn := range nodes {
			assign(kn, n)
		}
	}
	return nil
}

func (bpq *BackupProfileQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := bpq.querySpec()
	_spec.Node.Columns = bpq.ctx.Fields
	if len(bpq.ctx.Fields) > 0 {
		_spec.Unique = bpq.ctx.Unique != nil && *bpq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, bpq.driver, _spec)
}

func (bpq *BackupProfileQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(backupprofile.Table, backupprofile.Columns, sqlgraph.NewFieldSpec(backupprofile.FieldID, field.TypeInt))
	_spec.From = bpq.sql
	if unique := bpq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if bpq.path != nil {
		_spec.Unique = true
	}
	if fields := bpq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, backupprofile.FieldID)
		for i := range fields {
			if fields[i] != backupprofile.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := bpq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := bpq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := bpq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := bpq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (bpq *BackupProfileQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(bpq.driver.Dialect())
	t1 := builder.Table(backupprofile.Table)
	columns := bpq.ctx.Fields
	if len(columns) == 0 {
		columns = backupprofile.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if bpq.sql != nil {
		selector = bpq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if bpq.ctx.Unique != nil && *bpq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range bpq.predicates {
		p(selector)
	}
	for _, p := range bpq.order {
		p(selector)
	}
	if offset := bpq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := bpq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// BackupProfileGroupBy is the group-by builder for BackupProfile entities.
type BackupProfileGroupBy struct {
	selector
	build *BackupProfileQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (bpgb *BackupProfileGroupBy) Aggregate(fns ...AggregateFunc) *BackupProfileGroupBy {
	bpgb.fns = append(bpgb.fns, fns...)
	return bpgb
}

// Scan applies the selector query and scans the result into the given value.
func (bpgb *BackupProfileGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bpgb.build.ctx, "GroupBy")
	if err := bpgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BackupProfileQuery, *BackupProfileGroupBy](ctx, bpgb.build, bpgb, bpgb.build.inters, v)
}

func (bpgb *BackupProfileGroupBy) sqlScan(ctx context.Context, root *BackupProfileQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(bpgb.fns))
	for _, fn := range bpgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*bpgb.flds)+len(bpgb.fns))
		for _, f := range *bpgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*bpgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bpgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// BackupProfileSelect is the builder for selecting fields of BackupProfile entities.
type BackupProfileSelect struct {
	*BackupProfileQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (bps *BackupProfileSelect) Aggregate(fns ...AggregateFunc) *BackupProfileSelect {
	bps.fns = append(bps.fns, fns...)
	return bps
}

// Scan applies the selector query and scans the result into the given value.
func (bps *BackupProfileSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bps.ctx, "Select")
	if err := bps.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BackupProfileQuery, *BackupProfileSelect](ctx, bps.BackupProfileQuery, bps, bps.inters, v)
}

func (bps *BackupProfileSelect) sqlScan(ctx context.Context, root *BackupProfileQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(bps.fns))
	for _, fn := range bps.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*bps.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bps.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}