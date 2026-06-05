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

// IamOauthGrant is an object representing the database table.
type IamOauthGrant struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	UserID    string          `db:"user_id" `
	ClientID  string          `db:"client_id" `
	GrantedAt time.Time       `db:"granted_at" `
	Data      json.RawMessage `db:"data" `
}

// IamOauthGrantSlice is an alias for a slice of pointers to IamOauthGrant.
// This should almost always be used instead of []*IamOauthGrant.
type IamOauthGrantSlice []*IamOauthGrant

// IamOauthGrants contains methods to work with the iam_oauth_grants table
var IamOauthGrants = psql.NewTablex[*IamOauthGrant, IamOauthGrantSlice, *IamOauthGrantSetter]("", "iam_oauth_grants", buildIamOauthGrantColumns("iam_oauth_grants"))

// IamOauthGrantsQuery is a query on the iam_oauth_grants table
type IamOauthGrantsQuery = *psql.ViewQuery[*IamOauthGrant, IamOauthGrantSlice]

func buildIamOauthGrantColumns(tableName string) iamOauthGrantColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "user_id", "client_id", "granted_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamOauthGrantColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamOauthGrantColumn(tableName, "id"),
		ProjectID:   buildIamOauthGrantColumn(tableName, "project_id"),
		UserID:      buildIamOauthGrantColumn(tableName, "user_id"),
		ClientID:    buildIamOauthGrantColumn(tableName, "client_id"),
		GrantedAt:   buildIamOauthGrantColumn(tableName, "granted_at"),
		Data:        buildIamOauthGrantColumn(tableName, "data"),
	}
}

type iamOauthGrantColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamOauthGrantColumn
	ProjectID  iamOauthGrantColumn
	UserID     iamOauthGrantColumn
	ClientID   iamOauthGrantColumn
	GrantedAt  iamOauthGrantColumn
	Data       iamOauthGrantColumn
}

// Alias returns the current table alias for the columns set.
func (c iamOauthGrantColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamOauthGrantColumns) AliasedAs(tableName string) iamOauthGrantColumns {
	return buildIamOauthGrantColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamOauthGrantColumns) Unqualified() iamOauthGrantColumns {
	return buildIamOauthGrantColumns("")
}

func buildIamOauthGrantColumn(alias, name string) iamOauthGrantColumn {
	return iamOauthGrantColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamOauthGrantColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamOauthGrantColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamOauthGrantColumn) ShouldOmitParens() bool {
	return true
}

// IamOauthGrantSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamOauthGrantSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	UserID    *string          `db:"user_id" `
	ClientID  *string          `db:"client_id" `
	GrantedAt *time.Time       `db:"granted_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamOauthGrantSetter) SetColumns() []string {
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
	if s.ClientID != nil {
		vals = append(vals, "client_id")
	}
	if s.GrantedAt != nil {
		vals = append(vals, "granted_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamOauthGrantSetter) Overwrite(t *IamOauthGrant) {
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
	if s.ClientID != nil {
		t.ClientID = func() string {
			if s.ClientID == nil {
				return *new(string)
			}
			return *s.ClientID
		}()
	}
	if s.GrantedAt != nil {
		t.GrantedAt = func() time.Time {
			if s.GrantedAt == nil {
				return *new(time.Time)
			}
			return *s.GrantedAt
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

func (s *IamOauthGrantSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamOauthGrants.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.ClientID != nil {
			vals[3] = psql.Arg(func() string {
				if s.ClientID == nil {
					return *new(string)
				}
				return *s.ClientID
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.GrantedAt != nil {
			vals[4] = psql.Arg(func() time.Time {
				if s.GrantedAt == nil {
					return *new(time.Time)
				}
				return *s.GrantedAt
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

func (s IamOauthGrantSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamOauthGrantSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.ClientID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "client_id")...),
			psql.Arg(s.ClientID),
		}})
	}

	if s.GrantedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "granted_at")...),
			psql.Arg(s.GrantedAt),
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

// FindIamOauthGrant retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamOauthGrant(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamOauthGrant, error) {
	if len(cols) == 0 {
		return IamOauthGrants.Query(
			sm.Where(IamOauthGrants.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamOauthGrants.Query(
		sm.Where(IamOauthGrants.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamOauthGrants.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamOauthGrantExists checks the presence of a single record by primary key
func IamOauthGrantExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamOauthGrants.Query(
		sm.Where(IamOauthGrants.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamOauthGrant is retrieved from the database
func (o *IamOauthGrant) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamOauthGrants.AfterSelectHooks.RunHooks(ctx, exec, IamOauthGrantSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamOauthGrants.AfterInsertHooks.RunHooks(ctx, exec, IamOauthGrantSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamOauthGrants.AfterUpdateHooks.RunHooks(ctx, exec, IamOauthGrantSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamOauthGrants.AfterDeleteHooks.RunHooks(ctx, exec, IamOauthGrantSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamOauthGrants.AfterMergeHooks.RunHooks(ctx, exec, IamOauthGrantSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamOauthGrant
func (o *IamOauthGrant) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamOauthGrant) pkEQ() dialect.Expression {
	return psql.Quote("iam_oauth_grants", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamOauthGrant
func (o *IamOauthGrant) Update(ctx context.Context, exec bob.Executor, s *IamOauthGrantSetter) error {
	v, err := IamOauthGrants.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamOauthGrant record with an executor
func (o *IamOauthGrant) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamOauthGrants.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamOauthGrant using the executor
func (o *IamOauthGrant) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamOauthGrants.Query(
		sm.Where(IamOauthGrants.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamOauthGrantSlice is retrieved from the database
func (o IamOauthGrantSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamOauthGrants.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamOauthGrants.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamOauthGrants.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamOauthGrants.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamOauthGrants.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamOauthGrantSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_oauth_grants", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamOauthGrantSlice) copyMatchingRows(from ...*IamOauthGrant) {
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
func (o IamOauthGrantSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamOauthGrants.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamOauthGrant:
				o.copyMatchingRows(retrieved)
			case []*IamOauthGrant:
				o.copyMatchingRows(retrieved...)
			case IamOauthGrantSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamOauthGrant or a slice of IamOauthGrant
				// then run the AfterUpdateHooks on the slice
				_, err = IamOauthGrants.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamOauthGrantSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamOauthGrants.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamOauthGrant:
				o.copyMatchingRows(retrieved)
			case []*IamOauthGrant:
				o.copyMatchingRows(retrieved...)
			case IamOauthGrantSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamOauthGrant or a slice of IamOauthGrant
				// then run the AfterDeleteHooks on the slice
				_, err = IamOauthGrants.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamOauthGrantSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamOauthGrants.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamOauthGrant:
				o.copyMatchingRows(retrieved)
			case []*IamOauthGrant:
				o.copyMatchingRows(retrieved...)
			case IamOauthGrantSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamOauthGrant or a slice of IamOauthGrant
				// then run the AfterMergeHooks on the slice
				_, err = IamOauthGrants.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamOauthGrantSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamOauthGrantSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamOauthGrants.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamOauthGrantSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamOauthGrants.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamOauthGrantSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamOauthGrants.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamOauthGrantWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	UserID    psql.WhereMod[Q, string]
	ClientID  psql.WhereMod[Q, string]
	GrantedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamOauthGrantWhere[Q]) AliasedAs(alias string) iamOauthGrantWhere[Q] {
	return buildIamOauthGrantWhere[Q](buildIamOauthGrantColumns(alias))
}

func buildIamOauthGrantWhere[Q psql.Filterable](cols iamOauthGrantColumns) iamOauthGrantWhere[Q] {
	return iamOauthGrantWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		UserID:    psql.Where[Q, string](cols.UserID.Expression),
		ClientID:  psql.Where[Q, string](cols.ClientID.Expression),
		GrantedAt: psql.Where[Q, time.Time](cols.GrantedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
