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

// IamHook is an object representing the database table.
type IamHook struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	Type      string          `db:"type" `
	Enabled   bool            `db:"enabled" `
	CreatedAt time.Time       `db:"created_at" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamHookSlice is an alias for a slice of pointers to IamHook.
// This should almost always be used instead of []*IamHook.
type IamHookSlice []*IamHook

// IamHooks contains methods to work with the iam_hooks table
var IamHooks = psql.NewTablex[*IamHook, IamHookSlice, *IamHookSetter]("", "iam_hooks", buildIamHookColumns("iam_hooks"))

// IamHooksQuery is a query on the iam_hooks table
type IamHooksQuery = *psql.ViewQuery[*IamHook, IamHookSlice]

func buildIamHookColumns(tableName string) iamHookColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "type", "enabled", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamHookColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamHookColumn(tableName, "id"),
		ProjectID:   buildIamHookColumn(tableName, "project_id"),
		Type:        buildIamHookColumn(tableName, "type"),
		Enabled:     buildIamHookColumn(tableName, "enabled"),
		CreatedAt:   buildIamHookColumn(tableName, "created_at"),
		UpdatedAt:   buildIamHookColumn(tableName, "updated_at"),
		Data:        buildIamHookColumn(tableName, "data"),
	}
}

type iamHookColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamHookColumn
	ProjectID  iamHookColumn
	Type       iamHookColumn
	Enabled    iamHookColumn
	CreatedAt  iamHookColumn
	UpdatedAt  iamHookColumn
	Data       iamHookColumn
}

// Alias returns the current table alias for the columns set.
func (c iamHookColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamHookColumns) AliasedAs(tableName string) iamHookColumns {
	return buildIamHookColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamHookColumns) Unqualified() iamHookColumns {
	return buildIamHookColumns("")
}

func buildIamHookColumn(alias, name string) iamHookColumn {
	return iamHookColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamHookColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamHookColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamHookColumn) ShouldOmitParens() bool {
	return true
}

// IamHookSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamHookSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	Type      *string          `db:"type" `
	Enabled   *bool            `db:"enabled" `
	CreatedAt *time.Time       `db:"created_at" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamHookSetter) SetColumns() []string {
	vals := make([]string, 0, 7)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Type != nil {
		vals = append(vals, "type")
	}
	if s.Enabled != nil {
		vals = append(vals, "enabled")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.UpdatedAt != nil {
		vals = append(vals, "updated_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamHookSetter) Overwrite(t *IamHook) {
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
	if s.Type != nil {
		t.Type = func() string {
			if s.Type == nil {
				return *new(string)
			}
			return *s.Type
		}()
	}
	if s.Enabled != nil {
		t.Enabled = func() bool {
			if s.Enabled == nil {
				return *new(bool)
			}
			return *s.Enabled
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
	if s.UpdatedAt != nil {
		t.UpdatedAt = func() time.Time {
			if s.UpdatedAt == nil {
				return *new(time.Time)
			}
			return *s.UpdatedAt
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

func (s *IamHookSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamHooks.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Type != nil {
			vals[2] = psql.Arg(func() string {
				if s.Type == nil {
					return *new(string)
				}
				return *s.Type
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Enabled != nil {
			vals[3] = psql.Arg(func() bool {
				if s.Enabled == nil {
					return *new(bool)
				}
				return *s.Enabled
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

		if s.UpdatedAt != nil {
			vals[5] = psql.Arg(func() time.Time {
				if s.UpdatedAt == nil {
					return *new(time.Time)
				}
				return *s.UpdatedAt
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

func (s IamHookSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamHookSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Type != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "type")...),
			psql.Arg(s.Type),
		}})
	}

	if s.Enabled != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "enabled")...),
			psql.Arg(s.Enabled),
		}})
	}

	if s.CreatedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "created_at")...),
			psql.Arg(s.CreatedAt),
		}})
	}

	if s.UpdatedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "updated_at")...),
			psql.Arg(s.UpdatedAt),
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

// FindIamHook retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamHook(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamHook, error) {
	if len(cols) == 0 {
		return IamHooks.Query(
			sm.Where(IamHooks.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamHooks.Query(
		sm.Where(IamHooks.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamHooks.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamHookExists checks the presence of a single record by primary key
func IamHookExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamHooks.Query(
		sm.Where(IamHooks.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamHook is retrieved from the database
func (o *IamHook) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamHooks.AfterSelectHooks.RunHooks(ctx, exec, IamHookSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamHooks.AfterInsertHooks.RunHooks(ctx, exec, IamHookSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamHooks.AfterUpdateHooks.RunHooks(ctx, exec, IamHookSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamHooks.AfterDeleteHooks.RunHooks(ctx, exec, IamHookSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamHooks.AfterMergeHooks.RunHooks(ctx, exec, IamHookSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamHook
func (o *IamHook) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamHook) pkEQ() dialect.Expression {
	return psql.Quote("iam_hooks", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamHook
func (o *IamHook) Update(ctx context.Context, exec bob.Executor, s *IamHookSetter) error {
	v, err := IamHooks.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamHook record with an executor
func (o *IamHook) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamHooks.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamHook using the executor
func (o *IamHook) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamHooks.Query(
		sm.Where(IamHooks.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamHookSlice is retrieved from the database
func (o IamHookSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamHooks.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamHooks.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamHooks.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamHooks.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamHooks.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamHookSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_hooks", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamHookSlice) copyMatchingRows(from ...*IamHook) {
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
func (o IamHookSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamHooks.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamHook:
				o.copyMatchingRows(retrieved)
			case []*IamHook:
				o.copyMatchingRows(retrieved...)
			case IamHookSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamHook or a slice of IamHook
				// then run the AfterUpdateHooks on the slice
				_, err = IamHooks.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamHookSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamHooks.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamHook:
				o.copyMatchingRows(retrieved)
			case []*IamHook:
				o.copyMatchingRows(retrieved...)
			case IamHookSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamHook or a slice of IamHook
				// then run the AfterDeleteHooks on the slice
				_, err = IamHooks.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamHookSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamHooks.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamHook:
				o.copyMatchingRows(retrieved)
			case []*IamHook:
				o.copyMatchingRows(retrieved...)
			case IamHookSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamHook or a slice of IamHook
				// then run the AfterMergeHooks on the slice
				_, err = IamHooks.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamHookSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamHookSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamHooks.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamHookSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamHooks.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamHookSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamHooks.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamHookWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Type      psql.WhereMod[Q, string]
	Enabled   psql.WhereMod[Q, bool]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamHookWhere[Q]) AliasedAs(alias string) iamHookWhere[Q] {
	return buildIamHookWhere[Q](buildIamHookColumns(alias))
}

func buildIamHookWhere[Q psql.Filterable](cols iamHookColumns) iamHookWhere[Q] {
	return iamHookWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Type:      psql.Where[Q, string](cols.Type.Expression),
		Enabled:   psql.Where[Q, bool](cols.Enabled.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
