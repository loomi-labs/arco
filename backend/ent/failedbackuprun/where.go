// Code generated by ent, DO NOT EDIT.

package failedbackuprun

import (
	"arco/backend/ent/predicate"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

// ID filters vertices based on their ID field.
func ID(id int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id int) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldLTE(FieldID, id))
}

// Error applies equality check predicate on the "error" field. It's identical to ErrorEQ.
func Error(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldEQ(FieldError, v))
}

// ErrorEQ applies the EQ predicate on the "error" field.
func ErrorEQ(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldEQ(FieldError, v))
}

// ErrorNEQ applies the NEQ predicate on the "error" field.
func ErrorNEQ(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldNEQ(FieldError, v))
}

// ErrorIn applies the In predicate on the "error" field.
func ErrorIn(vs ...string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldIn(FieldError, vs...))
}

// ErrorNotIn applies the NotIn predicate on the "error" field.
func ErrorNotIn(vs ...string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldNotIn(FieldError, vs...))
}

// ErrorGT applies the GT predicate on the "error" field.
func ErrorGT(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldGT(FieldError, v))
}

// ErrorGTE applies the GTE predicate on the "error" field.
func ErrorGTE(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldGTE(FieldError, v))
}

// ErrorLT applies the LT predicate on the "error" field.
func ErrorLT(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldLT(FieldError, v))
}

// ErrorLTE applies the LTE predicate on the "error" field.
func ErrorLTE(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldLTE(FieldError, v))
}

// ErrorContains applies the Contains predicate on the "error" field.
func ErrorContains(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldContains(FieldError, v))
}

// ErrorHasPrefix applies the HasPrefix predicate on the "error" field.
func ErrorHasPrefix(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldHasPrefix(FieldError, v))
}

// ErrorHasSuffix applies the HasSuffix predicate on the "error" field.
func ErrorHasSuffix(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldHasSuffix(FieldError, v))
}

// ErrorEqualFold applies the EqualFold predicate on the "error" field.
func ErrorEqualFold(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldEqualFold(FieldError, v))
}

// ErrorContainsFold applies the ContainsFold predicate on the "error" field.
func ErrorContainsFold(v string) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.FieldContainsFold(FieldError, v))
}

// HasBackupProfile applies the HasEdge predicate on the "backup_profile" edge.
func HasBackupProfile() predicate.FailedBackupRun {
	return predicate.FailedBackupRun(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, BackupProfileTable, BackupProfileColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBackupProfileWith applies the HasEdge predicate on the "backup_profile" edge with a given conditions (other predicates).
func HasBackupProfileWith(preds ...predicate.BackupProfile) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(func(s *sql.Selector) {
		step := newBackupProfileStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasRepository applies the HasEdge predicate on the "repository" edge.
func HasRepository() predicate.FailedBackupRun {
	return predicate.FailedBackupRun(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, RepositoryTable, RepositoryColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasRepositoryWith applies the HasEdge predicate on the "repository" edge with a given conditions (other predicates).
func HasRepositoryWith(preds ...predicate.Repository) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(func(s *sql.Selector) {
		step := newRepositoryStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.FailedBackupRun) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.FailedBackupRun) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.FailedBackupRun) predicate.FailedBackupRun {
	return predicate.FailedBackupRun(sql.NotPredicates(p))
}