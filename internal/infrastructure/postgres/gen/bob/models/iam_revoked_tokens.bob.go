// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
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

// IamRevokedToken is an object representing the database table.
type IamRevokedToken struct {
	Jti         string    `db:"jti,pk" `
	ProjectID   string    `db:"project_id" `
	Environment string    `db:"environment" `
	ExpiresAt   time.Time `db:"expires_at" `
	RevokedAt   time.Time `db:"revoked_at" `
}

// IamRevokedTokenSlice is an alias for a slice of pointers to IamRevokedToken.
// This should almost always be used instead of []*IamRevokedToken.
type IamRevokedTokenSlice []*IamRevokedToken

// IamRevokedTokens contains methods to work with the iam_revoked_tokens table
var IamRevokedTokens = psql.NewTablex[*IamRevokedToken, IamRevokedTokenSlice, *IamRevokedTokenSetter]("", "iam_revoked_tokens", buildIamRevokedTokenColumns("iam_revoked_tokens"))

// IamRevokedTokensQuery is a query on the iam_revoked_tokens table
type IamRevokedTokensQuery = *psql.ViewQuery[*IamRevokedToken, IamRevokedTokenSlice]

func buildIamRevokedTokenColumns(tableName string) iamRevokedTokenColumns {
	columnsExpr := expr.NewColumnsExpr(
		"jti", "project_id", "environment", "expires_at", "revoked_at",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamRevokedTokenColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		Jti:         buildIamRevokedTokenColumn(tableName, "jti"),
		ProjectID:   buildIamRevokedTokenColumn(tableName, "project_id"),
		Environment: buildIamRevokedTokenColumn(tableName, "environment"),
		ExpiresAt:   buildIamRevokedTokenColumn(tableName, "expires_at"),
		RevokedAt:   buildIamRevokedTokenColumn(tableName, "revoked_at"),
	}
}

type iamRevokedTokenColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	Jti         iamRevokedTokenColumn
	ProjectID   iamRevokedTokenColumn
	Environment iamRevokedTokenColumn
	ExpiresAt   iamRevokedTokenColumn
	RevokedAt   iamRevokedTokenColumn
}

// Alias returns the current table alias for the columns set.
func (c iamRevokedTokenColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamRevokedTokenColumns) AliasedAs(tableName string) iamRevokedTokenColumns {
	return buildIamRevokedTokenColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamRevokedTokenColumns) Unqualified() iamRevokedTokenColumns {
	return buildIamRevokedTokenColumns("")
}

func buildIamRevokedTokenColumn(alias, name string) iamRevokedTokenColumn {
	return iamRevokedTokenColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamRevokedTokenColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamRevokedTokenColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamRevokedTokenColumn) ShouldOmitParens() bool {
	return true
}

// IamRevokedTokenSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamRevokedTokenSetter struct {
	Jti         *string    `db:"jti,pk" `
	ProjectID   *string    `db:"project_id" `
	Environment *string    `db:"environment" `
	ExpiresAt   *time.Time `db:"expires_at" `
	RevokedAt   *time.Time `db:"revoked_at" `
}

func (s IamRevokedTokenSetter) SetColumns() []string {
	vals := make([]string, 0, 5)
	if s.Jti != nil {
		vals = append(vals, "jti")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
	}
	if s.RevokedAt != nil {
		vals = append(vals, "revoked_at")
	}
	return vals
}

func (s IamRevokedTokenSetter) Overwrite(t *IamRevokedToken) {
	if s.Jti != nil {
		t.Jti = func() string {
			if s.Jti == nil {
				return *new(string)
			}
			return *s.Jti
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
	if s.ExpiresAt != nil {
		t.ExpiresAt = func() time.Time {
			if s.ExpiresAt == nil {
				return *new(time.Time)
			}
			return *s.ExpiresAt
		}()
	}
	if s.RevokedAt != nil {
		t.RevokedAt = func() time.Time {
			if s.RevokedAt == nil {
				return *new(time.Time)
			}
			return *s.RevokedAt
		}()
	}
}

func (s *IamRevokedTokenSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamRevokedTokens.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 5)
		if s.Jti != nil {
			vals[0] = psql.Arg(func() string {
				if s.Jti == nil {
					return *new(string)
				}
				return *s.Jti
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

		if s.ExpiresAt != nil {
			vals[3] = psql.Arg(func() time.Time {
				if s.ExpiresAt == nil {
					return *new(time.Time)
				}
				return *s.ExpiresAt
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.RevokedAt != nil {
			vals[4] = psql.Arg(func() time.Time {
				if s.RevokedAt == nil {
					return *new(time.Time)
				}
				return *s.RevokedAt
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamRevokedTokenSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamRevokedTokenSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 5)

	if s.Jti != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "jti")...),
			psql.Arg(s.Jti),
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

	if s.ExpiresAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "expires_at")...),
			psql.Arg(s.ExpiresAt),
		}})
	}

	if s.RevokedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "revoked_at")...),
			psql.Arg(s.RevokedAt),
		}})
	}

	return exprs
}

// FindIamRevokedToken retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamRevokedToken(ctx context.Context, exec bob.Executor, JtiPK string, cols ...string) (*IamRevokedToken, error) {
	if len(cols) == 0 {
		return IamRevokedTokens.Query(
			sm.Where(IamRevokedTokens.Columns.Jti.EQ(psql.Arg(JtiPK))),
		).One(ctx, exec)
	}

	return IamRevokedTokens.Query(
		sm.Where(IamRevokedTokens.Columns.Jti.EQ(psql.Arg(JtiPK))),
		sm.Columns(IamRevokedTokens.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamRevokedTokenExists checks the presence of a single record by primary key
func IamRevokedTokenExists(ctx context.Context, exec bob.Executor, JtiPK string) (bool, error) {
	return IamRevokedTokens.Query(
		sm.Where(IamRevokedTokens.Columns.Jti.EQ(psql.Arg(JtiPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamRevokedToken is retrieved from the database
func (o *IamRevokedToken) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamRevokedTokens.AfterSelectHooks.RunHooks(ctx, exec, IamRevokedTokenSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamRevokedTokens.AfterInsertHooks.RunHooks(ctx, exec, IamRevokedTokenSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamRevokedTokens.AfterUpdateHooks.RunHooks(ctx, exec, IamRevokedTokenSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamRevokedTokens.AfterDeleteHooks.RunHooks(ctx, exec, IamRevokedTokenSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamRevokedTokens.AfterMergeHooks.RunHooks(ctx, exec, IamRevokedTokenSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamRevokedToken
func (o *IamRevokedToken) primaryKeyVals() bob.Expression {
	return psql.Arg(o.Jti)
}

func (o *IamRevokedToken) pkEQ() dialect.Expression {
	return psql.Quote("iam_revoked_tokens", "jti").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamRevokedToken
func (o *IamRevokedToken) Update(ctx context.Context, exec bob.Executor, s *IamRevokedTokenSetter) error {
	v, err := IamRevokedTokens.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamRevokedToken record with an executor
func (o *IamRevokedToken) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamRevokedTokens.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamRevokedToken using the executor
func (o *IamRevokedToken) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamRevokedTokens.Query(
		sm.Where(IamRevokedTokens.Columns.Jti.EQ(psql.Arg(o.Jti))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamRevokedTokenSlice is retrieved from the database
func (o IamRevokedTokenSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamRevokedTokens.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamRevokedTokens.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamRevokedTokens.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamRevokedTokens.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamRevokedTokens.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamRevokedTokenSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_revoked_tokens", "jti").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamRevokedTokenSlice) copyMatchingRows(from ...*IamRevokedToken) {
	for i, old := range o {
		for _, new := range from {
			if new.Jti != old.Jti {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamRevokedTokenSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRevokedTokens.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRevokedToken:
				o.copyMatchingRows(retrieved)
			case []*IamRevokedToken:
				o.copyMatchingRows(retrieved...)
			case IamRevokedTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRevokedToken or a slice of IamRevokedToken
				// then run the AfterUpdateHooks on the slice
				_, err = IamRevokedTokens.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamRevokedTokenSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRevokedTokens.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRevokedToken:
				o.copyMatchingRows(retrieved)
			case []*IamRevokedToken:
				o.copyMatchingRows(retrieved...)
			case IamRevokedTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRevokedToken or a slice of IamRevokedToken
				// then run the AfterDeleteHooks on the slice
				_, err = IamRevokedTokens.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamRevokedTokenSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRevokedTokens.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRevokedToken:
				o.copyMatchingRows(retrieved)
			case []*IamRevokedToken:
				o.copyMatchingRows(retrieved...)
			case IamRevokedTokenSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRevokedToken or a slice of IamRevokedToken
				// then run the AfterMergeHooks on the slice
				_, err = IamRevokedTokens.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamRevokedTokenSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamRevokedTokenSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamRevokedTokens.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamRevokedTokenSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamRevokedTokens.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamRevokedTokenSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamRevokedTokens.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamRevokedTokenWhere[Q psql.Filterable] struct {
	Jti         psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	ExpiresAt   psql.WhereMod[Q, time.Time]
	RevokedAt   psql.WhereMod[Q, time.Time]
}

func (iamRevokedTokenWhere[Q]) AliasedAs(alias string) iamRevokedTokenWhere[Q] {
	return buildIamRevokedTokenWhere[Q](buildIamRevokedTokenColumns(alias))
}

func buildIamRevokedTokenWhere[Q psql.Filterable](cols iamRevokedTokenColumns) iamRevokedTokenWhere[Q] {
	return iamRevokedTokenWhere[Q]{
		Jti:         psql.Where[Q, string](cols.Jti.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		ExpiresAt:   psql.Where[Q, time.Time](cols.ExpiresAt.Expression),
		RevokedAt:   psql.Where[Q, time.Time](cols.RevokedAt.Expression),
	}
}
