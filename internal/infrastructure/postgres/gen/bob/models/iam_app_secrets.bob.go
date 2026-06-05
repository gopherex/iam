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

// IamAppSecret is an object representing the database table.
type IamAppSecret struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	AppID     string          `db:"app_id" `
	Hash      string          `db:"hash" `
	CreatedAt time.Time       `db:"created_at" `
	Data      json.RawMessage `db:"data" `
}

// IamAppSecretSlice is an alias for a slice of pointers to IamAppSecret.
// This should almost always be used instead of []*IamAppSecret.
type IamAppSecretSlice []*IamAppSecret

// IamAppSecrets contains methods to work with the iam_app_secrets table
var IamAppSecrets = psql.NewTablex[*IamAppSecret, IamAppSecretSlice, *IamAppSecretSetter]("", "iam_app_secrets", buildIamAppSecretColumns("iam_app_secrets"))

// IamAppSecretsQuery is a query on the iam_app_secrets table
type IamAppSecretsQuery = *psql.ViewQuery[*IamAppSecret, IamAppSecretSlice]

func buildIamAppSecretColumns(tableName string) iamAppSecretColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "app_id", "hash", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamAppSecretColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamAppSecretColumn(tableName, "id"),
		ProjectID:   buildIamAppSecretColumn(tableName, "project_id"),
		AppID:       buildIamAppSecretColumn(tableName, "app_id"),
		Hash:        buildIamAppSecretColumn(tableName, "hash"),
		CreatedAt:   buildIamAppSecretColumn(tableName, "created_at"),
		Data:        buildIamAppSecretColumn(tableName, "data"),
	}
}

type iamAppSecretColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamAppSecretColumn
	ProjectID  iamAppSecretColumn
	AppID      iamAppSecretColumn
	Hash       iamAppSecretColumn
	CreatedAt  iamAppSecretColumn
	Data       iamAppSecretColumn
}

// Alias returns the current table alias for the columns set.
func (c iamAppSecretColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamAppSecretColumns) AliasedAs(tableName string) iamAppSecretColumns {
	return buildIamAppSecretColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamAppSecretColumns) Unqualified() iamAppSecretColumns {
	return buildIamAppSecretColumns("")
}

func buildIamAppSecretColumn(alias, name string) iamAppSecretColumn {
	return iamAppSecretColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamAppSecretColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamAppSecretColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamAppSecretColumn) ShouldOmitParens() bool {
	return true
}

// IamAppSecretSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamAppSecretSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	AppID     *string          `db:"app_id" `
	Hash      *string          `db:"hash" `
	CreatedAt *time.Time       `db:"created_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamAppSecretSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.AppID != nil {
		vals = append(vals, "app_id")
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

func (s IamAppSecretSetter) Overwrite(t *IamAppSecret) {
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
	if s.AppID != nil {
		t.AppID = func() string {
			if s.AppID == nil {
				return *new(string)
			}
			return *s.AppID
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

func (s *IamAppSecretSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamAppSecrets.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.AppID != nil {
			vals[2] = psql.Arg(func() string {
				if s.AppID == nil {
					return *new(string)
				}
				return *s.AppID
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

func (s IamAppSecretSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamAppSecretSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.AppID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "app_id")...),
			psql.Arg(s.AppID),
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

// FindIamAppSecret retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamAppSecret(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamAppSecret, error) {
	if len(cols) == 0 {
		return IamAppSecrets.Query(
			sm.Where(IamAppSecrets.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamAppSecrets.Query(
		sm.Where(IamAppSecrets.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamAppSecrets.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamAppSecretExists checks the presence of a single record by primary key
func IamAppSecretExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamAppSecrets.Query(
		sm.Where(IamAppSecrets.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamAppSecret is retrieved from the database
func (o *IamAppSecret) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAppSecrets.AfterSelectHooks.RunHooks(ctx, exec, IamAppSecretSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamAppSecrets.AfterInsertHooks.RunHooks(ctx, exec, IamAppSecretSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamAppSecrets.AfterUpdateHooks.RunHooks(ctx, exec, IamAppSecretSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamAppSecrets.AfterDeleteHooks.RunHooks(ctx, exec, IamAppSecretSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamAppSecrets.AfterMergeHooks.RunHooks(ctx, exec, IamAppSecretSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamAppSecret
func (o *IamAppSecret) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamAppSecret) pkEQ() dialect.Expression {
	return psql.Quote("iam_app_secrets", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamAppSecret
func (o *IamAppSecret) Update(ctx context.Context, exec bob.Executor, s *IamAppSecretSetter) error {
	v, err := IamAppSecrets.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamAppSecret record with an executor
func (o *IamAppSecret) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamAppSecrets.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamAppSecret using the executor
func (o *IamAppSecret) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamAppSecrets.Query(
		sm.Where(IamAppSecrets.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamAppSecretSlice is retrieved from the database
func (o IamAppSecretSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAppSecrets.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamAppSecrets.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamAppSecrets.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamAppSecrets.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamAppSecrets.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamAppSecretSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_app_secrets", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamAppSecretSlice) copyMatchingRows(from ...*IamAppSecret) {
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
func (o IamAppSecretSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAppSecrets.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAppSecret:
				o.copyMatchingRows(retrieved)
			case []*IamAppSecret:
				o.copyMatchingRows(retrieved...)
			case IamAppSecretSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAppSecret or a slice of IamAppSecret
				// then run the AfterUpdateHooks on the slice
				_, err = IamAppSecrets.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamAppSecretSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAppSecrets.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAppSecret:
				o.copyMatchingRows(retrieved)
			case []*IamAppSecret:
				o.copyMatchingRows(retrieved...)
			case IamAppSecretSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAppSecret or a slice of IamAppSecret
				// then run the AfterDeleteHooks on the slice
				_, err = IamAppSecrets.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamAppSecretSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAppSecrets.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAppSecret:
				o.copyMatchingRows(retrieved)
			case []*IamAppSecret:
				o.copyMatchingRows(retrieved...)
			case IamAppSecretSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAppSecret or a slice of IamAppSecret
				// then run the AfterMergeHooks on the slice
				_, err = IamAppSecrets.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamAppSecretSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamAppSecretSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAppSecrets.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamAppSecretSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAppSecrets.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamAppSecretSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamAppSecrets.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamAppSecretWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	AppID     psql.WhereMod[Q, string]
	Hash      psql.WhereMod[Q, string]
	CreatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamAppSecretWhere[Q]) AliasedAs(alias string) iamAppSecretWhere[Q] {
	return buildIamAppSecretWhere[Q](buildIamAppSecretColumns(alias))
}

func buildIamAppSecretWhere[Q psql.Filterable](cols iamAppSecretColumns) iamAppSecretWhere[Q] {
	return iamAppSecretWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		AppID:     psql.Where[Q, string](cols.AppID.Expression),
		Hash:      psql.Where[Q, string](cols.Hash.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
