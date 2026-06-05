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

// IamAPIKey is an object representing the database table.
type IamAPIKey struct {
	ID        string              `db:"id,pk" `
	ProjectID string              `db:"project_id" `
	Prefix    string              `db:"prefix" `
	Hash      string              `db:"hash" `
	Disabled  bool                `db:"disabled" `
	ExpiresAt null.Val[time.Time] `db:"expires_at" `
	CreatedAt time.Time           `db:"created_at" `
	Data      json.RawMessage     `db:"data" `
}

// IamAPIKeySlice is an alias for a slice of pointers to IamAPIKey.
// This should almost always be used instead of []*IamAPIKey.
type IamAPIKeySlice []*IamAPIKey

// IamAPIKeys contains methods to work with the iam_api_keys table
var IamAPIKeys = psql.NewTablex[*IamAPIKey, IamAPIKeySlice, *IamAPIKeySetter]("", "iam_api_keys", buildIamAPIKeyColumns("iam_api_keys"))

// IamAPIKeysQuery is a query on the iam_api_keys table
type IamAPIKeysQuery = *psql.ViewQuery[*IamAPIKey, IamAPIKeySlice]

func buildIamAPIKeyColumns(tableName string) iamAPIKeyColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "prefix", "hash", "disabled", "expires_at", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamAPIKeyColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamAPIKeyColumn(tableName, "id"),
		ProjectID:   buildIamAPIKeyColumn(tableName, "project_id"),
		Prefix:      buildIamAPIKeyColumn(tableName, "prefix"),
		Hash:        buildIamAPIKeyColumn(tableName, "hash"),
		Disabled:    buildIamAPIKeyColumn(tableName, "disabled"),
		ExpiresAt:   buildIamAPIKeyColumn(tableName, "expires_at"),
		CreatedAt:   buildIamAPIKeyColumn(tableName, "created_at"),
		Data:        buildIamAPIKeyColumn(tableName, "data"),
	}
}

type iamAPIKeyColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamAPIKeyColumn
	ProjectID  iamAPIKeyColumn
	Prefix     iamAPIKeyColumn
	Hash       iamAPIKeyColumn
	Disabled   iamAPIKeyColumn
	ExpiresAt  iamAPIKeyColumn
	CreatedAt  iamAPIKeyColumn
	Data       iamAPIKeyColumn
}

// Alias returns the current table alias for the columns set.
func (c iamAPIKeyColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamAPIKeyColumns) AliasedAs(tableName string) iamAPIKeyColumns {
	return buildIamAPIKeyColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamAPIKeyColumns) Unqualified() iamAPIKeyColumns {
	return buildIamAPIKeyColumns("")
}

func buildIamAPIKeyColumn(alias, name string) iamAPIKeyColumn {
	return iamAPIKeyColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamAPIKeyColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamAPIKeyColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamAPIKeyColumn) ShouldOmitParens() bool {
	return true
}

// IamAPIKeySetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamAPIKeySetter struct {
	ID        *string              `db:"id,pk" `
	ProjectID *string              `db:"project_id" `
	Prefix    *string              `db:"prefix" `
	Hash      *string              `db:"hash" `
	Disabled  *bool                `db:"disabled" `
	ExpiresAt *null.Val[time.Time] `db:"expires_at" `
	CreatedAt *time.Time           `db:"created_at" `
	Data      *json.RawMessage     `db:"data" `
}

func (s IamAPIKeySetter) SetColumns() []string {
	vals := make([]string, 0, 8)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Prefix != nil {
		vals = append(vals, "prefix")
	}
	if s.Hash != nil {
		vals = append(vals, "hash")
	}
	if s.Disabled != nil {
		vals = append(vals, "disabled")
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

func (s IamAPIKeySetter) Overwrite(t *IamAPIKey) {
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
	if s.Prefix != nil {
		t.Prefix = func() string {
			if s.Prefix == nil {
				return *new(string)
			}
			return *s.Prefix
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
	if s.Disabled != nil {
		t.Disabled = func() bool {
			if s.Disabled == nil {
				return *new(bool)
			}
			return *s.Disabled
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

func (s *IamAPIKeySetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamAPIKeys.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 8)
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

		if s.Prefix != nil {
			vals[2] = psql.Arg(func() string {
				if s.Prefix == nil {
					return *new(string)
				}
				return *s.Prefix
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

		if s.Disabled != nil {
			vals[4] = psql.Arg(func() bool {
				if s.Disabled == nil {
					return *new(bool)
				}
				return *s.Disabled
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[5] = psql.Arg(func() null.Val[time.Time] {
				if s.ExpiresAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.ExpiresAt
				return *v
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[6] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[7] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamAPIKeySetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamAPIKeySetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 8)

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

	if s.Prefix != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "prefix")...),
			psql.Arg(s.Prefix),
		}})
	}

	if s.Hash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "hash")...),
			psql.Arg(s.Hash),
		}})
	}

	if s.Disabled != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "disabled")...),
			psql.Arg(s.Disabled),
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

// FindIamAPIKey retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamAPIKey(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamAPIKey, error) {
	if len(cols) == 0 {
		return IamAPIKeys.Query(
			sm.Where(IamAPIKeys.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamAPIKeys.Query(
		sm.Where(IamAPIKeys.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamAPIKeys.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamAPIKeyExists checks the presence of a single record by primary key
func IamAPIKeyExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamAPIKeys.Query(
		sm.Where(IamAPIKeys.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamAPIKey is retrieved from the database
func (o *IamAPIKey) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAPIKeys.AfterSelectHooks.RunHooks(ctx, exec, IamAPIKeySlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamAPIKeys.AfterInsertHooks.RunHooks(ctx, exec, IamAPIKeySlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamAPIKeys.AfterUpdateHooks.RunHooks(ctx, exec, IamAPIKeySlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamAPIKeys.AfterDeleteHooks.RunHooks(ctx, exec, IamAPIKeySlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamAPIKeys.AfterMergeHooks.RunHooks(ctx, exec, IamAPIKeySlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamAPIKey
func (o *IamAPIKey) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamAPIKey) pkEQ() dialect.Expression {
	return psql.Quote("iam_api_keys", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamAPIKey
func (o *IamAPIKey) Update(ctx context.Context, exec bob.Executor, s *IamAPIKeySetter) error {
	v, err := IamAPIKeys.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamAPIKey record with an executor
func (o *IamAPIKey) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamAPIKeys.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamAPIKey using the executor
func (o *IamAPIKey) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamAPIKeys.Query(
		sm.Where(IamAPIKeys.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamAPIKeySlice is retrieved from the database
func (o IamAPIKeySlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAPIKeys.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamAPIKeys.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamAPIKeys.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamAPIKeys.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamAPIKeys.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamAPIKeySlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_api_keys", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamAPIKeySlice) copyMatchingRows(from ...*IamAPIKey) {
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
func (o IamAPIKeySlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAPIKeys.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAPIKey:
				o.copyMatchingRows(retrieved)
			case []*IamAPIKey:
				o.copyMatchingRows(retrieved...)
			case IamAPIKeySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAPIKey or a slice of IamAPIKey
				// then run the AfterUpdateHooks on the slice
				_, err = IamAPIKeys.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamAPIKeySlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAPIKeys.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAPIKey:
				o.copyMatchingRows(retrieved)
			case []*IamAPIKey:
				o.copyMatchingRows(retrieved...)
			case IamAPIKeySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAPIKey or a slice of IamAPIKey
				// then run the AfterDeleteHooks on the slice
				_, err = IamAPIKeys.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamAPIKeySlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAPIKeys.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAPIKey:
				o.copyMatchingRows(retrieved)
			case []*IamAPIKey:
				o.copyMatchingRows(retrieved...)
			case IamAPIKeySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAPIKey or a slice of IamAPIKey
				// then run the AfterMergeHooks on the slice
				_, err = IamAPIKeys.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamAPIKeySlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamAPIKeySetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAPIKeys.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamAPIKeySlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAPIKeys.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamAPIKeySlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamAPIKeys.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamAPIKeyWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Prefix    psql.WhereMod[Q, string]
	Hash      psql.WhereMod[Q, string]
	Disabled  psql.WhereMod[Q, bool]
	ExpiresAt psql.WhereNullMod[Q, time.Time]
	CreatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamAPIKeyWhere[Q]) AliasedAs(alias string) iamAPIKeyWhere[Q] {
	return buildIamAPIKeyWhere[Q](buildIamAPIKeyColumns(alias))
}

func buildIamAPIKeyWhere[Q psql.Filterable](cols iamAPIKeyColumns) iamAPIKeyWhere[Q] {
	return iamAPIKeyWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Prefix:    psql.Where[Q, string](cols.Prefix.Expression),
		Hash:      psql.Where[Q, string](cols.Hash.Expression),
		Disabled:  psql.Where[Q, bool](cols.Disabled.Expression),
		ExpiresAt: psql.WhereNull[Q, time.Time](cols.ExpiresAt.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
