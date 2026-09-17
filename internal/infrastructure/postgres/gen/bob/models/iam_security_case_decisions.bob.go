// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
	"io"
	"time"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/expr"
)

// IamSecurityCaseDecision is an object representing the database table.
type IamSecurityCaseDecision struct {
	ID          string    `db:"id,pk" `
	ProjectID   string    `db:"project_id" `
	Environment string    `db:"environment" `
	CaseID      string    `db:"case_id" `
	ActorID     string    `db:"actor_id" `
	Action      string    `db:"action" `
	Evidence    string    `db:"evidence" `
	CreatedAt   time.Time `db:"created_at" `
}

// IamSecurityCaseDecisionSlice is an alias for a slice of pointers to IamSecurityCaseDecision.
// This should almost always be used instead of []*IamSecurityCaseDecision.
type IamSecurityCaseDecisionSlice []*IamSecurityCaseDecision

// IamSecurityCaseDecisions contains methods to work with the iam_security_case_decisions table
var IamSecurityCaseDecisions = psql.NewTablex[*IamSecurityCaseDecision, IamSecurityCaseDecisionSlice, *IamSecurityCaseDecisionSetter]("", "iam_security_case_decisions", buildIamSecurityCaseDecisionColumns("iam_security_case_decisions"))

// IamSecurityCaseDecisionsQuery is a query on the iam_security_case_decisions table
type IamSecurityCaseDecisionsQuery = *psql.ViewQuery[*IamSecurityCaseDecision, IamSecurityCaseDecisionSlice]

func buildIamSecurityCaseDecisionColumns(tableName string) iamSecurityCaseDecisionColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "case_id", "actor_id", "action", "evidence", "created_at",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSecurityCaseDecisionColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamSecurityCaseDecisionColumn(tableName, "id"),
		ProjectID:   buildIamSecurityCaseDecisionColumn(tableName, "project_id"),
		Environment: buildIamSecurityCaseDecisionColumn(tableName, "environment"),
		CaseID:      buildIamSecurityCaseDecisionColumn(tableName, "case_id"),
		ActorID:     buildIamSecurityCaseDecisionColumn(tableName, "actor_id"),
		Action:      buildIamSecurityCaseDecisionColumn(tableName, "action"),
		Evidence:    buildIamSecurityCaseDecisionColumn(tableName, "evidence"),
		CreatedAt:   buildIamSecurityCaseDecisionColumn(tableName, "created_at"),
	}
}

type iamSecurityCaseDecisionColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamSecurityCaseDecisionColumn
	ProjectID   iamSecurityCaseDecisionColumn
	Environment iamSecurityCaseDecisionColumn
	CaseID      iamSecurityCaseDecisionColumn
	ActorID     iamSecurityCaseDecisionColumn
	Action      iamSecurityCaseDecisionColumn
	Evidence    iamSecurityCaseDecisionColumn
	CreatedAt   iamSecurityCaseDecisionColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSecurityCaseDecisionColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSecurityCaseDecisionColumns) AliasedAs(tableName string) iamSecurityCaseDecisionColumns {
	return buildIamSecurityCaseDecisionColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSecurityCaseDecisionColumns) Unqualified() iamSecurityCaseDecisionColumns {
	return buildIamSecurityCaseDecisionColumns("")
}

func buildIamSecurityCaseDecisionColumn(alias, name string) iamSecurityCaseDecisionColumn {
	return iamSecurityCaseDecisionColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSecurityCaseDecisionColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSecurityCaseDecisionColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSecurityCaseDecisionColumn) ShouldOmitParens() bool {
	return true
}

// IamSecurityCaseDecisionSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSecurityCaseDecisionSetter struct {
	ID          *string    `db:"id,pk" `
	ProjectID   *string    `db:"project_id" `
	Environment *string    `db:"environment" `
	CaseID      *string    `db:"case_id" `
	ActorID     *string    `db:"actor_id" `
	Action      *string    `db:"action" `
	Evidence    *string    `db:"evidence" `
	CreatedAt   *time.Time `db:"created_at" `
}

func (s IamSecurityCaseDecisionSetter) SetColumns() []string {
	vals := make([]string, 0, 8)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.CaseID != nil {
		vals = append(vals, "case_id")
	}
	if s.ActorID != nil {
		vals = append(vals, "actor_id")
	}
	if s.Action != nil {
		vals = append(vals, "action")
	}
	if s.Evidence != nil {
		vals = append(vals, "evidence")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	return vals
}

func (s IamSecurityCaseDecisionSetter) Overwrite(t *IamSecurityCaseDecision) {
	if s.ID != nil {
		t.ID = func() string {
			if s.ID == nil {
				return *new(string)
			}
			return *s.ID
		}()
	}
	if s.ProjectID != nil {
		t.ProjectID = func() string {
			if s.ProjectID == nil {
				return *new(string)
			}
			return *s.ProjectID
		}()
	}
	if s.Environment != nil {
		t.Environment = func() string {
			if s.Environment == nil {
				return *new(string)
			}
			return *s.Environment
		}()
	}
	if s.CaseID != nil {
		t.CaseID = func() string {
			if s.CaseID == nil {
				return *new(string)
			}
			return *s.CaseID
		}()
	}
	if s.ActorID != nil {
		t.ActorID = func() string {
			if s.ActorID == nil {
				return *new(string)
			}
			return *s.ActorID
		}()
	}
	if s.Action != nil {
		t.Action = func() string {
			if s.Action == nil {
				return *new(string)
			}
			return *s.Action
		}()
	}
	if s.Evidence != nil {
		t.Evidence = func() string {
			if s.Evidence == nil {
				return *new(string)
			}
			return *s.Evidence
		}()
	}
	if s.CreatedAt != nil {
		t.CreatedAt = func() time.Time {
			if s.CreatedAt == nil {
				return *new(time.Time)
			}
			return *s.CreatedAt
		}()
	}
}

func (s *IamSecurityCaseDecisionSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSecurityCaseDecisions.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 8)
		if s.ID != nil {
			vals[0] = psql.Arg(func() string {
				if s.ID == nil {
					return *new(string)
				}
				return *s.ID
			}())
		} else {
			vals[0] = psql.Raw("DEFAULT")
		}

		if s.ProjectID != nil {
			vals[1] = psql.Arg(func() string {
				if s.ProjectID == nil {
					return *new(string)
				}
				return *s.ProjectID
			}())
		} else {
			vals[1] = psql.Raw("DEFAULT")
		}

		if s.Environment != nil {
			vals[2] = psql.Arg(func() string {
				if s.Environment == nil {
					return *new(string)
				}
				return *s.Environment
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.CaseID != nil {
			vals[3] = psql.Arg(func() string {
				if s.CaseID == nil {
					return *new(string)
				}
				return *s.CaseID
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.ActorID != nil {
			vals[4] = psql.Arg(func() string {
				if s.ActorID == nil {
					return *new(string)
				}
				return *s.ActorID
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.Action != nil {
			vals[5] = psql.Arg(func() string {
				if s.Action == nil {
					return *new(string)
				}
				return *s.Action
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.Evidence != nil {
			vals[6] = psql.Arg(func() string {
				if s.Evidence == nil {
					return *new(string)
				}
				return *s.Evidence
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[7] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamSecurityCaseDecisionSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSecurityCaseDecisionSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 8)

	if s.ID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "id")...),
			psql.Arg(s.ID),
		}})
	}

	if s.ProjectID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "project_id")...),
			psql.Arg(s.ProjectID),
		}})
	}

	if s.Environment != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "environment")...),
			psql.Arg(s.Environment),
		}})
	}

	if s.CaseID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "case_id")...),
			psql.Arg(s.CaseID),
		}})
	}

	if s.ActorID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "actor_id")...),
			psql.Arg(s.ActorID),
		}})
	}

	if s.Action != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "action")...),
			psql.Arg(s.Action),
		}})
	}

	if s.Evidence != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "evidence")...),
			psql.Arg(s.Evidence),
		}})
	}

	if s.CreatedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "created_at")...),
			psql.Arg(s.CreatedAt),
		}})
	}

	return exprs
}

// FindIamSecurityCaseDecision retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSecurityCaseDecision(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamSecurityCaseDecision, error) {
	if len(cols) == 0 {
		return IamSecurityCaseDecisions.Query(
			sm.Where(IamSecurityCaseDecisions.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamSecurityCaseDecisions.Query(
		sm.Where(IamSecurityCaseDecisions.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamSecurityCaseDecisions.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSecurityCaseDecisionExists checks the presence of a single record by primary key
func IamSecurityCaseDecisionExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamSecurityCaseDecisions.Query(
		sm.Where(IamSecurityCaseDecisions.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSecurityCaseDecision is retrieved from the database
func (o *IamSecurityCaseDecision) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityCaseDecisions.AfterSelectHooks.RunHooks(ctx, exec, IamSecurityCaseDecisionSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityCaseDecisions.AfterInsertHooks.RunHooks(ctx, exec, IamSecurityCaseDecisionSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityCaseDecisions.AfterUpdateHooks.RunHooks(ctx, exec, IamSecurityCaseDecisionSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityCaseDecisions.AfterDeleteHooks.RunHooks(ctx, exec, IamSecurityCaseDecisionSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityCaseDecisions.AfterMergeHooks.RunHooks(ctx, exec, IamSecurityCaseDecisionSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSecurityCaseDecision
func (o *IamSecurityCaseDecision) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamSecurityCaseDecision) pkEQ() dialect.Expression {
	return psql.Quote("iam_security_case_decisions", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSecurityCaseDecision
func (o *IamSecurityCaseDecision) Update(ctx context.Context, exec bob.Executor, s *IamSecurityCaseDecisionSetter) error {
	v, err := IamSecurityCaseDecisions.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSecurityCaseDecision record with an executor
func (o *IamSecurityCaseDecision) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSecurityCaseDecisions.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSecurityCaseDecision using the executor
func (o *IamSecurityCaseDecision) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSecurityCaseDecisions.Query(
		sm.Where(IamSecurityCaseDecisions.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSecurityCaseDecisionSlice is retrieved from the database
func (o IamSecurityCaseDecisionSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityCaseDecisions.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityCaseDecisions.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityCaseDecisions.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityCaseDecisions.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityCaseDecisions.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSecurityCaseDecisionSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_security_case_decisions", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		pkPairs := make([]bob.Expression, len(o))
		for i, row := range o {
			pkPairs[i] = row.primaryKeyVals()
		}
		return bob.ExpressSlice(ctx, w, d, start, pkPairs, "", ", ", "")
	}))
}

// copyMatchingRows finds models in the given slice that have the same primary key
// then it first copies the existing relationships from the old model to the new model
// and then replaces the old model in the slice with the new model
func (o IamSecurityCaseDecisionSlice) copyMatchingRows(from ...*IamSecurityCaseDecision) {
	for i, old := range o {
		for _, new := range from {
			if new.ID != old.ID {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamSecurityCaseDecisionSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityCaseDecisions.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityCaseDecision:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityCaseDecision:
				o.copyMatchingRows(retrieved...)
			case IamSecurityCaseDecisionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityCaseDecision or a slice of IamSecurityCaseDecision
				// then run the AfterUpdateHooks on the slice
				_, err = IamSecurityCaseDecisions.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSecurityCaseDecisionSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityCaseDecisions.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityCaseDecision:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityCaseDecision:
				o.copyMatchingRows(retrieved...)
			case IamSecurityCaseDecisionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityCaseDecision or a slice of IamSecurityCaseDecision
				// then run the AfterDeleteHooks on the slice
				_, err = IamSecurityCaseDecisions.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSecurityCaseDecisionSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityCaseDecisions.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityCaseDecision:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityCaseDecision:
				o.copyMatchingRows(retrieved...)
			case IamSecurityCaseDecisionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityCaseDecision or a slice of IamSecurityCaseDecision
				// then run the AfterMergeHooks on the slice
				_, err = IamSecurityCaseDecisions.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSecurityCaseDecisionSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSecurityCaseDecisionSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityCaseDecisions.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSecurityCaseDecisionSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityCaseDecisions.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSecurityCaseDecisionSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSecurityCaseDecisions.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSecurityCaseDecisionWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	CaseID      psql.WhereMod[Q, string]
	ActorID     psql.WhereMod[Q, string]
	Action      psql.WhereMod[Q, string]
	Evidence    psql.WhereMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
}

func (iamSecurityCaseDecisionWhere[Q]) AliasedAs(alias string) iamSecurityCaseDecisionWhere[Q] {
	return buildIamSecurityCaseDecisionWhere[Q](buildIamSecurityCaseDecisionColumns(alias))
}

func buildIamSecurityCaseDecisionWhere[Q psql.Filterable](cols iamSecurityCaseDecisionColumns) iamSecurityCaseDecisionWhere[Q] {
	return iamSecurityCaseDecisionWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		CaseID:      psql.Where[Q, string](cols.CaseID.Expression),
		ActorID:     psql.Where[Q, string](cols.ActorID.Expression),
		Action:      psql.Where[Q, string](cols.Action.Expression),
		Evidence:    psql.Where[Q, string](cols.Evidence.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
	}
}
