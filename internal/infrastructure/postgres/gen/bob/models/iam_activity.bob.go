// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
	"encoding/json"
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

// IamActivity is an object representing the database table.
type IamActivity struct {
	ID          string          `db:"id,pk" `
	ProjectID   string          `db:"project_id" `
	Environment string          `db:"environment" `
	UserID      string          `db:"user_id" `
	Type        string          `db:"type" `
	At          time.Time       `db:"at" `
	Data        json.RawMessage `db:"data" `
}

// IamActivitySlice is an alias for a slice of pointers to IamActivity.
// This should almost always be used instead of []*IamActivity.
type IamActivitySlice []*IamActivity

// IamActivities contains methods to work with the iam_activity table
var IamActivities = psql.NewTablex[*IamActivity, IamActivitySlice, *IamActivitySetter]("", "iam_activity", buildIamActivityColumns("iam_activity"))

// IamActivitiesQuery is a query on the iam_activity table
type IamActivitiesQuery = *psql.ViewQuery[*IamActivity, IamActivitySlice]

func buildIamActivityColumns(tableName string) iamActivityColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "user_id", "type", "at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamActivityColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamActivityColumn(tableName, "id"),
		ProjectID:   buildIamActivityColumn(tableName, "project_id"),
		Environment: buildIamActivityColumn(tableName, "environment"),
		UserID:      buildIamActivityColumn(tableName, "user_id"),
		Type:        buildIamActivityColumn(tableName, "type"),
		At:          buildIamActivityColumn(tableName, "at"),
		Data:        buildIamActivityColumn(tableName, "data"),
	}
}

type iamActivityColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamActivityColumn
	ProjectID   iamActivityColumn
	Environment iamActivityColumn
	UserID      iamActivityColumn
	Type        iamActivityColumn
	At          iamActivityColumn
	Data        iamActivityColumn
}

// Alias returns the current table alias for the columns set.
func (c iamActivityColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamActivityColumns) AliasedAs(tableName string) iamActivityColumns {
	return buildIamActivityColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamActivityColumns) Unqualified() iamActivityColumns {
	return buildIamActivityColumns("")
}

func buildIamActivityColumn(alias, name string) iamActivityColumn {
	return iamActivityColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamActivityColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamActivityColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamActivityColumn) ShouldOmitParens() bool {
	return true
}

// IamActivitySetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamActivitySetter struct {
	ID          *string          `db:"id,pk" `
	ProjectID   *string          `db:"project_id" `
	Environment *string          `db:"environment" `
	UserID      *string          `db:"user_id" `
	Type        *string          `db:"type" `
	At          *time.Time       `db:"at" `
	Data        *json.RawMessage `db:"data" `
}

func (s IamActivitySetter) SetColumns() []string {
	vals := make([]string, 0, 7)
	if s.ID != nil {
		vals = append(vals, "id")
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
	if s.Type != nil {
		vals = append(vals, "type")
	}
	if s.At != nil {
		vals = append(vals, "at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamActivitySetter) Overwrite(t *IamActivity) {
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
	if s.UserID != nil {
		t.UserID = func() string {
			if s.UserID == nil {
				return *new(string)
			}
			return *s.UserID
		}()
	}
	if s.Type != nil {
		t.Type = func() string {
			if s.Type == nil {
				return *new(string)
			}
			return *s.Type
		}()
	}
	if s.At != nil {
		t.At = func() time.Time {
			if s.At == nil {
				return *new(time.Time)
			}
			return *s.At
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

func (s *IamActivitySetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamActivities.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 7)
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

		if s.Type != nil {
			vals[4] = psql.Arg(func() string {
				if s.Type == nil {
					return *new(string)
				}
				return *s.Type
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.At != nil {
			vals[5] = psql.Arg(func() time.Time {
				if s.At == nil {
					return *new(time.Time)
				}
				return *s.At
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[6] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamActivitySetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamActivitySetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 7)

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

	if s.UserID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_id")...),
			psql.Arg(s.UserID),
		}})
	}

	if s.Type != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "type")...),
			psql.Arg(s.Type),
		}})
	}

	if s.At != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "at")...),
			psql.Arg(s.At),
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

// FindIamActivity retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamActivity(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamActivity, error) {
	if len(cols) == 0 {
		return IamActivities.Query(
			sm.Where(IamActivities.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamActivities.Query(
		sm.Where(IamActivities.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamActivities.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamActivityExists checks the presence of a single record by primary key
func IamActivityExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamActivities.Query(
		sm.Where(IamActivities.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamActivity is retrieved from the database
func (o *IamActivity) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamActivities.AfterSelectHooks.RunHooks(ctx, exec, IamActivitySlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamActivities.AfterInsertHooks.RunHooks(ctx, exec, IamActivitySlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamActivities.AfterUpdateHooks.RunHooks(ctx, exec, IamActivitySlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamActivities.AfterDeleteHooks.RunHooks(ctx, exec, IamActivitySlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamActivities.AfterMergeHooks.RunHooks(ctx, exec, IamActivitySlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamActivity
func (o *IamActivity) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamActivity) pkEQ() dialect.Expression {
	return psql.Quote("iam_activity", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamActivity
func (o *IamActivity) Update(ctx context.Context, exec bob.Executor, s *IamActivitySetter) error {
	v, err := IamActivities.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamActivity record with an executor
func (o *IamActivity) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamActivities.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamActivity using the executor
func (o *IamActivity) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamActivities.Query(
		sm.Where(IamActivities.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamActivitySlice is retrieved from the database
func (o IamActivitySlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamActivities.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamActivities.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamActivities.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamActivities.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamActivities.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamActivitySlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_activity", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamActivitySlice) copyMatchingRows(from ...*IamActivity) {
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
func (o IamActivitySlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamActivities.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamActivity:
				o.copyMatchingRows(retrieved)
			case []*IamActivity:
				o.copyMatchingRows(retrieved...)
			case IamActivitySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamActivity or a slice of IamActivity
				// then run the AfterUpdateHooks on the slice
				_, err = IamActivities.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamActivitySlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamActivities.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamActivity:
				o.copyMatchingRows(retrieved)
			case []*IamActivity:
				o.copyMatchingRows(retrieved...)
			case IamActivitySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamActivity or a slice of IamActivity
				// then run the AfterDeleteHooks on the slice
				_, err = IamActivities.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamActivitySlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamActivities.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamActivity:
				o.copyMatchingRows(retrieved)
			case []*IamActivity:
				o.copyMatchingRows(retrieved...)
			case IamActivitySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamActivity or a slice of IamActivity
				// then run the AfterMergeHooks on the slice
				_, err = IamActivities.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamActivitySlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamActivitySetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamActivities.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamActivitySlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamActivities.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamActivitySlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamActivities.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamActivityWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	UserID      psql.WhereMod[Q, string]
	Type        psql.WhereMod[Q, string]
	At          psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamActivityWhere[Q]) AliasedAs(alias string) iamActivityWhere[Q] {
	return buildIamActivityWhere[Q](buildIamActivityColumns(alias))
}

func buildIamActivityWhere[Q psql.Filterable](cols iamActivityColumns) iamActivityWhere[Q] {
	return iamActivityWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		UserID:      psql.Where[Q, string](cols.UserID.Expression),
		Type:        psql.Where[Q, string](cols.Type.Expression),
		At:          psql.Where[Q, time.Time](cols.At.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
