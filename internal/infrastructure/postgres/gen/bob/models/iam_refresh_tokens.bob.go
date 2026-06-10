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

// IamRefreshToken is an object representing the database table.
type IamRefreshToken struct {
	ID          string              `db:"id,pk" `
	ProjectID   string              `db:"project_id" `
	Environment string              `db:"environment" `
	UserID      string              `db:"user_id" `
	SessionID   string              `db:"session_id" `
	Hash        string              `db:"hash" `
	Revoked     bool                `db:"revoked" `
	ExpiresAt   null.Val[time.Time] `db:"expires_at" `
	CreatedAt   time.Time           `db:"created_at" `
	Data        json.RawMessage     `db:"data" `
}

// IamRefreshTokenSlice is an alias for a slice of pointers to IamRefreshToken.
// This should almost always be used instead of []*IamRefreshToken.
type IamRefreshTokenSlice []*IamRefreshToken

// IamRefreshTokens contains methods to work with the iam_refresh_tokens table
var IamRefreshTokens = psql.NewTablex[*IamRefreshToken, IamRefreshTokenSlice, *IamRefreshTokenSetter]("", "iam_refresh_tokens", buildIamRefreshTokenColumns("iam_refresh_tokens"))

// IamRefreshTokensQuery is a query on the iam_refresh_tokens table
type IamRefreshTokensQuery = *psql.ViewQuery[*IamRefreshToken, IamRefreshTokenSlice]

func buildIamRefreshTokenColumns(tableName string) iamRefreshTokenColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "user_id", "session_id", "hash", "revoked", "expires_at", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamRefreshTokenColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamRefreshTokenColumn(tableName, "id"),
		ProjectID:   buildIamRefreshTokenColumn(tableName, "project_id"),
		Environment: buildIamRefreshTokenColumn(tableName, "environment"),
		UserID:      buildIamRefreshTokenColumn(tableName, "user_id"),
		SessionID:   buildIamRefreshTokenColumn(tableName, "session_id"),
		Hash:        buildIamRefreshTokenColumn(tableName, "hash"),
		Revoked:     buildIamRefreshTokenColumn(tableName, "revoked"),
		ExpiresAt:   buildIamRefreshTokenColumn(tableName, "expires_at"),
		CreatedAt:   buildIamRefreshTokenColumn(tableName, "created_at"),
		Data:        buildIamRefreshTokenColumn(tableName, "data"),
	}
}

type iamRefreshTokenColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamRefreshTokenColumn
	ProjectID   iamRefreshTokenColumn
	Environment iamRefreshTokenColumn
	UserID      iamRefreshTokenColumn
	SessionID   iamRefreshTokenColumn
	Hash        iamRefreshTokenColumn
	Revoked     iamRefreshTokenColumn
	ExpiresAt   iamRefreshTokenColumn
	CreatedAt   iamRefreshTokenColumn
	Data        iamRefreshTokenColumn
}

// Alias returns the current table alias for the columns set.
func (c iamRefreshTokenColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamRefreshTokenColumns) AliasedAs(tableName string) iamRefreshTokenColumns {
	return buildIamRefreshTokenColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamRefreshTokenColumns) Unqualified() iamRefreshTokenColumns {
	return buildIamRefreshTokenColumns("")
}

func buildIamRefreshTokenColumn(alias, name string) iamRefreshTokenColumn {
	return iamRefreshTokenColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamRefreshTokenColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamRefreshTokenColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamRefreshTokenColumn) ShouldOmitParens() bool {
	return true
}

// IamRefreshTokenSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamRefreshTokenSetter struct {
	ID          *string              `db:"id,pk" `
	ProjectID   *string              `db:"project_id" `
	Environment *string              `db:"environment" `
	UserID      *string              `db:"user_id" `
	SessionID   *string              `db:"session_id" `
	Hash        *string              `db:"hash" `
	Revoked     *bool                `db:"revoked" `
	ExpiresAt   *null.Val[time.Time] `db:"expires_at" `
	CreatedAt   *time.Time           `db:"created_at" `
	Data        *json.RawMessage     `db:"data" `
}

func (s IamRefreshTokenSetter) SetColumns() []string {
	vals := make([]string, 0, 10)
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
	if s.SessionID != nil {
		vals = append(vals, "session_id")
	}
	if s.Hash != nil {
		vals = append(vals, "hash")
	}
	if s.Revoked != nil {
		vals = append(vals, "revoked")
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

func (s IamRefreshTokenSetter) Overwrite(t *IamRefreshToken) {
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
	if s.SessionID != nil {
		t.SessionID = func() string {
			if s.SessionID == nil {
				return *new(string)
			}
			return *s.SessionID
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
	if s.Revoked != nil {
		t.Revoked = func() bool {
			if s.Revoked == nil {
				return *new(bool)
			}
			return *s.Revoked
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

func (s *IamRefreshTokenSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamRefreshTokens.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 10)
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

		if s.SessionID != nil {
			vals[4] = psql.Arg(func() string {
				if s.SessionID == nil {
					return *new(string)
				}
				return *s.SessionID
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.Hash != nil {
			vals[5] = psql.Arg(func() string {
				if s.Hash == nil {
					return *new(string)
				}
				return *s.Hash
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.Revoked != nil {
			vals[6] = psql.Arg(func() bool {
				if s.Revoked == nil {
					return *new(bool)
				}
				return *s.Revoked
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[7] = psql.Arg(func() null.Val[time.Time] {
				if s.ExpiresAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.ExpiresAt
				return *v
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[8] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[8] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[9] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[9] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamRefreshTokenSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamRefreshTokenSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 10)

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

	if s.SessionID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "session_id")...),
			psql.Arg(s.SessionID),
		}})
	}

	if s.Hash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "hash")...),
			psql.Arg(s.Hash),
		}})
	}

	if s.Revoked != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "revoked")...),
			psql.Arg(s.Revoked),
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

// FindIamRefreshToken retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamRefreshToken(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamRefreshToken, error) {
	if len(cols) == 0 {
		return IamRefreshTokens.Query(
			sm.Where(IamRefreshTokens.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamRefreshTokens.Query(
		sm.Where(IamRefreshTokens.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamRefreshTokens.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamRefreshTokenExists checks the presence of a single record by primary key
func IamRefreshTokenExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamRefreshTokens.Query(
		sm.Where(IamRefreshTokens.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamRefreshToken is retrieved from the database
func (o *IamRefreshToken) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamRefreshTokens.AfterSelectHooks.RunHooks(ctx, exec, IamRefreshTokenSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamRefreshTokens.AfterInsertHooks.RunHooks(ctx, exec, IamRefreshTokenSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamRefreshTokens.AfterUpdateHooks.RunHooks(ctx, exec, IamRefreshTokenSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamRefreshTokens.AfterDeleteHooks.RunHooks(ctx, exec, IamRefreshTokenSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamRefreshTokens.AfterMergeHooks.RunHooks(ctx, exec, IamRefreshTokenSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamRefreshToken
func (o *IamRefreshToken) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamRefreshToken) pkEQ() dialect.Expression {
	return psql.Quote("iam_refresh_tokens", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamRefreshToken
func (o *IamRefreshToken) Update(ctx context.Context, exec bob.Executor, s *IamRefreshTokenSetter) error {
	v, err := IamRefreshTokens.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamRefreshToken record with an executor
func (o *IamRefreshToken) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamRefreshTokens.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamRefreshToken using the executor
func (o *IamRefreshToken) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamRefreshTokens.Query(
		sm.Where(IamRefreshTokens.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamRefreshTokenSlice is retrieved from the database
func (o IamRefreshTokenSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamRefreshTokens.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamRefreshTokens.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamRefreshTokens.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamRefreshTokens.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamRefreshTokens.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamRefreshTokenSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_refresh_tokens", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamRefreshTokenSlice) copyMatchingRows(from ...*IamRefreshToken) {
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
func (o IamRefreshTokenSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRefreshTokens.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRefreshToken:
				o.copyMatchingRows(retrieved)
			case []*IamRefreshToken:
				o.copyMatchingRows(retrieved...)
			case IamRefreshTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRefreshToken or a slice of IamRefreshToken
				// then run the AfterUpdateHooks on the slice
				_, err = IamRefreshTokens.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamRefreshTokenSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRefreshTokens.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRefreshToken:
				o.copyMatchingRows(retrieved)
			case []*IamRefreshToken:
				o.copyMatchingRows(retrieved...)
			case IamRefreshTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRefreshToken or a slice of IamRefreshToken
				// then run the AfterDeleteHooks on the slice
				_, err = IamRefreshTokens.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamRefreshTokenSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRefreshTokens.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRefreshToken:
				o.copyMatchingRows(retrieved)
			case []*IamRefreshToken:
				o.copyMatchingRows(retrieved...)
			case IamRefreshTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRefreshToken or a slice of IamRefreshToken
				// then run the AfterMergeHooks on the slice
				_, err = IamRefreshTokens.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamRefreshTokenSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamRefreshTokenSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamRefreshTokens.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamRefreshTokenSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamRefreshTokens.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamRefreshTokenSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamRefreshTokens.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamRefreshTokenWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	UserID      psql.WhereMod[Q, string]
	SessionID   psql.WhereMod[Q, string]
	Hash        psql.WhereMod[Q, string]
	Revoked     psql.WhereMod[Q, bool]
	ExpiresAt   psql.WhereNullMod[Q, time.Time]
	CreatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamRefreshTokenWhere[Q]) AliasedAs(alias string) iamRefreshTokenWhere[Q] {
	return buildIamRefreshTokenWhere[Q](buildIamRefreshTokenColumns(alias))
}

func buildIamRefreshTokenWhere[Q psql.Filterable](cols iamRefreshTokenColumns) iamRefreshTokenWhere[Q] {
	return iamRefreshTokenWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		UserID:      psql.Where[Q, string](cols.UserID.Expression),
		SessionID:   psql.Where[Q, string](cols.SessionID.Expression),
		Hash:        psql.Where[Q, string](cols.Hash.Expression),
		Revoked:     psql.Where[Q, bool](cols.Revoked.Expression),
		ExpiresAt:   psql.WhereNull[Q, time.Time](cols.ExpiresAt.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
