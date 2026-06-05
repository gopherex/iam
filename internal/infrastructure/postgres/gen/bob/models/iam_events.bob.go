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

// IamEvent is an object representing the database table.
type IamEvent struct {
	ID          string          `db:"id,pk" `
	ProjectID   string          `db:"project_id" `
	Environment string          `db:"environment" `
	Type        string          `db:"type" `
	Published   bool            `db:"published" `
	CreatedAt   time.Time       `db:"created_at" `
	Data        json.RawMessage `db:"data" `
}

// IamEventSlice is an alias for a slice of pointers to IamEvent.
// This should almost always be used instead of []*IamEvent.
type IamEventSlice []*IamEvent

// IamEvents contains methods to work with the iam_events table
var IamEvents = psql.NewTablex[*IamEvent, IamEventSlice, *IamEventSetter]("", "iam_events", buildIamEventColumns("iam_events"))

// IamEventsQuery is a query on the iam_events table
type IamEventsQuery = *psql.ViewQuery[*IamEvent, IamEventSlice]

func buildIamEventColumns(tableName string) iamEventColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "type", "published", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamEventColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamEventColumn(tableName, "id"),
		ProjectID:   buildIamEventColumn(tableName, "project_id"),
		Environment: buildIamEventColumn(tableName, "environment"),
		Type:        buildIamEventColumn(tableName, "type"),
		Published:   buildIamEventColumn(tableName, "published"),
		CreatedAt:   buildIamEventColumn(tableName, "created_at"),
		Data:        buildIamEventColumn(tableName, "data"),
	}
}

type iamEventColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamEventColumn
	ProjectID   iamEventColumn
	Environment iamEventColumn
	Type        iamEventColumn
	Published   iamEventColumn
	CreatedAt   iamEventColumn
	Data        iamEventColumn
}

// Alias returns the current table alias for the columns set.
func (c iamEventColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamEventColumns) AliasedAs(tableName string) iamEventColumns {
	return buildIamEventColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamEventColumns) Unqualified() iamEventColumns {
	return buildIamEventColumns("")
}

func buildIamEventColumn(alias, name string) iamEventColumn {
	return iamEventColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamEventColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamEventColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamEventColumn) ShouldOmitParens() bool {
	return true
}

// IamEventSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamEventSetter struct {
	ID          *string          `db:"id,pk" `
	ProjectID   *string          `db:"project_id" `
	Environment *string          `db:"environment" `
	Type        *string          `db:"type" `
	Published   *bool            `db:"published" `
	CreatedAt   *time.Time       `db:"created_at" `
	Data        *json.RawMessage `db:"data" `
}

func (s IamEventSetter) SetColumns() []string {
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
	if s.Type != nil {
		vals = append(vals, "type")
	}
	if s.Published != nil {
		vals = append(vals, "published")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamEventSetter) Overwrite(t *IamEvent) {
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
	if s.Type != nil {
		t.Type = func() string {
			if s.Type == nil {
				return *new(string)
			}
			return *s.Type
		}()
	}
	if s.Published != nil {
		t.Published = func() bool {
			if s.Published == nil {
				return *new(bool)
			}
			return *s.Published
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

func (s *IamEventSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamEvents.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Type != nil {
			vals[3] = psql.Arg(func() string {
				if s.Type == nil {
					return *new(string)
				}
				return *s.Type
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.Published != nil {
			vals[4] = psql.Arg(func() bool {
				if s.Published == nil {
					return *new(bool)
				}
				return *s.Published
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[5] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
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

func (s IamEventSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamEventSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Type != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "type")...),
			psql.Arg(s.Type),
		}})
	}

	if s.Published != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "published")...),
			psql.Arg(s.Published),
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

// FindIamEvent retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamEvent(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamEvent, error) {
	if len(cols) == 0 {
		return IamEvents.Query(
			sm.Where(IamEvents.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamEvents.Query(
		sm.Where(IamEvents.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamEvents.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamEventExists checks the presence of a single record by primary key
func IamEventExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamEvents.Query(
		sm.Where(IamEvents.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamEvent is retrieved from the database
func (o *IamEvent) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamEvents.AfterSelectHooks.RunHooks(ctx, exec, IamEventSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamEvents.AfterInsertHooks.RunHooks(ctx, exec, IamEventSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamEvents.AfterUpdateHooks.RunHooks(ctx, exec, IamEventSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamEvents.AfterDeleteHooks.RunHooks(ctx, exec, IamEventSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamEvents.AfterMergeHooks.RunHooks(ctx, exec, IamEventSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamEvent
func (o *IamEvent) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamEvent) pkEQ() dialect.Expression {
	return psql.Quote("iam_events", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamEvent
func (o *IamEvent) Update(ctx context.Context, exec bob.Executor, s *IamEventSetter) error {
	v, err := IamEvents.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamEvent record with an executor
func (o *IamEvent) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamEvents.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamEvent using the executor
func (o *IamEvent) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamEvents.Query(
		sm.Where(IamEvents.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamEventSlice is retrieved from the database
func (o IamEventSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamEvents.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamEvents.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamEvents.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamEvents.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamEvents.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamEventSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_events", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamEventSlice) copyMatchingRows(from ...*IamEvent) {
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
func (o IamEventSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEvents.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEvent:
				o.copyMatchingRows(retrieved)
			case []*IamEvent:
				o.copyMatchingRows(retrieved...)
			case IamEventSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEvent or a slice of IamEvent
				// then run the AfterUpdateHooks on the slice
				_, err = IamEvents.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamEventSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEvents.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEvent:
				o.copyMatchingRows(retrieved)
			case []*IamEvent:
				o.copyMatchingRows(retrieved...)
			case IamEventSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEvent or a slice of IamEvent
				// then run the AfterDeleteHooks on the slice
				_, err = IamEvents.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamEventSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEvents.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEvent:
				o.copyMatchingRows(retrieved)
			case []*IamEvent:
				o.copyMatchingRows(retrieved...)
			case IamEventSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEvent or a slice of IamEvent
				// then run the AfterMergeHooks on the slice
				_, err = IamEvents.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamEventSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamEventSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamEvents.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamEventSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamEvents.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamEventSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamEvents.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamEventWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	Type        psql.WhereMod[Q, string]
	Published   psql.WhereMod[Q, bool]
	CreatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamEventWhere[Q]) AliasedAs(alias string) iamEventWhere[Q] {
	return buildIamEventWhere[Q](buildIamEventColumns(alias))
}

func buildIamEventWhere[Q psql.Filterable](cols iamEventColumns) iamEventWhere[Q] {
	return iamEventWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		Type:        psql.Where[Q, string](cols.Type.Expression),
		Published:   psql.Where[Q, bool](cols.Published.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
