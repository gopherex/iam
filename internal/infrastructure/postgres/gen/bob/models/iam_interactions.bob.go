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

// IamInteraction is an object representing the database table.
type IamInteraction struct {
	ID        string              `db:"id,pk" `
	ProjectID string              `db:"project_id" `
	ClientID  null.Val[string]    `db:"client_id" `
	SessionID null.Val[string]    `db:"session_id" `
	ExpiresAt null.Val[time.Time] `db:"expires_at" `
	CreatedAt time.Time           `db:"created_at" `
	Data      json.RawMessage     `db:"data" `
}

// IamInteractionSlice is an alias for a slice of pointers to IamInteraction.
// This should almost always be used instead of []*IamInteraction.
type IamInteractionSlice []*IamInteraction

// IamInteractions contains methods to work with the iam_interactions table
var IamInteractions = psql.NewTablex[*IamInteraction, IamInteractionSlice, *IamInteractionSetter]("", "iam_interactions", buildIamInteractionColumns("iam_interactions"))

// IamInteractionsQuery is a query on the iam_interactions table
type IamInteractionsQuery = *psql.ViewQuery[*IamInteraction, IamInteractionSlice]

func buildIamInteractionColumns(tableName string) iamInteractionColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "client_id", "session_id", "expires_at", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamInteractionColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamInteractionColumn(tableName, "id"),
		ProjectID:   buildIamInteractionColumn(tableName, "project_id"),
		ClientID:    buildIamInteractionColumn(tableName, "client_id"),
		SessionID:   buildIamInteractionColumn(tableName, "session_id"),
		ExpiresAt:   buildIamInteractionColumn(tableName, "expires_at"),
		CreatedAt:   buildIamInteractionColumn(tableName, "created_at"),
		Data:        buildIamInteractionColumn(tableName, "data"),
	}
}

type iamInteractionColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamInteractionColumn
	ProjectID  iamInteractionColumn
	ClientID   iamInteractionColumn
	SessionID  iamInteractionColumn
	ExpiresAt  iamInteractionColumn
	CreatedAt  iamInteractionColumn
	Data       iamInteractionColumn
}

// Alias returns the current table alias for the columns set.
func (c iamInteractionColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamInteractionColumns) AliasedAs(tableName string) iamInteractionColumns {
	return buildIamInteractionColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamInteractionColumns) Unqualified() iamInteractionColumns {
	return buildIamInteractionColumns("")
}

func buildIamInteractionColumn(alias, name string) iamInteractionColumn {
	return iamInteractionColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamInteractionColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamInteractionColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamInteractionColumn) ShouldOmitParens() bool {
	return true
}

// IamInteractionSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamInteractionSetter struct {
	ID        *string              `db:"id,pk" `
	ProjectID *string              `db:"project_id" `
	ClientID  *null.Val[string]    `db:"client_id" `
	SessionID *null.Val[string]    `db:"session_id" `
	ExpiresAt *null.Val[time.Time] `db:"expires_at" `
	CreatedAt *time.Time           `db:"created_at" `
	Data      *json.RawMessage     `db:"data" `
}

func (s IamInteractionSetter) SetColumns() []string {
	vals := make([]string, 0, 7)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.ClientID != nil {
		vals = append(vals, "client_id")
	}
	if s.SessionID != nil {
		vals = append(vals, "session_id")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamInteractionSetter) Overwrite(t *IamInteraction) {
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
	if s.ClientID != nil {
		t.ClientID = func() null.Val[string] {
			if s.ClientID == nil {
				return *new(null.Val[string])
			}
			v := s.ClientID
			return *v
		}()
	}
	if s.SessionID != nil {
		t.SessionID = func() null.Val[string] {
			if s.SessionID == nil {
				return *new(null.Val[string])
			}
			v := s.SessionID
			return *v
		}()
	}
	if s.ExpiresAt != nil {
		t.ExpiresAt = func() null.Val[time.Time] {
			if s.ExpiresAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.ExpiresAt
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

func (s *IamInteractionSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamInteractions.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.ClientID != nil {
			vals[2] = psql.Arg(func() null.Val[string] {
				if s.ClientID == nil {
					return *new(null.Val[string])
				}
				v := s.ClientID
				return *v
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.SessionID != nil {
			vals[3] = psql.Arg(func() null.Val[string] {
				if s.SessionID == nil {
					return *new(null.Val[string])
				}
				v := s.SessionID
				return *v
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[4] = psql.Arg(func() null.Val[time.Time] {
				if s.ExpiresAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.ExpiresAt
				return *v
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

func (s IamInteractionSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamInteractionSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.ClientID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "client_id")...),
			psql.Arg(s.ClientID),
		}})
	}

	if s.SessionID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "session_id")...),
			psql.Arg(s.SessionID),
		}})
	}

	if s.ExpiresAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "expires_at")...),
			psql.Arg(s.ExpiresAt),
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

// FindIamInteraction retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamInteraction(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamInteraction, error) {
	if len(cols) == 0 {
		return IamInteractions.Query(
			sm.Where(IamInteractions.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamInteractions.Query(
		sm.Where(IamInteractions.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamInteractions.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamInteractionExists checks the presence of a single record by primary key
func IamInteractionExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamInteractions.Query(
		sm.Where(IamInteractions.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamInteraction is retrieved from the database
func (o *IamInteraction) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamInteractions.AfterSelectHooks.RunHooks(ctx, exec, IamInteractionSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamInteractions.AfterInsertHooks.RunHooks(ctx, exec, IamInteractionSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamInteractions.AfterUpdateHooks.RunHooks(ctx, exec, IamInteractionSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamInteractions.AfterDeleteHooks.RunHooks(ctx, exec, IamInteractionSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamInteractions.AfterMergeHooks.RunHooks(ctx, exec, IamInteractionSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamInteraction
func (o *IamInteraction) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamInteraction) pkEQ() dialect.Expression {
	return psql.Quote("iam_interactions", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamInteraction
func (o *IamInteraction) Update(ctx context.Context, exec bob.Executor, s *IamInteractionSetter) error {
	v, err := IamInteractions.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamInteraction record with an executor
func (o *IamInteraction) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamInteractions.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamInteraction using the executor
func (o *IamInteraction) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamInteractions.Query(
		sm.Where(IamInteractions.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamInteractionSlice is retrieved from the database
func (o IamInteractionSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamInteractions.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamInteractions.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamInteractions.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamInteractions.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamInteractions.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamInteractionSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_interactions", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamInteractionSlice) copyMatchingRows(from ...*IamInteraction) {
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
func (o IamInteractionSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamInteractions.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamInteraction:
				o.copyMatchingRows(retrieved)
			case []*IamInteraction:
				o.copyMatchingRows(retrieved...)
			case IamInteractionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamInteraction or a slice of IamInteraction
				// then run the AfterUpdateHooks on the slice
				_, err = IamInteractions.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamInteractionSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamInteractions.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamInteraction:
				o.copyMatchingRows(retrieved)
			case []*IamInteraction:
				o.copyMatchingRows(retrieved...)
			case IamInteractionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamInteraction or a slice of IamInteraction
				// then run the AfterDeleteHooks on the slice
				_, err = IamInteractions.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamInteractionSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamInteractions.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamInteraction:
				o.copyMatchingRows(retrieved)
			case []*IamInteraction:
				o.copyMatchingRows(retrieved...)
			case IamInteractionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamInteraction or a slice of IamInteraction
				// then run the AfterMergeHooks on the slice
				_, err = IamInteractions.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamInteractionSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamInteractionSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamInteractions.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamInteractionSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamInteractions.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamInteractionSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamInteractions.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamInteractionWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	ClientID  psql.WhereNullMod[Q, string]
	SessionID psql.WhereNullMod[Q, string]
	ExpiresAt psql.WhereNullMod[Q, time.Time]
	CreatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamInteractionWhere[Q]) AliasedAs(alias string) iamInteractionWhere[Q] {
	return buildIamInteractionWhere[Q](buildIamInteractionColumns(alias))
}

func buildIamInteractionWhere[Q psql.Filterable](cols iamInteractionColumns) iamInteractionWhere[Q] {
	return iamInteractionWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		ClientID:  psql.WhereNull[Q, string](cols.ClientID.Expression),
		SessionID: psql.WhereNull[Q, string](cols.SessionID.Expression),
		ExpiresAt: psql.WhereNull[Q, time.Time](cols.ExpiresAt.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
