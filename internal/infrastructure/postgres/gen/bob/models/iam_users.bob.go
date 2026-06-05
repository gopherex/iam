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

// IamUser is an object representing the database table.
type IamUser struct {
	ID           string           `db:"id,pk" `
	ProjectID    string           `db:"project_id" `
	PrimaryEmail null.Val[string] `db:"primary_email" `
	CreatedAt    time.Time        `db:"created_at" `
	UpdatedAt    time.Time        `db:"updated_at" `
	Data         json.RawMessage  `db:"data" `
}

// IamUserSlice is an alias for a slice of pointers to IamUser.
// This should almost always be used instead of []*IamUser.
type IamUserSlice []*IamUser

// IamUsers contains methods to work with the iam_users table
var IamUsers = psql.NewTablex[*IamUser, IamUserSlice, *IamUserSetter]("", "iam_users", buildIamUserColumns("iam_users"))

// IamUsersQuery is a query on the iam_users table
type IamUsersQuery = *psql.ViewQuery[*IamUser, IamUserSlice]

func buildIamUserColumns(tableName string) iamUserColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "primary_email", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamUserColumns{
		ColumnsExpr:  columnsExpr,
		tableAlias:   tableName,
		ID:           buildIamUserColumn(tableName, "id"),
		ProjectID:    buildIamUserColumn(tableName, "project_id"),
		PrimaryEmail: buildIamUserColumn(tableName, "primary_email"),
		CreatedAt:    buildIamUserColumn(tableName, "created_at"),
		UpdatedAt:    buildIamUserColumn(tableName, "updated_at"),
		Data:         buildIamUserColumn(tableName, "data"),
	}
}

type iamUserColumns struct {
	expr.ColumnsExpr
	tableAlias   string
	ID           iamUserColumn
	ProjectID    iamUserColumn
	PrimaryEmail iamUserColumn
	CreatedAt    iamUserColumn
	UpdatedAt    iamUserColumn
	Data         iamUserColumn
}

// Alias returns the current table alias for the columns set.
func (c iamUserColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamUserColumns) AliasedAs(tableName string) iamUserColumns {
	return buildIamUserColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamUserColumns) Unqualified() iamUserColumns {
	return buildIamUserColumns("")
}

func buildIamUserColumn(alias, name string) iamUserColumn {
	return iamUserColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamUserColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamUserColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamUserColumn) ShouldOmitParens() bool {
	return true
}

// IamUserSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamUserSetter struct {
	ID           *string           `db:"id,pk" `
	ProjectID    *string           `db:"project_id" `
	PrimaryEmail *null.Val[string] `db:"primary_email" `
	CreatedAt    *time.Time        `db:"created_at" `
	UpdatedAt    *time.Time        `db:"updated_at" `
	Data         *json.RawMessage  `db:"data" `
}

func (s IamUserSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.PrimaryEmail != nil {
		vals = append(vals, "primary_email")
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

func (s IamUserSetter) Overwrite(t *IamUser) {
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
	if s.PrimaryEmail != nil {
		t.PrimaryEmail = func() null.Val[string] {
			if s.PrimaryEmail == nil {
				return *new(null.Val[string])
			}
			v := s.PrimaryEmail
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

func (s *IamUserSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamUsers.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.PrimaryEmail != nil {
			vals[2] = psql.Arg(func() null.Val[string] {
				if s.PrimaryEmail == nil {
					return *new(null.Val[string])
				}
				v := s.PrimaryEmail
				return *v
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

func (s IamUserSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamUserSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.PrimaryEmail != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "primary_email")...),
			psql.Arg(s.PrimaryEmail),
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

// FindIamUser retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamUser(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamUser, error) {
	if len(cols) == 0 {
		return IamUsers.Query(
			sm.Where(IamUsers.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamUsers.Query(
		sm.Where(IamUsers.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamUsers.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamUserExists checks the presence of a single record by primary key
func IamUserExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamUsers.Query(
		sm.Where(IamUsers.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamUser is retrieved from the database
func (o *IamUser) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamUsers.AfterSelectHooks.RunHooks(ctx, exec, IamUserSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamUsers.AfterInsertHooks.RunHooks(ctx, exec, IamUserSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamUsers.AfterUpdateHooks.RunHooks(ctx, exec, IamUserSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamUsers.AfterDeleteHooks.RunHooks(ctx, exec, IamUserSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamUsers.AfterMergeHooks.RunHooks(ctx, exec, IamUserSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamUser
func (o *IamUser) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamUser) pkEQ() dialect.Expression {
	return psql.Quote("iam_users", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamUser
func (o *IamUser) Update(ctx context.Context, exec bob.Executor, s *IamUserSetter) error {
	v, err := IamUsers.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamUser record with an executor
func (o *IamUser) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamUsers.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamUser using the executor
func (o *IamUser) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamUsers.Query(
		sm.Where(IamUsers.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamUserSlice is retrieved from the database
func (o IamUserSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamUsers.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamUsers.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamUsers.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamUsers.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamUsers.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamUserSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_users", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamUserSlice) copyMatchingRows(from ...*IamUser) {
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
func (o IamUserSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamUsers.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamUser:
				o.copyMatchingRows(retrieved)
			case []*IamUser:
				o.copyMatchingRows(retrieved...)
			case IamUserSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamUser or a slice of IamUser
				// then run the AfterUpdateHooks on the slice
				_, err = IamUsers.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamUserSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamUsers.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamUser:
				o.copyMatchingRows(retrieved)
			case []*IamUser:
				o.copyMatchingRows(retrieved...)
			case IamUserSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamUser or a slice of IamUser
				// then run the AfterDeleteHooks on the slice
				_, err = IamUsers.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamUserSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamUsers.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamUser:
				o.copyMatchingRows(retrieved)
			case []*IamUser:
				o.copyMatchingRows(retrieved...)
			case IamUserSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamUser or a slice of IamUser
				// then run the AfterMergeHooks on the slice
				_, err = IamUsers.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamUserSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamUserSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamUsers.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamUserSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamUsers.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamUserSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamUsers.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamUserWhere[Q psql.Filterable] struct {
	ID           psql.WhereMod[Q, string]
	ProjectID    psql.WhereMod[Q, string]
	PrimaryEmail psql.WhereNullMod[Q, string]
	CreatedAt    psql.WhereMod[Q, time.Time]
	UpdatedAt    psql.WhereMod[Q, time.Time]
	Data         psql.WhereMod[Q, json.RawMessage]
}

func (iamUserWhere[Q]) AliasedAs(alias string) iamUserWhere[Q] {
	return buildIamUserWhere[Q](buildIamUserColumns(alias))
}

func buildIamUserWhere[Q psql.Filterable](cols iamUserColumns) iamUserWhere[Q] {
	return iamUserWhere[Q]{
		ID:           psql.Where[Q, string](cols.ID.Expression),
		ProjectID:    psql.Where[Q, string](cols.ProjectID.Expression),
		PrimaryEmail: psql.WhereNull[Q, string](cols.PrimaryEmail.Expression),
		CreatedAt:    psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt:    psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:         psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
