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

// IamWebhook is an object representing the database table.
type IamWebhook struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	Enabled   bool            `db:"enabled" `
	CreatedAt time.Time       `db:"created_at" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamWebhookSlice is an alias for a slice of pointers to IamWebhook.
// This should almost always be used instead of []*IamWebhook.
type IamWebhookSlice []*IamWebhook

// IamWebhooks contains methods to work with the iam_webhooks table
var IamWebhooks = psql.NewTablex[*IamWebhook, IamWebhookSlice, *IamWebhookSetter]("", "iam_webhooks", buildIamWebhookColumns("iam_webhooks"))

// IamWebhooksQuery is a query on the iam_webhooks table
type IamWebhooksQuery = *psql.ViewQuery[*IamWebhook, IamWebhookSlice]

func buildIamWebhookColumns(tableName string) iamWebhookColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "enabled", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamWebhookColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamWebhookColumn(tableName, "id"),
		ProjectID:   buildIamWebhookColumn(tableName, "project_id"),
		Enabled:     buildIamWebhookColumn(tableName, "enabled"),
		CreatedAt:   buildIamWebhookColumn(tableName, "created_at"),
		UpdatedAt:   buildIamWebhookColumn(tableName, "updated_at"),
		Data:        buildIamWebhookColumn(tableName, "data"),
	}
}

type iamWebhookColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamWebhookColumn
	ProjectID  iamWebhookColumn
	Enabled    iamWebhookColumn
	CreatedAt  iamWebhookColumn
	UpdatedAt  iamWebhookColumn
	Data       iamWebhookColumn
}

// Alias returns the current table alias for the columns set.
func (c iamWebhookColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamWebhookColumns) AliasedAs(tableName string) iamWebhookColumns {
	return buildIamWebhookColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamWebhookColumns) Unqualified() iamWebhookColumns {
	return buildIamWebhookColumns("")
}

func buildIamWebhookColumn(alias, name string) iamWebhookColumn {
	return iamWebhookColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamWebhookColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamWebhookColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamWebhookColumn) ShouldOmitParens() bool {
	return true
}

// IamWebhookSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamWebhookSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	Enabled   *bool            `db:"enabled" `
	CreatedAt *time.Time       `db:"created_at" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamWebhookSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
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

func (s IamWebhookSetter) Overwrite(t *IamWebhook) {
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

func (s *IamWebhookSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamWebhooks.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 6)
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

		if s.Enabled != nil {
			vals[2] = psql.Arg(func() bool {
				if s.Enabled == nil {
					return *new(bool)
				}
				return *s.Enabled
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

		if s.UpdatedAt != nil {
			vals[4] = psql.Arg(func() time.Time {
				if s.UpdatedAt == nil {
					return *new(time.Time)
				}
				return *s.UpdatedAt
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[5] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamWebhookSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamWebhookSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 6)

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

// FindIamWebhook retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamWebhook(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamWebhook, error) {
	if len(cols) == 0 {
		return IamWebhooks.Query(
			sm.Where(IamWebhooks.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamWebhooks.Query(
		sm.Where(IamWebhooks.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamWebhooks.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamWebhookExists checks the presence of a single record by primary key
func IamWebhookExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamWebhooks.Query(
		sm.Where(IamWebhooks.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamWebhook is retrieved from the database
func (o *IamWebhook) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamWebhooks.AfterSelectHooks.RunHooks(ctx, exec, IamWebhookSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamWebhooks.AfterInsertHooks.RunHooks(ctx, exec, IamWebhookSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamWebhooks.AfterUpdateHooks.RunHooks(ctx, exec, IamWebhookSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamWebhooks.AfterDeleteHooks.RunHooks(ctx, exec, IamWebhookSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamWebhooks.AfterMergeHooks.RunHooks(ctx, exec, IamWebhookSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamWebhook
func (o *IamWebhook) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamWebhook) pkEQ() dialect.Expression {
	return psql.Quote("iam_webhooks", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamWebhook
func (o *IamWebhook) Update(ctx context.Context, exec bob.Executor, s *IamWebhookSetter) error {
	v, err := IamWebhooks.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamWebhook record with an executor
func (o *IamWebhook) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamWebhooks.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamWebhook using the executor
func (o *IamWebhook) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamWebhooks.Query(
		sm.Where(IamWebhooks.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamWebhookSlice is retrieved from the database
func (o IamWebhookSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamWebhooks.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamWebhooks.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamWebhooks.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamWebhooks.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamWebhooks.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamWebhookSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_webhooks", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamWebhookSlice) copyMatchingRows(from ...*IamWebhook) {
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
func (o IamWebhookSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebhooks.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebhook:
				o.copyMatchingRows(retrieved)
			case []*IamWebhook:
				o.copyMatchingRows(retrieved...)
			case IamWebhookSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebhook or a slice of IamWebhook
				// then run the AfterUpdateHooks on the slice
				_, err = IamWebhooks.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamWebhookSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebhooks.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebhook:
				o.copyMatchingRows(retrieved)
			case []*IamWebhook:
				o.copyMatchingRows(retrieved...)
			case IamWebhookSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebhook or a slice of IamWebhook
				// then run the AfterDeleteHooks on the slice
				_, err = IamWebhooks.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamWebhookSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebhooks.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebhook:
				o.copyMatchingRows(retrieved)
			case []*IamWebhook:
				o.copyMatchingRows(retrieved...)
			case IamWebhookSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebhook or a slice of IamWebhook
				// then run the AfterMergeHooks on the slice
				_, err = IamWebhooks.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamWebhookSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamWebhookSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamWebhooks.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamWebhookSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamWebhooks.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamWebhookSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamWebhooks.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamWebhookWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Enabled   psql.WhereMod[Q, bool]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamWebhookWhere[Q]) AliasedAs(alias string) iamWebhookWhere[Q] {
	return buildIamWebhookWhere[Q](buildIamWebhookColumns(alias))
}

func buildIamWebhookWhere[Q psql.Filterable](cols iamWebhookColumns) iamWebhookWhere[Q] {
	return iamWebhookWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Enabled:   psql.Where[Q, bool](cols.Enabled.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
