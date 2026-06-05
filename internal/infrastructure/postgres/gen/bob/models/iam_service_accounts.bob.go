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

// IamServiceAccount is an object representing the database table.
type IamServiceAccount struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	Name      string          `db:"name" `
	Disabled  bool            `db:"disabled" `
	CreatedAt time.Time       `db:"created_at" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamServiceAccountSlice is an alias for a slice of pointers to IamServiceAccount.
// This should almost always be used instead of []*IamServiceAccount.
type IamServiceAccountSlice []*IamServiceAccount

// IamServiceAccounts contains methods to work with the iam_service_accounts table
var IamServiceAccounts = psql.NewTablex[*IamServiceAccount, IamServiceAccountSlice, *IamServiceAccountSetter]("", "iam_service_accounts", buildIamServiceAccountColumns("iam_service_accounts"))

// IamServiceAccountsQuery is a query on the iam_service_accounts table
type IamServiceAccountsQuery = *psql.ViewQuery[*IamServiceAccount, IamServiceAccountSlice]

func buildIamServiceAccountColumns(tableName string) iamServiceAccountColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "name", "disabled", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamServiceAccountColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamServiceAccountColumn(tableName, "id"),
		ProjectID:   buildIamServiceAccountColumn(tableName, "project_id"),
		Name:        buildIamServiceAccountColumn(tableName, "name"),
		Disabled:    buildIamServiceAccountColumn(tableName, "disabled"),
		CreatedAt:   buildIamServiceAccountColumn(tableName, "created_at"),
		UpdatedAt:   buildIamServiceAccountColumn(tableName, "updated_at"),
		Data:        buildIamServiceAccountColumn(tableName, "data"),
	}
}

type iamServiceAccountColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamServiceAccountColumn
	ProjectID  iamServiceAccountColumn
	Name       iamServiceAccountColumn
	Disabled   iamServiceAccountColumn
	CreatedAt  iamServiceAccountColumn
	UpdatedAt  iamServiceAccountColumn
	Data       iamServiceAccountColumn
}

// Alias returns the current table alias for the columns set.
func (c iamServiceAccountColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamServiceAccountColumns) AliasedAs(tableName string) iamServiceAccountColumns {
	return buildIamServiceAccountColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamServiceAccountColumns) Unqualified() iamServiceAccountColumns {
	return buildIamServiceAccountColumns("")
}

func buildIamServiceAccountColumn(alias, name string) iamServiceAccountColumn {
	return iamServiceAccountColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamServiceAccountColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamServiceAccountColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamServiceAccountColumn) ShouldOmitParens() bool {
	return true
}

// IamServiceAccountSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamServiceAccountSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	Name      *string          `db:"name" `
	Disabled  *bool            `db:"disabled" `
	CreatedAt *time.Time       `db:"created_at" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamServiceAccountSetter) SetColumns() []string {
	vals := make([]string, 0, 7)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Name != nil {
		vals = append(vals, "name")
	}
	if s.Disabled != nil {
		vals = append(vals, "disabled")
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

func (s IamServiceAccountSetter) Overwrite(t *IamServiceAccount) {
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
	if s.Name != nil {
		t.Name = func() string {
			if s.Name == nil {
				return *new(string)
			}
			return *s.Name
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

func (s *IamServiceAccountSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamServiceAccounts.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Name != nil {
			vals[2] = psql.Arg(func() string {
				if s.Name == nil {
					return *new(string)
				}
				return *s.Name
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Disabled != nil {
			vals[3] = psql.Arg(func() bool {
				if s.Disabled == nil {
					return *new(bool)
				}
				return *s.Disabled
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

		if s.UpdatedAt != nil {
			vals[5] = psql.Arg(func() time.Time {
				if s.UpdatedAt == nil {
					return *new(time.Time)
				}
				return *s.UpdatedAt
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

func (s IamServiceAccountSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamServiceAccountSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Name != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "name")...),
			psql.Arg(s.Name),
		}})
	}

	if s.Disabled != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "disabled")...),
			psql.Arg(s.Disabled),
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

// FindIamServiceAccount retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamServiceAccount(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamServiceAccount, error) {
	if len(cols) == 0 {
		return IamServiceAccounts.Query(
			sm.Where(IamServiceAccounts.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamServiceAccounts.Query(
		sm.Where(IamServiceAccounts.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamServiceAccounts.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamServiceAccountExists checks the presence of a single record by primary key
func IamServiceAccountExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamServiceAccounts.Query(
		sm.Where(IamServiceAccounts.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamServiceAccount is retrieved from the database
func (o *IamServiceAccount) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamServiceAccounts.AfterSelectHooks.RunHooks(ctx, exec, IamServiceAccountSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamServiceAccounts.AfterInsertHooks.RunHooks(ctx, exec, IamServiceAccountSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamServiceAccounts.AfterUpdateHooks.RunHooks(ctx, exec, IamServiceAccountSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamServiceAccounts.AfterDeleteHooks.RunHooks(ctx, exec, IamServiceAccountSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamServiceAccounts.AfterMergeHooks.RunHooks(ctx, exec, IamServiceAccountSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamServiceAccount
func (o *IamServiceAccount) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamServiceAccount) pkEQ() dialect.Expression {
	return psql.Quote("iam_service_accounts", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamServiceAccount
func (o *IamServiceAccount) Update(ctx context.Context, exec bob.Executor, s *IamServiceAccountSetter) error {
	v, err := IamServiceAccounts.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamServiceAccount record with an executor
func (o *IamServiceAccount) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamServiceAccounts.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamServiceAccount using the executor
func (o *IamServiceAccount) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamServiceAccounts.Query(
		sm.Where(IamServiceAccounts.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamServiceAccountSlice is retrieved from the database
func (o IamServiceAccountSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamServiceAccounts.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamServiceAccounts.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamServiceAccounts.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamServiceAccounts.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamServiceAccounts.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamServiceAccountSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_service_accounts", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamServiceAccountSlice) copyMatchingRows(from ...*IamServiceAccount) {
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
func (o IamServiceAccountSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamServiceAccounts.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamServiceAccount:
				o.copyMatchingRows(retrieved)
			case []*IamServiceAccount:
				o.copyMatchingRows(retrieved...)
			case IamServiceAccountSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamServiceAccount or a slice of IamServiceAccount
				// then run the AfterUpdateHooks on the slice
				_, err = IamServiceAccounts.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamServiceAccountSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamServiceAccounts.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamServiceAccount:
				o.copyMatchingRows(retrieved)
			case []*IamServiceAccount:
				o.copyMatchingRows(retrieved...)
			case IamServiceAccountSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamServiceAccount or a slice of IamServiceAccount
				// then run the AfterDeleteHooks on the slice
				_, err = IamServiceAccounts.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamServiceAccountSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamServiceAccounts.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamServiceAccount:
				o.copyMatchingRows(retrieved)
			case []*IamServiceAccount:
				o.copyMatchingRows(retrieved...)
			case IamServiceAccountSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamServiceAccount or a slice of IamServiceAccount
				// then run the AfterMergeHooks on the slice
				_, err = IamServiceAccounts.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamServiceAccountSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamServiceAccountSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamServiceAccounts.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamServiceAccountSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamServiceAccounts.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamServiceAccountSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamServiceAccounts.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamServiceAccountWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Name      psql.WhereMod[Q, string]
	Disabled  psql.WhereMod[Q, bool]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamServiceAccountWhere[Q]) AliasedAs(alias string) iamServiceAccountWhere[Q] {
	return buildIamServiceAccountWhere[Q](buildIamServiceAccountColumns(alias))
}

func buildIamServiceAccountWhere[Q psql.Filterable](cols iamServiceAccountColumns) iamServiceAccountWhere[Q] {
	return iamServiceAccountWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Name:      psql.Where[Q, string](cols.Name.Expression),
		Disabled:  psql.Where[Q, bool](cols.Disabled.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
