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

// IamAuthCode is an object representing the database table.
type IamAuthCode struct {
	ID        string           `db:"id,pk" `
	ProjectID string           `db:"project_id" `
	CodeHash  string           `db:"code_hash" `
	ClientID  null.Val[string] `db:"client_id" `
	UserID    null.Val[string] `db:"user_id" `
	ExpiresAt time.Time        `db:"expires_at" `
	Consumed  bool             `db:"consumed" `
	CreatedAt time.Time        `db:"created_at" `
	Data      json.RawMessage  `db:"data" `
}

// IamAuthCodeSlice is an alias for a slice of pointers to IamAuthCode.
// This should almost always be used instead of []*IamAuthCode.
type IamAuthCodeSlice []*IamAuthCode

// IamAuthCodes contains methods to work with the iam_auth_codes table
var IamAuthCodes = psql.NewTablex[*IamAuthCode, IamAuthCodeSlice, *IamAuthCodeSetter]("", "iam_auth_codes", buildIamAuthCodeColumns("iam_auth_codes"))

// IamAuthCodesQuery is a query on the iam_auth_codes table
type IamAuthCodesQuery = *psql.ViewQuery[*IamAuthCode, IamAuthCodeSlice]

func buildIamAuthCodeColumns(tableName string) iamAuthCodeColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "code_hash", "client_id", "user_id", "expires_at", "consumed", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamAuthCodeColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamAuthCodeColumn(tableName, "id"),
		ProjectID:   buildIamAuthCodeColumn(tableName, "project_id"),
		CodeHash:    buildIamAuthCodeColumn(tableName, "code_hash"),
		ClientID:    buildIamAuthCodeColumn(tableName, "client_id"),
		UserID:      buildIamAuthCodeColumn(tableName, "user_id"),
		ExpiresAt:   buildIamAuthCodeColumn(tableName, "expires_at"),
		Consumed:    buildIamAuthCodeColumn(tableName, "consumed"),
		CreatedAt:   buildIamAuthCodeColumn(tableName, "created_at"),
		Data:        buildIamAuthCodeColumn(tableName, "data"),
	}
}

type iamAuthCodeColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamAuthCodeColumn
	ProjectID  iamAuthCodeColumn
	CodeHash   iamAuthCodeColumn
	ClientID   iamAuthCodeColumn
	UserID     iamAuthCodeColumn
	ExpiresAt  iamAuthCodeColumn
	Consumed   iamAuthCodeColumn
	CreatedAt  iamAuthCodeColumn
	Data       iamAuthCodeColumn
}

// Alias returns the current table alias for the columns set.
func (c iamAuthCodeColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamAuthCodeColumns) AliasedAs(tableName string) iamAuthCodeColumns {
	return buildIamAuthCodeColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamAuthCodeColumns) Unqualified() iamAuthCodeColumns {
	return buildIamAuthCodeColumns("")
}

func buildIamAuthCodeColumn(alias, name string) iamAuthCodeColumn {
	return iamAuthCodeColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamAuthCodeColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamAuthCodeColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamAuthCodeColumn) ShouldOmitParens() bool {
	return true
}

// IamAuthCodeSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamAuthCodeSetter struct {
	ID        *string           `db:"id,pk" `
	ProjectID *string           `db:"project_id" `
	CodeHash  *string           `db:"code_hash" `
	ClientID  *null.Val[string] `db:"client_id" `
	UserID    *null.Val[string] `db:"user_id" `
	ExpiresAt *time.Time        `db:"expires_at" `
	Consumed  *bool             `db:"consumed" `
	CreatedAt *time.Time        `db:"created_at" `
	Data      *json.RawMessage  `db:"data" `
}

func (s IamAuthCodeSetter) SetColumns() []string {
	vals := make([]string, 0, 9)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.CodeHash != nil {
		vals = append(vals, "code_hash")
	}
	if s.ClientID != nil {
		vals = append(vals, "client_id")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
	}
	if s.Consumed != nil {
		vals = append(vals, "consumed")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamAuthCodeSetter) Overwrite(t *IamAuthCode) {
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
	if s.CodeHash != nil {
		t.CodeHash = func() string {
			if s.CodeHash == nil {
				return *new(string)
			}
			return *s.CodeHash
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
	if s.UserID != nil {
		t.UserID = func() null.Val[string] {
			if s.UserID == nil {
				return *new(null.Val[string])
			}
			v := s.UserID
			return *v
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
	if s.Consumed != nil {
		t.Consumed = func() bool {
			if s.Consumed == nil {
				return *new(bool)
			}
			return *s.Consumed
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

func (s *IamAuthCodeSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamAuthCodes.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 9)
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

		if s.CodeHash != nil {
			vals[2] = psql.Arg(func() string {
				if s.CodeHash == nil {
					return *new(string)
				}
				return *s.CodeHash
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.ClientID != nil {
			vals[3] = psql.Arg(func() null.Val[string] {
				if s.ClientID == nil {
					return *new(null.Val[string])
				}
				v := s.ClientID
				return *v
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.UserID != nil {
			vals[4] = psql.Arg(func() null.Val[string] {
				if s.UserID == nil {
					return *new(null.Val[string])
				}
				v := s.UserID
				return *v
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[5] = psql.Arg(func() time.Time {
				if s.ExpiresAt == nil {
					return *new(time.Time)
				}
				return *s.ExpiresAt
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.Consumed != nil {
			vals[6] = psql.Arg(func() bool {
				if s.Consumed == nil {
					return *new(bool)
				}
				return *s.Consumed
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[7] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[8] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[8] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamAuthCodeSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamAuthCodeSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 9)

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

	if s.CodeHash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "code_hash")...),
			psql.Arg(s.CodeHash),
		}})
	}

	if s.ClientID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "client_id")...),
			psql.Arg(s.ClientID),
		}})
	}

	if s.UserID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_id")...),
			psql.Arg(s.UserID),
		}})
	}

	if s.ExpiresAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "expires_at")...),
			psql.Arg(s.ExpiresAt),
		}})
	}

	if s.Consumed != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "consumed")...),
			psql.Arg(s.Consumed),
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

// FindIamAuthCode retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamAuthCode(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamAuthCode, error) {
	if len(cols) == 0 {
		return IamAuthCodes.Query(
			sm.Where(IamAuthCodes.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamAuthCodes.Query(
		sm.Where(IamAuthCodes.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamAuthCodes.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamAuthCodeExists checks the presence of a single record by primary key
func IamAuthCodeExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamAuthCodes.Query(
		sm.Where(IamAuthCodes.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamAuthCode is retrieved from the database
func (o *IamAuthCode) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAuthCodes.AfterSelectHooks.RunHooks(ctx, exec, IamAuthCodeSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamAuthCodes.AfterInsertHooks.RunHooks(ctx, exec, IamAuthCodeSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamAuthCodes.AfterUpdateHooks.RunHooks(ctx, exec, IamAuthCodeSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamAuthCodes.AfterDeleteHooks.RunHooks(ctx, exec, IamAuthCodeSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamAuthCodes.AfterMergeHooks.RunHooks(ctx, exec, IamAuthCodeSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamAuthCode
func (o *IamAuthCode) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamAuthCode) pkEQ() dialect.Expression {
	return psql.Quote("iam_auth_codes", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamAuthCode
func (o *IamAuthCode) Update(ctx context.Context, exec bob.Executor, s *IamAuthCodeSetter) error {
	v, err := IamAuthCodes.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamAuthCode record with an executor
func (o *IamAuthCode) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamAuthCodes.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamAuthCode using the executor
func (o *IamAuthCode) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamAuthCodes.Query(
		sm.Where(IamAuthCodes.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamAuthCodeSlice is retrieved from the database
func (o IamAuthCodeSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAuthCodes.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamAuthCodes.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamAuthCodes.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamAuthCodes.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamAuthCodes.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamAuthCodeSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_auth_codes", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamAuthCodeSlice) copyMatchingRows(from ...*IamAuthCode) {
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
func (o IamAuthCodeSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAuthCodes.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAuthCode:
				o.copyMatchingRows(retrieved)
			case []*IamAuthCode:
				o.copyMatchingRows(retrieved...)
			case IamAuthCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAuthCode or a slice of IamAuthCode
				// then run the AfterUpdateHooks on the slice
				_, err = IamAuthCodes.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamAuthCodeSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAuthCodes.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAuthCode:
				o.copyMatchingRows(retrieved)
			case []*IamAuthCode:
				o.copyMatchingRows(retrieved...)
			case IamAuthCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAuthCode or a slice of IamAuthCode
				// then run the AfterDeleteHooks on the slice
				_, err = IamAuthCodes.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamAuthCodeSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAuthCodes.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAuthCode:
				o.copyMatchingRows(retrieved)
			case []*IamAuthCode:
				o.copyMatchingRows(retrieved...)
			case IamAuthCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAuthCode or a slice of IamAuthCode
				// then run the AfterMergeHooks on the slice
				_, err = IamAuthCodes.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamAuthCodeSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamAuthCodeSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAuthCodes.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamAuthCodeSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAuthCodes.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamAuthCodeSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamAuthCodes.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamAuthCodeWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	CodeHash  psql.WhereMod[Q, string]
	ClientID  psql.WhereNullMod[Q, string]
	UserID    psql.WhereNullMod[Q, string]
	ExpiresAt psql.WhereMod[Q, time.Time]
	Consumed  psql.WhereMod[Q, bool]
	CreatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamAuthCodeWhere[Q]) AliasedAs(alias string) iamAuthCodeWhere[Q] {
	return buildIamAuthCodeWhere[Q](buildIamAuthCodeColumns(alias))
}

func buildIamAuthCodeWhere[Q psql.Filterable](cols iamAuthCodeColumns) iamAuthCodeWhere[Q] {
	return iamAuthCodeWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		CodeHash:  psql.Where[Q, string](cols.CodeHash.Expression),
		ClientID:  psql.WhereNull[Q, string](cols.ClientID.Expression),
		UserID:    psql.WhereNull[Q, string](cols.UserID.Expression),
		ExpiresAt: psql.Where[Q, time.Time](cols.ExpiresAt.Expression),
		Consumed:  psql.Where[Q, bool](cols.Consumed.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
