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

// IamSession is an object representing the database table.
type IamSession struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	UserID    string          `db:"user_id" `
	CreatedAt time.Time       `db:"created_at" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamSessionSlice is an alias for a slice of pointers to IamSession.
// This should almost always be used instead of []*IamSession.
type IamSessionSlice []*IamSession

// IamSessions contains methods to work with the iam_sessions table
var IamSessions = psql.NewTablex[*IamSession, IamSessionSlice, *IamSessionSetter]("", "iam_sessions", buildIamSessionColumns("iam_sessions"))

// IamSessionsQuery is a query on the iam_sessions table
type IamSessionsQuery = *psql.ViewQuery[*IamSession, IamSessionSlice]

func buildIamSessionColumns(tableName string) iamSessionColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "user_id", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSessionColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamSessionColumn(tableName, "id"),
		ProjectID:   buildIamSessionColumn(tableName, "project_id"),
		UserID:      buildIamSessionColumn(tableName, "user_id"),
		CreatedAt:   buildIamSessionColumn(tableName, "created_at"),
		UpdatedAt:   buildIamSessionColumn(tableName, "updated_at"),
		Data:        buildIamSessionColumn(tableName, "data"),
	}
}

type iamSessionColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamSessionColumn
	ProjectID  iamSessionColumn
	UserID     iamSessionColumn
	CreatedAt  iamSessionColumn
	UpdatedAt  iamSessionColumn
	Data       iamSessionColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSessionColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSessionColumns) AliasedAs(tableName string) iamSessionColumns {
	return buildIamSessionColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSessionColumns) Unqualified() iamSessionColumns {
	return buildIamSessionColumns("")
}

func buildIamSessionColumn(alias, name string) iamSessionColumn {
	return iamSessionColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSessionColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSessionColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSessionColumn) ShouldOmitParens() bool {
	return true
}

// IamSessionSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSessionSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	UserID    *string          `db:"user_id" `
	CreatedAt *time.Time       `db:"created_at" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamSessionSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
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

func (s IamSessionSetter) Overwrite(t *IamSession) {
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

func (s *IamSessionSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSessions.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.UserID != nil {
			vals[2] = psql.Arg(func() string {
				if s.UserID == nil {
					return *new(string)
				}
				return *s.UserID
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

func (s IamSessionSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSessionSetter) Expressions(prefix ...string) []bob.Expression {
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

// FindIamSession retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSession(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamSession, error) {
	if len(cols) == 0 {
		return IamSessions.Query(
			sm.Where(IamSessions.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamSessions.Query(
		sm.Where(IamSessions.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamSessions.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSessionExists checks the presence of a single record by primary key
func IamSessionExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamSessions.Query(
		sm.Where(IamSessions.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSession is retrieved from the database
func (o *IamSession) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSessions.AfterSelectHooks.RunHooks(ctx, exec, IamSessionSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSessions.AfterInsertHooks.RunHooks(ctx, exec, IamSessionSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSessions.AfterUpdateHooks.RunHooks(ctx, exec, IamSessionSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSessions.AfterDeleteHooks.RunHooks(ctx, exec, IamSessionSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSessions.AfterMergeHooks.RunHooks(ctx, exec, IamSessionSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSession
func (o *IamSession) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamSession) pkEQ() dialect.Expression {
	return psql.Quote("iam_sessions", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSession
func (o *IamSession) Update(ctx context.Context, exec bob.Executor, s *IamSessionSetter) error {
	v, err := IamSessions.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSession record with an executor
func (o *IamSession) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSessions.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSession using the executor
func (o *IamSession) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSessions.Query(
		sm.Where(IamSessions.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSessionSlice is retrieved from the database
func (o IamSessionSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSessions.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSessions.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSessions.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSessions.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSessions.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSessionSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_sessions", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSessionSlice) copyMatchingRows(from ...*IamSession) {
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
func (o IamSessionSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSessions.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSession:
				o.copyMatchingRows(retrieved)
			case []*IamSession:
				o.copyMatchingRows(retrieved...)
			case IamSessionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSession or a slice of IamSession
				// then run the AfterUpdateHooks on the slice
				_, err = IamSessions.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSessionSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSessions.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSession:
				o.copyMatchingRows(retrieved)
			case []*IamSession:
				o.copyMatchingRows(retrieved...)
			case IamSessionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSession or a slice of IamSession
				// then run the AfterDeleteHooks on the slice
				_, err = IamSessions.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSessionSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSessions.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSession:
				o.copyMatchingRows(retrieved)
			case []*IamSession:
				o.copyMatchingRows(retrieved...)
			case IamSessionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSession or a slice of IamSession
				// then run the AfterMergeHooks on the slice
				_, err = IamSessions.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSessionSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSessionSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSessions.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSessionSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSessions.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSessionSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSessions.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSessionWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	UserID    psql.WhereMod[Q, string]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamSessionWhere[Q]) AliasedAs(alias string) iamSessionWhere[Q] {
	return buildIamSessionWhere[Q](buildIamSessionColumns(alias))
}

func buildIamSessionWhere[Q psql.Filterable](cols iamSessionColumns) iamSessionWhere[Q] {
	return iamSessionWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		UserID:    psql.Where[Q, string](cols.UserID.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
