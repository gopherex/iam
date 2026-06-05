// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/expr"
)

// IamEnvironment is an object representing the database table.
type IamEnvironment struct {
	ProjectID string           `db:"project_id,pk" `
	Name      string           `db:"name,pk" `
	Issuer    null.Val[string] `db:"issuer" `
	CreatedAt time.Time        `db:"created_at" `
	Data      json.RawMessage  `db:"data" `
}

// IamEnvironmentSlice is an alias for a slice of pointers to IamEnvironment.
// This should almost always be used instead of []*IamEnvironment.
type IamEnvironmentSlice []*IamEnvironment

// IamEnvironments contains methods to work with the iam_environments table
var IamEnvironments = psql.NewTablex[*IamEnvironment, IamEnvironmentSlice, *IamEnvironmentSetter]("", "iam_environments", buildIamEnvironmentColumns("iam_environments"))

// IamEnvironmentsQuery is a query on the iam_environments table
type IamEnvironmentsQuery = *psql.ViewQuery[*IamEnvironment, IamEnvironmentSlice]

func buildIamEnvironmentColumns(tableName string) iamEnvironmentColumns {
	columnsExpr := expr.NewColumnsExpr(
		"project_id", "name", "issuer", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamEnvironmentColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ProjectID:   buildIamEnvironmentColumn(tableName, "project_id"),
		Name:        buildIamEnvironmentColumn(tableName, "name"),
		Issuer:      buildIamEnvironmentColumn(tableName, "issuer"),
		CreatedAt:   buildIamEnvironmentColumn(tableName, "created_at"),
		Data:        buildIamEnvironmentColumn(tableName, "data"),
	}
}

type iamEnvironmentColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ProjectID  iamEnvironmentColumn
	Name       iamEnvironmentColumn
	Issuer     iamEnvironmentColumn
	CreatedAt  iamEnvironmentColumn
	Data       iamEnvironmentColumn
}

// Alias returns the current table alias for the columns set.
func (c iamEnvironmentColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamEnvironmentColumns) AliasedAs(tableName string) iamEnvironmentColumns {
	return buildIamEnvironmentColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamEnvironmentColumns) Unqualified() iamEnvironmentColumns {
	return buildIamEnvironmentColumns("")
}

func buildIamEnvironmentColumn(alias, name string) iamEnvironmentColumn {
	return iamEnvironmentColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamEnvironmentColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamEnvironmentColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamEnvironmentColumn) ShouldOmitParens() bool {
	return true
}

// IamEnvironmentSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamEnvironmentSetter struct {
	ProjectID *string           `db:"project_id,pk" `
	Name      *string           `db:"name,pk" `
	Issuer    *null.Val[string] `db:"issuer" `
	CreatedAt *time.Time        `db:"created_at" `
	Data      *json.RawMessage  `db:"data" `
}

func (s IamEnvironmentSetter) SetColumns() []string {
	vals := make([]string, 0, 5)
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Name != nil {
		vals = append(vals, "name")
	}
	if s.Issuer != nil {
		vals = append(vals, "issuer")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamEnvironmentSetter) Overwrite(t *IamEnvironment) {
	if s.ProjectID != nil {
		t.ProjectID = func() string {
			if s.ProjectID == nil {
				return *new(string)
			}
			return *s.ProjectID
		}()
	}
	if s.Name != nil {
		t.Name = func() string {
			if s.Name == nil {
				return *new(string)
			}
			return *s.Name
		}()
	}
	if s.Issuer != nil {
		t.Issuer = func() null.Val[string] {
			if s.Issuer == nil {
				return *new(null.Val[string])
			}
			v := s.Issuer
			return *v
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
	if s.Data != nil {
		t.Data = func() json.RawMessage {
			if s.Data == nil {
				return *new(json.RawMessage)
			}
			return *s.Data
		}()
	}
}

func (s *IamEnvironmentSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamEnvironments.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 5)
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

		if s.Name != nil {
			vals[1] = psql.Arg(func() string {
				if s.Name == nil {
					return *new(string)
				}
				return *s.Name
			}())
		} else {
			vals[1] = psql.Raw("DEFAULT")
		}

		if s.Issuer != nil {
			vals[2] = psql.Arg(func() null.Val[string] {
				if s.Issuer == nil {
					return *new(null.Val[string])
				}
				v := s.Issuer
				return *v
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[3] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[4] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamEnvironmentSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamEnvironmentSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 5)

	if s.ProjectID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "project_id")...),
			psql.Arg(s.ProjectID),
		}})
	}

	if s.Name != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "name")...),
			psql.Arg(s.Name),
		}})
	}

	if s.Issuer != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "issuer")...),
			psql.Arg(s.Issuer),
		}})
	}

	if s.CreatedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "created_at")...),
			psql.Arg(s.CreatedAt),
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

// FindIamEnvironment retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamEnvironment(ctx context.Context, exec bob.Executor, ProjectIDPK string, NamePK string, cols ...string) (*IamEnvironment, error) {
	if len(cols) == 0 {
		return IamEnvironments.Query(
			sm.Where(IamEnvironments.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
			sm.Where(IamEnvironments.Columns.Name.EQ(psql.Arg(NamePK))),
		).One(ctx, exec)
	}

	return IamEnvironments.Query(
		sm.Where(IamEnvironments.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamEnvironments.Columns.Name.EQ(psql.Arg(NamePK))),
		sm.Columns(IamEnvironments.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamEnvironmentExists checks the presence of a single record by primary key
func IamEnvironmentExists(ctx context.Context, exec bob.Executor, ProjectIDPK string, NamePK string) (bool, error) {
	return IamEnvironments.Query(
		sm.Where(IamEnvironments.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamEnvironments.Columns.Name.EQ(psql.Arg(NamePK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamEnvironment is retrieved from the database
func (o *IamEnvironment) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamEnvironments.AfterSelectHooks.RunHooks(ctx, exec, IamEnvironmentSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamEnvironments.AfterInsertHooks.RunHooks(ctx, exec, IamEnvironmentSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamEnvironments.AfterUpdateHooks.RunHooks(ctx, exec, IamEnvironmentSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamEnvironments.AfterDeleteHooks.RunHooks(ctx, exec, IamEnvironmentSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamEnvironments.AfterMergeHooks.RunHooks(ctx, exec, IamEnvironmentSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamEnvironment
func (o *IamEnvironment) primaryKeyVals() bob.Expression {
	return psql.ArgGroup(
		o.ProjectID,
		o.Name,
	)
}

func (o *IamEnvironment) pkEQ() dialect.Expression {
	return psql.Group(psql.Quote("iam_environments", "project_id"), psql.Quote("iam_environments", "name")).EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamEnvironment
func (o *IamEnvironment) Update(ctx context.Context, exec bob.Executor, s *IamEnvironmentSetter) error {
	v, err := IamEnvironments.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamEnvironment record with an executor
func (o *IamEnvironment) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamEnvironments.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamEnvironment using the executor
func (o *IamEnvironment) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamEnvironments.Query(
		sm.Where(IamEnvironments.Columns.ProjectID.EQ(psql.Arg(o.ProjectID))),
		sm.Where(IamEnvironments.Columns.Name.EQ(psql.Arg(o.Name))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamEnvironmentSlice is retrieved from the database
func (o IamEnvironmentSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamEnvironments.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamEnvironments.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamEnvironments.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamEnvironments.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamEnvironments.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamEnvironmentSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Group(psql.Quote("iam_environments", "project_id"), psql.Quote("iam_environments", "name")).In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamEnvironmentSlice) copyMatchingRows(from ...*IamEnvironment) {
	for i, old := range o {
		for _, new := range from {
			if new.ProjectID != old.ProjectID {
				continue
			}
			if new.Name != old.Name {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamEnvironmentSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEnvironments.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEnvironment:
				o.copyMatchingRows(retrieved)
			case []*IamEnvironment:
				o.copyMatchingRows(retrieved...)
			case IamEnvironmentSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEnvironment or a slice of IamEnvironment
				// then run the AfterUpdateHooks on the slice
				_, err = IamEnvironments.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamEnvironmentSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEnvironments.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEnvironment:
				o.copyMatchingRows(retrieved)
			case []*IamEnvironment:
				o.copyMatchingRows(retrieved...)
			case IamEnvironmentSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEnvironment or a slice of IamEnvironment
				// then run the AfterDeleteHooks on the slice
				_, err = IamEnvironments.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamEnvironmentSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEnvironments.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEnvironment:
				o.copyMatchingRows(retrieved)
			case []*IamEnvironment:
				o.copyMatchingRows(retrieved...)
			case IamEnvironmentSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEnvironment or a slice of IamEnvironment
				// then run the AfterMergeHooks on the slice
				_, err = IamEnvironments.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamEnvironmentSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamEnvironmentSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamEnvironments.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamEnvironmentSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamEnvironments.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamEnvironmentSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamEnvironments.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamEnvironmentWhere[Q psql.Filterable] struct {
	ProjectID psql.WhereMod[Q, string]
	Name      psql.WhereMod[Q, string]
	Issuer    psql.WhereNullMod[Q, string]
	CreatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamEnvironmentWhere[Q]) AliasedAs(alias string) iamEnvironmentWhere[Q] {
	return buildIamEnvironmentWhere[Q](buildIamEnvironmentColumns(alias))
}

func buildIamEnvironmentWhere[Q psql.Filterable](cols iamEnvironmentColumns) iamEnvironmentWhere[Q] {
	return iamEnvironmentWhere[Q]{
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Name:      psql.Where[Q, string](cols.Name.Expression),
		Issuer:    psql.WhereNull[Q, string](cols.Issuer.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
