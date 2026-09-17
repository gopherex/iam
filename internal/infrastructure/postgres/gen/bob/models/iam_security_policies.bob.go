// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
	"encoding/json"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/expr"
)

// IamSecurityPolicy is an object representing the database table.
type IamSecurityPolicy struct {
	ProjectID   string          `db:"project_id,pk" `
	Environment string          `db:"environment,pk" `
	Data        json.RawMessage `db:"data" `
}

// IamSecurityPolicySlice is an alias for a slice of pointers to IamSecurityPolicy.
// This should almost always be used instead of []*IamSecurityPolicy.
type IamSecurityPolicySlice []*IamSecurityPolicy

// IamSecurityPolicies contains methods to work with the iam_security_policies table
var IamSecurityPolicies = psql.NewTablex[*IamSecurityPolicy, IamSecurityPolicySlice, *IamSecurityPolicySetter]("", "iam_security_policies", buildIamSecurityPolicyColumns("iam_security_policies"))

// IamSecurityPoliciesQuery is a query on the iam_security_policies table
type IamSecurityPoliciesQuery = *psql.ViewQuery[*IamSecurityPolicy, IamSecurityPolicySlice]

func buildIamSecurityPolicyColumns(tableName string) iamSecurityPolicyColumns {
	columnsExpr := expr.NewColumnsExpr(
		"project_id", "environment", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSecurityPolicyColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ProjectID:   buildIamSecurityPolicyColumn(tableName, "project_id"),
		Environment: buildIamSecurityPolicyColumn(tableName, "environment"),
		Data:        buildIamSecurityPolicyColumn(tableName, "data"),
	}
}

type iamSecurityPolicyColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ProjectID   iamSecurityPolicyColumn
	Environment iamSecurityPolicyColumn
	Data        iamSecurityPolicyColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSecurityPolicyColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSecurityPolicyColumns) AliasedAs(tableName string) iamSecurityPolicyColumns {
	return buildIamSecurityPolicyColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSecurityPolicyColumns) Unqualified() iamSecurityPolicyColumns {
	return buildIamSecurityPolicyColumns("")
}

func buildIamSecurityPolicyColumn(alias, name string) iamSecurityPolicyColumn {
	return iamSecurityPolicyColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSecurityPolicyColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSecurityPolicyColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSecurityPolicyColumn) ShouldOmitParens() bool {
	return true
}

// IamSecurityPolicySetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSecurityPolicySetter struct {
	ProjectID   *string          `db:"project_id,pk" `
	Environment *string          `db:"environment,pk" `
	Data        *json.RawMessage `db:"data" `
}

func (s IamSecurityPolicySetter) SetColumns() []string {
	vals := make([]string, 0, 3)
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamSecurityPolicySetter) Overwrite(t *IamSecurityPolicy) {
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
	if s.Data != nil {
		t.Data = func() json.RawMessage {
			if s.Data == nil {
				return *new(json.RawMessage)
			}
			return *s.Data
		}()
	}
}

func (s *IamSecurityPolicySetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSecurityPolicies.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 3)
		if s.ProjectID != nil {
			vals[0] = psql.Arg(func() string {
				if s.ProjectID == nil {
					return *new(string)
				}
				return *s.ProjectID
			}())
		} else {
			vals[0] = psql.Raw("DEFAULT")
		}

		if s.Environment != nil {
			vals[1] = psql.Arg(func() string {
				if s.Environment == nil {
					return *new(string)
				}
				return *s.Environment
			}())
		} else {
			vals[1] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[2] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamSecurityPolicySetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSecurityPolicySetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 3)

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

	if s.Data != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "data")...),
			psql.Arg(s.Data),
		}})
	}

	return exprs
}

// FindIamSecurityPolicy retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSecurityPolicy(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, cols ...string) (*IamSecurityPolicy, error) {
	if len(cols) == 0 {
		return IamSecurityPolicies.Query(
			sm.Where(IamSecurityPolicies.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
			sm.Where(IamSecurityPolicies.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		).One(ctx, exec)
	}

	return IamSecurityPolicies.Query(
		sm.Where(IamSecurityPolicies.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamSecurityPolicies.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Columns(IamSecurityPolicies.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSecurityPolicyExists checks the presence of a single record by primary key
func IamSecurityPolicyExists(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string) (bool, error) {
	return IamSecurityPolicies.Query(
		sm.Where(IamSecurityPolicies.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamSecurityPolicies.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSecurityPolicy is retrieved from the database
func (o *IamSecurityPolicy) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityPolicies.AfterSelectHooks.RunHooks(ctx, exec, IamSecurityPolicySlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityPolicies.AfterInsertHooks.RunHooks(ctx, exec, IamSecurityPolicySlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityPolicies.AfterUpdateHooks.RunHooks(ctx, exec, IamSecurityPolicySlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityPolicies.AfterDeleteHooks.RunHooks(ctx, exec, IamSecurityPolicySlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityPolicies.AfterMergeHooks.RunHooks(ctx, exec, IamSecurityPolicySlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSecurityPolicy
func (o *IamSecurityPolicy) primaryKeyVals() bob.Expression {
	return psql.ArgGroup(
		o.ProjectID,
		o.Environment,
	)
}

func (o *IamSecurityPolicy) pkEQ() dialect.Expression {
	return psql.Group(psql.Quote("iam_security_policies", "project_id"), psql.Quote("iam_security_policies", "environment")).EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSecurityPolicy
func (o *IamSecurityPolicy) Update(ctx context.Context, exec bob.Executor, s *IamSecurityPolicySetter) error {
	v, err := IamSecurityPolicies.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSecurityPolicy record with an executor
func (o *IamSecurityPolicy) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSecurityPolicies.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSecurityPolicy using the executor
func (o *IamSecurityPolicy) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSecurityPolicies.Query(
		sm.Where(IamSecurityPolicies.Columns.ProjectID.EQ(psql.Arg(o.ProjectID))),
		sm.Where(IamSecurityPolicies.Columns.Environment.EQ(psql.Arg(o.Environment))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSecurityPolicySlice is retrieved from the database
func (o IamSecurityPolicySlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityPolicies.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityPolicies.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityPolicies.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityPolicies.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityPolicies.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSecurityPolicySlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Group(psql.Quote("iam_security_policies", "project_id"), psql.Quote("iam_security_policies", "environment")).In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSecurityPolicySlice) copyMatchingRows(from ...*IamSecurityPolicy) {
	for i, old := range o {
		for _, new := range from {
			if new.ProjectID != old.ProjectID {
				continue
			}
			if new.Environment != old.Environment {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamSecurityPolicySlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityPolicies.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityPolicy:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityPolicy:
				o.copyMatchingRows(retrieved...)
			case IamSecurityPolicySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityPolicy or a slice of IamSecurityPolicy
				// then run the AfterUpdateHooks on the slice
				_, err = IamSecurityPolicies.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSecurityPolicySlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityPolicies.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityPolicy:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityPolicy:
				o.copyMatchingRows(retrieved...)
			case IamSecurityPolicySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityPolicy or a slice of IamSecurityPolicy
				// then run the AfterDeleteHooks on the slice
				_, err = IamSecurityPolicies.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSecurityPolicySlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityPolicies.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityPolicy:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityPolicy:
				o.copyMatchingRows(retrieved...)
			case IamSecurityPolicySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityPolicy or a slice of IamSecurityPolicy
				// then run the AfterMergeHooks on the slice
				_, err = IamSecurityPolicies.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSecurityPolicySlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSecurityPolicySetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityPolicies.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSecurityPolicySlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityPolicies.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSecurityPolicySlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSecurityPolicies.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSecurityPolicyWhere[Q psql.Filterable] struct {
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamSecurityPolicyWhere[Q]) AliasedAs(alias string) iamSecurityPolicyWhere[Q] {
	return buildIamSecurityPolicyWhere[Q](buildIamSecurityPolicyColumns(alias))
}

func buildIamSecurityPolicyWhere[Q psql.Filterable](cols iamSecurityPolicyColumns) iamSecurityPolicyWhere[Q] {
	return iamSecurityPolicyWhere[Q]{
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
