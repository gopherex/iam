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

// IamAdminToken is an object representing the database table.
type IamAdminToken struct {
	ID        string              `db:"id,pk" `
	ProjectID string              `db:"project_id" `
	Hash      string              `db:"hash" `
	ExpiresAt null.Val[time.Time] `db:"expires_at" `
	CreatedAt time.Time           `db:"created_at" `
	Data      json.RawMessage     `db:"data" `
}

// IamAdminTokenSlice is an alias for a slice of pointers to IamAdminToken.
// This should almost always be used instead of []*IamAdminToken.
type IamAdminTokenSlice []*IamAdminToken

// IamAdminTokens contains methods to work with the iam_admin_tokens table
var IamAdminTokens = psql.NewTablex[*IamAdminToken, IamAdminTokenSlice, *IamAdminTokenSetter]("", "iam_admin_tokens", buildIamAdminTokenColumns("iam_admin_tokens"))

// IamAdminTokensQuery is a query on the iam_admin_tokens table
type IamAdminTokensQuery = *psql.ViewQuery[*IamAdminToken, IamAdminTokenSlice]

func buildIamAdminTokenColumns(tableName string) iamAdminTokenColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "hash", "expires_at", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamAdminTokenColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamAdminTokenColumn(tableName, "id"),
		ProjectID:   buildIamAdminTokenColumn(tableName, "project_id"),
		Hash:        buildIamAdminTokenColumn(tableName, "hash"),
		ExpiresAt:   buildIamAdminTokenColumn(tableName, "expires_at"),
		CreatedAt:   buildIamAdminTokenColumn(tableName, "created_at"),
		Data:        buildIamAdminTokenColumn(tableName, "data"),
	}
}

type iamAdminTokenColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamAdminTokenColumn
	ProjectID  iamAdminTokenColumn
	Hash       iamAdminTokenColumn
	ExpiresAt  iamAdminTokenColumn
	CreatedAt  iamAdminTokenColumn
	Data       iamAdminTokenColumn
}

// Alias returns the current table alias for the columns set.
func (c iamAdminTokenColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamAdminTokenColumns) AliasedAs(tableName string) iamAdminTokenColumns {
	return buildIamAdminTokenColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamAdminTokenColumns) Unqualified() iamAdminTokenColumns {
	return buildIamAdminTokenColumns("")
}

func buildIamAdminTokenColumn(alias, name string) iamAdminTokenColumn {
	return iamAdminTokenColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamAdminTokenColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamAdminTokenColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamAdminTokenColumn) ShouldOmitParens() bool {
	return true
}

// IamAdminTokenSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamAdminTokenSetter struct {
	ID        *string              `db:"id,pk" `
	ProjectID *string              `db:"project_id" `
	Hash      *string              `db:"hash" `
	ExpiresAt *null.Val[time.Time] `db:"expires_at" `
	CreatedAt *time.Time           `db:"created_at" `
	Data      *json.RawMessage     `db:"data" `
}

func (s IamAdminTokenSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Hash != nil {
		vals = append(vals, "hash")
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

func (s IamAdminTokenSetter) Overwrite(t *IamAdminToken) {
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
	if s.Hash != nil {
		t.Hash = func() string {
			if s.Hash == nil {
				return *new(string)
			}
			return *s.Hash
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

func (s *IamAdminTokenSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamAdminTokens.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Hash != nil {
			vals[2] = psql.Arg(func() string {
				if s.Hash == nil {
					return *new(string)
				}
				return *s.Hash
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[3] = psql.Arg(func() null.Val[time.Time] {
				if s.ExpiresAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.ExpiresAt
				return *v
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

func (s IamAdminTokenSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamAdminTokenSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Hash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "hash")...),
			psql.Arg(s.Hash),
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

// FindIamAdminToken retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamAdminToken(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamAdminToken, error) {
	if len(cols) == 0 {
		return IamAdminTokens.Query(
			sm.Where(IamAdminTokens.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamAdminTokens.Query(
		sm.Where(IamAdminTokens.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamAdminTokens.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamAdminTokenExists checks the presence of a single record by primary key
func IamAdminTokenExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamAdminTokens.Query(
		sm.Where(IamAdminTokens.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamAdminToken is retrieved from the database
func (o *IamAdminToken) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAdminTokens.AfterSelectHooks.RunHooks(ctx, exec, IamAdminTokenSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamAdminTokens.AfterInsertHooks.RunHooks(ctx, exec, IamAdminTokenSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamAdminTokens.AfterUpdateHooks.RunHooks(ctx, exec, IamAdminTokenSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamAdminTokens.AfterDeleteHooks.RunHooks(ctx, exec, IamAdminTokenSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamAdminTokens.AfterMergeHooks.RunHooks(ctx, exec, IamAdminTokenSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamAdminToken
func (o *IamAdminToken) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamAdminToken) pkEQ() dialect.Expression {
	return psql.Quote("iam_admin_tokens", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamAdminToken
func (o *IamAdminToken) Update(ctx context.Context, exec bob.Executor, s *IamAdminTokenSetter) error {
	v, err := IamAdminTokens.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamAdminToken record with an executor
func (o *IamAdminToken) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamAdminTokens.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamAdminToken using the executor
func (o *IamAdminToken) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamAdminTokens.Query(
		sm.Where(IamAdminTokens.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamAdminTokenSlice is retrieved from the database
func (o IamAdminTokenSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAdminTokens.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamAdminTokens.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamAdminTokens.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamAdminTokens.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamAdminTokens.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamAdminTokenSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_admin_tokens", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamAdminTokenSlice) copyMatchingRows(from ...*IamAdminToken) {
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
func (o IamAdminTokenSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAdminTokens.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAdminToken:
				o.copyMatchingRows(retrieved)
			case []*IamAdminToken:
				o.copyMatchingRows(retrieved...)
			case IamAdminTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAdminToken or a slice of IamAdminToken
				// then run the AfterUpdateHooks on the slice
				_, err = IamAdminTokens.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamAdminTokenSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAdminTokens.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAdminToken:
				o.copyMatchingRows(retrieved)
			case []*IamAdminToken:
				o.copyMatchingRows(retrieved...)
			case IamAdminTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAdminToken or a slice of IamAdminToken
				// then run the AfterDeleteHooks on the slice
				_, err = IamAdminTokens.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamAdminTokenSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAdminTokens.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAdminToken:
				o.copyMatchingRows(retrieved)
			case []*IamAdminToken:
				o.copyMatchingRows(retrieved...)
			case IamAdminTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAdminToken or a slice of IamAdminToken
				// then run the AfterMergeHooks on the slice
				_, err = IamAdminTokens.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamAdminTokenSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamAdminTokenSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAdminTokens.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamAdminTokenSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAdminTokens.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamAdminTokenSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamAdminTokens.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamAdminTokenWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Hash      psql.WhereMod[Q, string]
	ExpiresAt psql.WhereNullMod[Q, time.Time]
	CreatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamAdminTokenWhere[Q]) AliasedAs(alias string) iamAdminTokenWhere[Q] {
	return buildIamAdminTokenWhere[Q](buildIamAdminTokenColumns(alias))
}

func buildIamAdminTokenWhere[Q psql.Filterable](cols iamAdminTokenColumns) iamAdminTokenWhere[Q] {
	return iamAdminTokenWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Hash:      psql.Where[Q, string](cols.Hash.Expression),
		ExpiresAt: psql.WhereNull[Q, time.Time](cols.ExpiresAt.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
