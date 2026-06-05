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

// IamScimToken is an object representing the database table.
type IamScimToken struct {
	ID           string          `db:"id,pk" `
	ProjectID    string          `db:"project_id" `
	ConnectionID string          `db:"connection_id" `
	Hash         string          `db:"hash" `
	CreatedAt    time.Time       `db:"created_at" `
	Data         json.RawMessage `db:"data" `
}

// IamScimTokenSlice is an alias for a slice of pointers to IamScimToken.
// This should almost always be used instead of []*IamScimToken.
type IamScimTokenSlice []*IamScimToken

// IamScimTokens contains methods to work with the iam_scim_tokens table
var IamScimTokens = psql.NewTablex[*IamScimToken, IamScimTokenSlice, *IamScimTokenSetter]("", "iam_scim_tokens", buildIamScimTokenColumns("iam_scim_tokens"))

// IamScimTokensQuery is a query on the iam_scim_tokens table
type IamScimTokensQuery = *psql.ViewQuery[*IamScimToken, IamScimTokenSlice]

func buildIamScimTokenColumns(tableName string) iamScimTokenColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "connection_id", "hash", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamScimTokenColumns{
		ColumnsExpr:  columnsExpr,
		tableAlias:   tableName,
		ID:           buildIamScimTokenColumn(tableName, "id"),
		ProjectID:    buildIamScimTokenColumn(tableName, "project_id"),
		ConnectionID: buildIamScimTokenColumn(tableName, "connection_id"),
		Hash:         buildIamScimTokenColumn(tableName, "hash"),
		CreatedAt:    buildIamScimTokenColumn(tableName, "created_at"),
		Data:         buildIamScimTokenColumn(tableName, "data"),
	}
}

type iamScimTokenColumns struct {
	expr.ColumnsExpr
	tableAlias   string
	ID           iamScimTokenColumn
	ProjectID    iamScimTokenColumn
	ConnectionID iamScimTokenColumn
	Hash         iamScimTokenColumn
	CreatedAt    iamScimTokenColumn
	Data         iamScimTokenColumn
}

// Alias returns the current table alias for the columns set.
func (c iamScimTokenColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamScimTokenColumns) AliasedAs(tableName string) iamScimTokenColumns {
	return buildIamScimTokenColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamScimTokenColumns) Unqualified() iamScimTokenColumns {
	return buildIamScimTokenColumns("")
}

func buildIamScimTokenColumn(alias, name string) iamScimTokenColumn {
	return iamScimTokenColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamScimTokenColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamScimTokenColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamScimTokenColumn) ShouldOmitParens() bool {
	return true
}

// IamScimTokenSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamScimTokenSetter struct {
	ID           *string          `db:"id,pk" `
	ProjectID    *string          `db:"project_id" `
	ConnectionID *string          `db:"connection_id" `
	Hash         *string          `db:"hash" `
	CreatedAt    *time.Time       `db:"created_at" `
	Data         *json.RawMessage `db:"data" `
}

func (s IamScimTokenSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.ConnectionID != nil {
		vals = append(vals, "connection_id")
	}
	if s.Hash != nil {
		vals = append(vals, "hash")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamScimTokenSetter) Overwrite(t *IamScimToken) {
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
	if s.ConnectionID != nil {
		t.ConnectionID = func() string {
			if s.ConnectionID == nil {
				return *new(string)
			}
			return *s.ConnectionID
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

func (s *IamScimTokenSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamScimTokens.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.ConnectionID != nil {
			vals[2] = psql.Arg(func() string {
				if s.ConnectionID == nil {
					return *new(string)
				}
				return *s.ConnectionID
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Hash != nil {
			vals[3] = psql.Arg(func() string {
				if s.Hash == nil {
					return *new(string)
				}
				return *s.Hash
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

func (s IamScimTokenSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamScimTokenSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.ConnectionID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "connection_id")...),
			psql.Arg(s.ConnectionID),
		}})
	}

	if s.Hash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "hash")...),
			psql.Arg(s.Hash),
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

// FindIamScimToken retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamScimToken(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamScimToken, error) {
	if len(cols) == 0 {
		return IamScimTokens.Query(
			sm.Where(IamScimTokens.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamScimTokens.Query(
		sm.Where(IamScimTokens.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamScimTokens.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamScimTokenExists checks the presence of a single record by primary key
func IamScimTokenExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamScimTokens.Query(
		sm.Where(IamScimTokens.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamScimToken is retrieved from the database
func (o *IamScimToken) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamScimTokens.AfterSelectHooks.RunHooks(ctx, exec, IamScimTokenSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamScimTokens.AfterInsertHooks.RunHooks(ctx, exec, IamScimTokenSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamScimTokens.AfterUpdateHooks.RunHooks(ctx, exec, IamScimTokenSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamScimTokens.AfterDeleteHooks.RunHooks(ctx, exec, IamScimTokenSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamScimTokens.AfterMergeHooks.RunHooks(ctx, exec, IamScimTokenSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamScimToken
func (o *IamScimToken) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamScimToken) pkEQ() dialect.Expression {
	return psql.Quote("iam_scim_tokens", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamScimToken
func (o *IamScimToken) Update(ctx context.Context, exec bob.Executor, s *IamScimTokenSetter) error {
	v, err := IamScimTokens.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamScimToken record with an executor
func (o *IamScimToken) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamScimTokens.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamScimToken using the executor
func (o *IamScimToken) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamScimTokens.Query(
		sm.Where(IamScimTokens.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamScimTokenSlice is retrieved from the database
func (o IamScimTokenSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamScimTokens.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamScimTokens.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamScimTokens.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamScimTokens.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamScimTokens.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamScimTokenSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_scim_tokens", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamScimTokenSlice) copyMatchingRows(from ...*IamScimToken) {
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
func (o IamScimTokenSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamScimTokens.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamScimToken:
				o.copyMatchingRows(retrieved)
			case []*IamScimToken:
				o.copyMatchingRows(retrieved...)
			case IamScimTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamScimToken or a slice of IamScimToken
				// then run the AfterUpdateHooks on the slice
				_, err = IamScimTokens.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamScimTokenSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamScimTokens.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamScimToken:
				o.copyMatchingRows(retrieved)
			case []*IamScimToken:
				o.copyMatchingRows(retrieved...)
			case IamScimTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamScimToken or a slice of IamScimToken
				// then run the AfterDeleteHooks on the slice
				_, err = IamScimTokens.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamScimTokenSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamScimTokens.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamScimToken:
				o.copyMatchingRows(retrieved)
			case []*IamScimToken:
				o.copyMatchingRows(retrieved...)
			case IamScimTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamScimToken or a slice of IamScimToken
				// then run the AfterMergeHooks on the slice
				_, err = IamScimTokens.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamScimTokenSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamScimTokenSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamScimTokens.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamScimTokenSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamScimTokens.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamScimTokenSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamScimTokens.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamScimTokenWhere[Q psql.Filterable] struct {
	ID           psql.WhereMod[Q, string]
	ProjectID    psql.WhereMod[Q, string]
	ConnectionID psql.WhereMod[Q, string]
	Hash         psql.WhereMod[Q, string]
	CreatedAt    psql.WhereMod[Q, time.Time]
	Data         psql.WhereMod[Q, json.RawMessage]
}

func (iamScimTokenWhere[Q]) AliasedAs(alias string) iamScimTokenWhere[Q] {
	return buildIamScimTokenWhere[Q](buildIamScimTokenColumns(alias))
}

func buildIamScimTokenWhere[Q psql.Filterable](cols iamScimTokenColumns) iamScimTokenWhere[Q] {
	return iamScimTokenWhere[Q]{
		ID:           psql.Where[Q, string](cols.ID.Expression),
		ProjectID:    psql.Where[Q, string](cols.ProjectID.Expression),
		ConnectionID: psql.Where[Q, string](cols.ConnectionID.Expression),
		Hash:         psql.Where[Q, string](cols.Hash.Expression),
		CreatedAt:    psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:         psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
