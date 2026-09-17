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

// IamSecurityGuard is an object representing the database table.
type IamSecurityGuard struct {
	FlowID      string    `db:"flow_id" `
	ProjectID   string    `db:"project_id,pk" `
	Environment string    `db:"environment,pk" `
	UserID      string    `db:"user_id,pk" `
	CreatedAt   time.Time `db:"created_at" `
}

// IamSecurityGuardSlice is an alias for a slice of pointers to IamSecurityGuard.
// This should almost always be used instead of []*IamSecurityGuard.
type IamSecurityGuardSlice []*IamSecurityGuard

// IamSecurityGuards contains methods to work with the iam_security_guards table
var IamSecurityGuards = psql.NewTablex[*IamSecurityGuard, IamSecurityGuardSlice, *IamSecurityGuardSetter]("", "iam_security_guards", buildIamSecurityGuardColumns("iam_security_guards"))

// IamSecurityGuardsQuery is a query on the iam_security_guards table
type IamSecurityGuardsQuery = *psql.ViewQuery[*IamSecurityGuard, IamSecurityGuardSlice]

func buildIamSecurityGuardColumns(tableName string) iamSecurityGuardColumns {
	columnsExpr := expr.NewColumnsExpr(
		"flow_id", "project_id", "environment", "user_id", "created_at",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSecurityGuardColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		FlowID:      buildIamSecurityGuardColumn(tableName, "flow_id"),
		ProjectID:   buildIamSecurityGuardColumn(tableName, "project_id"),
		Environment: buildIamSecurityGuardColumn(tableName, "environment"),
		UserID:      buildIamSecurityGuardColumn(tableName, "user_id"),
		CreatedAt:   buildIamSecurityGuardColumn(tableName, "created_at"),
	}
}

type iamSecurityGuardColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	FlowID      iamSecurityGuardColumn
	ProjectID   iamSecurityGuardColumn
	Environment iamSecurityGuardColumn
	UserID      iamSecurityGuardColumn
	CreatedAt   iamSecurityGuardColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSecurityGuardColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSecurityGuardColumns) AliasedAs(tableName string) iamSecurityGuardColumns {
	return buildIamSecurityGuardColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSecurityGuardColumns) Unqualified() iamSecurityGuardColumns {
	return buildIamSecurityGuardColumns("")
}

func buildIamSecurityGuardColumn(alias, name string) iamSecurityGuardColumn {
	return iamSecurityGuardColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSecurityGuardColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSecurityGuardColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSecurityGuardColumn) ShouldOmitParens() bool {
	return true
}

// IamSecurityGuardSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSecurityGuardSetter struct {
	FlowID      *string    `db:"flow_id" `
	ProjectID   *string    `db:"project_id,pk" `
	Environment *string    `db:"environment,pk" `
	UserID      *string    `db:"user_id,pk" `
	CreatedAt   *time.Time `db:"created_at" `
}

func (s IamSecurityGuardSetter) SetColumns() []string {
	vals := make([]string, 0, 5)
	if s.FlowID != nil {
		vals = append(vals, "flow_id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	return vals
}

func (s IamSecurityGuardSetter) Overwrite(t *IamSecurityGuard) {
	if s.FlowID != nil {
		t.FlowID = func() string {
			if s.FlowID == nil {
				return *new(string)
			}
			return *s.FlowID
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
	if s.UserID != nil {
		t.UserID = func() string {
			if s.UserID == nil {
				return *new(string)
			}
			return *s.UserID
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

func (s *IamSecurityGuardSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSecurityGuards.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 5)
		if s.FlowID != nil {
			vals[0] = psql.Arg(func() string {
				if s.FlowID == nil {
					return *new(string)
				}
				return *s.FlowID
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

		if s.UserID != nil {
			vals[3] = psql.Arg(func() string {
				if s.UserID == nil {
					return *new(string)
				}
				return *s.UserID
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[4] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamSecurityGuardSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSecurityGuardSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 5)

	if s.FlowID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "flow_id")...),
			psql.Arg(s.FlowID),
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

	if s.UserID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_id")...),
			psql.Arg(s.UserID),
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

// FindIamSecurityGuard retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSecurityGuard(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, UserIDPK string, cols ...string) (*IamSecurityGuard, error) {
	if len(cols) == 0 {
		return IamSecurityGuards.Query(
			sm.Where(IamSecurityGuards.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
			sm.Where(IamSecurityGuards.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
			sm.Where(IamSecurityGuards.Columns.UserID.EQ(psql.Arg(UserIDPK))),
		).One(ctx, exec)
	}

	return IamSecurityGuards.Query(
		sm.Where(IamSecurityGuards.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamSecurityGuards.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamSecurityGuards.Columns.UserID.EQ(psql.Arg(UserIDPK))),
		sm.Columns(IamSecurityGuards.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSecurityGuardExists checks the presence of a single record by primary key
func IamSecurityGuardExists(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, UserIDPK string) (bool, error) {
	return IamSecurityGuards.Query(
		sm.Where(IamSecurityGuards.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamSecurityGuards.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamSecurityGuards.Columns.UserID.EQ(psql.Arg(UserIDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSecurityGuard is retrieved from the database
func (o *IamSecurityGuard) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityGuards.AfterSelectHooks.RunHooks(ctx, exec, IamSecurityGuardSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityGuards.AfterInsertHooks.RunHooks(ctx, exec, IamSecurityGuardSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityGuards.AfterUpdateHooks.RunHooks(ctx, exec, IamSecurityGuardSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityGuards.AfterDeleteHooks.RunHooks(ctx, exec, IamSecurityGuardSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityGuards.AfterMergeHooks.RunHooks(ctx, exec, IamSecurityGuardSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSecurityGuard
func (o *IamSecurityGuard) primaryKeyVals() bob.Expression {
	return psql.ArgGroup(
		o.ProjectID,
		o.Environment,
		o.UserID,
	)
}

func (o *IamSecurityGuard) pkEQ() dialect.Expression {
	return psql.Group(psql.Quote("iam_security_guards", "project_id"), psql.Quote("iam_security_guards", "environment"), psql.Quote("iam_security_guards", "user_id")).EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSecurityGuard
func (o *IamSecurityGuard) Update(ctx context.Context, exec bob.Executor, s *IamSecurityGuardSetter) error {
	v, err := IamSecurityGuards.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSecurityGuard record with an executor
func (o *IamSecurityGuard) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSecurityGuards.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSecurityGuard using the executor
func (o *IamSecurityGuard) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSecurityGuards.Query(
		sm.Where(IamSecurityGuards.Columns.ProjectID.EQ(psql.Arg(o.ProjectID))),
		sm.Where(IamSecurityGuards.Columns.Environment.EQ(psql.Arg(o.Environment))),
		sm.Where(IamSecurityGuards.Columns.UserID.EQ(psql.Arg(o.UserID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSecurityGuardSlice is retrieved from the database
func (o IamSecurityGuardSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityGuards.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityGuards.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityGuards.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityGuards.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityGuards.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSecurityGuardSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Group(psql.Quote("iam_security_guards", "project_id"), psql.Quote("iam_security_guards", "environment"), psql.Quote("iam_security_guards", "user_id")).In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSecurityGuardSlice) copyMatchingRows(from ...*IamSecurityGuard) {
	for i, old := range o {
		for _, new := range from {
			if new.ProjectID != old.ProjectID {
				continue
			}
			if new.Environment != old.Environment {
				continue
			}
			if new.UserID != old.UserID {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamSecurityGuardSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityGuards.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityGuard:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityGuard:
				o.copyMatchingRows(retrieved...)
			case IamSecurityGuardSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityGuard or a slice of IamSecurityGuard
				// then run the AfterUpdateHooks on the slice
				_, err = IamSecurityGuards.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSecurityGuardSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityGuards.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityGuard:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityGuard:
				o.copyMatchingRows(retrieved...)
			case IamSecurityGuardSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityGuard or a slice of IamSecurityGuard
				// then run the AfterDeleteHooks on the slice
				_, err = IamSecurityGuards.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSecurityGuardSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityGuards.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityGuard:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityGuard:
				o.copyMatchingRows(retrieved...)
			case IamSecurityGuardSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityGuard or a slice of IamSecurityGuard
				// then run the AfterMergeHooks on the slice
				_, err = IamSecurityGuards.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSecurityGuardSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSecurityGuardSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityGuards.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSecurityGuardSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityGuards.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSecurityGuardSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSecurityGuards.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSecurityGuardWhere[Q psql.Filterable] struct {
	FlowID      psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	UserID      psql.WhereMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
}

func (iamSecurityGuardWhere[Q]) AliasedAs(alias string) iamSecurityGuardWhere[Q] {
	return buildIamSecurityGuardWhere[Q](buildIamSecurityGuardColumns(alias))
}

func buildIamSecurityGuardWhere[Q psql.Filterable](cols iamSecurityGuardColumns) iamSecurityGuardWhere[Q] {
	return iamSecurityGuardWhere[Q]{
		FlowID:      psql.Where[Q, string](cols.FlowID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		UserID:      psql.Where[Q, string](cols.UserID.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
	}
}
