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

// IamDeviceCode is an object representing the database table.
type IamDeviceCode struct {
	ID         string           `db:"id,pk" `
	ProjectID  string           `db:"project_id" `
	DeviceCode string           `db:"device_code" `
	UserCode   string           `db:"user_code" `
	Status     string           `db:"status" `
	UserID     null.Val[string] `db:"user_id" `
	ExpiresAt  time.Time        `db:"expires_at" `
	CreatedAt  time.Time        `db:"created_at" `
	Data       json.RawMessage  `db:"data" `
}

// IamDeviceCodeSlice is an alias for a slice of pointers to IamDeviceCode.
// This should almost always be used instead of []*IamDeviceCode.
type IamDeviceCodeSlice []*IamDeviceCode

// IamDeviceCodes contains methods to work with the iam_device_codes table
var IamDeviceCodes = psql.NewTablex[*IamDeviceCode, IamDeviceCodeSlice, *IamDeviceCodeSetter]("", "iam_device_codes", buildIamDeviceCodeColumns("iam_device_codes"))

// IamDeviceCodesQuery is a query on the iam_device_codes table
type IamDeviceCodesQuery = *psql.ViewQuery[*IamDeviceCode, IamDeviceCodeSlice]

func buildIamDeviceCodeColumns(tableName string) iamDeviceCodeColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "device_code", "user_code", "status", "user_id", "expires_at", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamDeviceCodeColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamDeviceCodeColumn(tableName, "id"),
		ProjectID:   buildIamDeviceCodeColumn(tableName, "project_id"),
		DeviceCode:  buildIamDeviceCodeColumn(tableName, "device_code"),
		UserCode:    buildIamDeviceCodeColumn(tableName, "user_code"),
		Status:      buildIamDeviceCodeColumn(tableName, "status"),
		UserID:      buildIamDeviceCodeColumn(tableName, "user_id"),
		ExpiresAt:   buildIamDeviceCodeColumn(tableName, "expires_at"),
		CreatedAt:   buildIamDeviceCodeColumn(tableName, "created_at"),
		Data:        buildIamDeviceCodeColumn(tableName, "data"),
	}
}

type iamDeviceCodeColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamDeviceCodeColumn
	ProjectID  iamDeviceCodeColumn
	DeviceCode iamDeviceCodeColumn
	UserCode   iamDeviceCodeColumn
	Status     iamDeviceCodeColumn
	UserID     iamDeviceCodeColumn
	ExpiresAt  iamDeviceCodeColumn
	CreatedAt  iamDeviceCodeColumn
	Data       iamDeviceCodeColumn
}

// Alias returns the current table alias for the columns set.
func (c iamDeviceCodeColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamDeviceCodeColumns) AliasedAs(tableName string) iamDeviceCodeColumns {
	return buildIamDeviceCodeColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamDeviceCodeColumns) Unqualified() iamDeviceCodeColumns {
	return buildIamDeviceCodeColumns("")
}

func buildIamDeviceCodeColumn(alias, name string) iamDeviceCodeColumn {
	return iamDeviceCodeColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamDeviceCodeColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamDeviceCodeColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamDeviceCodeColumn) ShouldOmitParens() bool {
	return true
}

// IamDeviceCodeSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamDeviceCodeSetter struct {
	ID         *string           `db:"id,pk" `
	ProjectID  *string           `db:"project_id" `
	DeviceCode *string           `db:"device_code" `
	UserCode   *string           `db:"user_code" `
	Status     *string           `db:"status" `
	UserID     *null.Val[string] `db:"user_id" `
	ExpiresAt  *time.Time        `db:"expires_at" `
	CreatedAt  *time.Time        `db:"created_at" `
	Data       *json.RawMessage  `db:"data" `
}

func (s IamDeviceCodeSetter) SetColumns() []string {
	vals := make([]string, 0, 9)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.DeviceCode != nil {
		vals = append(vals, "device_code")
	}
	if s.UserCode != nil {
		vals = append(vals, "user_code")
	}
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
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

func (s IamDeviceCodeSetter) Overwrite(t *IamDeviceCode) {
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
	if s.DeviceCode != nil {
		t.DeviceCode = func() string {
			if s.DeviceCode == nil {
				return *new(string)
			}
			return *s.DeviceCode
		}()
	}
	if s.UserCode != nil {
		t.UserCode = func() string {
			if s.UserCode == nil {
				return *new(string)
			}
			return *s.UserCode
		}()
	}
	if s.Status != nil {
		t.Status = func() string {
			if s.Status == nil {
				return *new(string)
			}
			return *s.Status
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

func (s *IamDeviceCodeSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamDeviceCodes.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.DeviceCode != nil {
			vals[2] = psql.Arg(func() string {
				if s.DeviceCode == nil {
					return *new(string)
				}
				return *s.DeviceCode
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.UserCode != nil {
			vals[3] = psql.Arg(func() string {
				if s.UserCode == nil {
					return *new(string)
				}
				return *s.UserCode
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.Status != nil {
			vals[4] = psql.Arg(func() string {
				if s.Status == nil {
					return *new(string)
				}
				return *s.Status
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.UserID != nil {
			vals[5] = psql.Arg(func() null.Val[string] {
				if s.UserID == nil {
					return *new(null.Val[string])
				}
				v := s.UserID
				return *v
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[6] = psql.Arg(func() time.Time {
				if s.ExpiresAt == nil {
					return *new(time.Time)
				}
				return *s.ExpiresAt
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

func (s IamDeviceCodeSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamDeviceCodeSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.DeviceCode != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "device_code")...),
			psql.Arg(s.DeviceCode),
		}})
	}

	if s.UserCode != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_code")...),
			psql.Arg(s.UserCode),
		}})
	}

	if s.Status != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "status")...),
			psql.Arg(s.Status),
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

// FindIamDeviceCode retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamDeviceCode(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamDeviceCode, error) {
	if len(cols) == 0 {
		return IamDeviceCodes.Query(
			sm.Where(IamDeviceCodes.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamDeviceCodes.Query(
		sm.Where(IamDeviceCodes.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamDeviceCodes.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamDeviceCodeExists checks the presence of a single record by primary key
func IamDeviceCodeExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamDeviceCodes.Query(
		sm.Where(IamDeviceCodes.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamDeviceCode is retrieved from the database
func (o *IamDeviceCode) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamDeviceCodes.AfterSelectHooks.RunHooks(ctx, exec, IamDeviceCodeSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamDeviceCodes.AfterInsertHooks.RunHooks(ctx, exec, IamDeviceCodeSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamDeviceCodes.AfterUpdateHooks.RunHooks(ctx, exec, IamDeviceCodeSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamDeviceCodes.AfterDeleteHooks.RunHooks(ctx, exec, IamDeviceCodeSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamDeviceCodes.AfterMergeHooks.RunHooks(ctx, exec, IamDeviceCodeSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamDeviceCode
func (o *IamDeviceCode) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamDeviceCode) pkEQ() dialect.Expression {
	return psql.Quote("iam_device_codes", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamDeviceCode
func (o *IamDeviceCode) Update(ctx context.Context, exec bob.Executor, s *IamDeviceCodeSetter) error {
	v, err := IamDeviceCodes.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamDeviceCode record with an executor
func (o *IamDeviceCode) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamDeviceCodes.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamDeviceCode using the executor
func (o *IamDeviceCode) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamDeviceCodes.Query(
		sm.Where(IamDeviceCodes.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamDeviceCodeSlice is retrieved from the database
func (o IamDeviceCodeSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamDeviceCodes.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamDeviceCodes.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamDeviceCodes.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamDeviceCodes.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamDeviceCodes.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamDeviceCodeSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_device_codes", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamDeviceCodeSlice) copyMatchingRows(from ...*IamDeviceCode) {
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
func (o IamDeviceCodeSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamDeviceCodes.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamDeviceCode:
				o.copyMatchingRows(retrieved)
			case []*IamDeviceCode:
				o.copyMatchingRows(retrieved...)
			case IamDeviceCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamDeviceCode or a slice of IamDeviceCode
				// then run the AfterUpdateHooks on the slice
				_, err = IamDeviceCodes.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamDeviceCodeSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamDeviceCodes.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamDeviceCode:
				o.copyMatchingRows(retrieved)
			case []*IamDeviceCode:
				o.copyMatchingRows(retrieved...)
			case IamDeviceCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamDeviceCode or a slice of IamDeviceCode
				// then run the AfterDeleteHooks on the slice
				_, err = IamDeviceCodes.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamDeviceCodeSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamDeviceCodes.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamDeviceCode:
				o.copyMatchingRows(retrieved)
			case []*IamDeviceCode:
				o.copyMatchingRows(retrieved...)
			case IamDeviceCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamDeviceCode or a slice of IamDeviceCode
				// then run the AfterMergeHooks on the slice
				_, err = IamDeviceCodes.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamDeviceCodeSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamDeviceCodeSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamDeviceCodes.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamDeviceCodeSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamDeviceCodes.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamDeviceCodeSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamDeviceCodes.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamDeviceCodeWhere[Q psql.Filterable] struct {
	ID         psql.WhereMod[Q, string]
	ProjectID  psql.WhereMod[Q, string]
	DeviceCode psql.WhereMod[Q, string]
	UserCode   psql.WhereMod[Q, string]
	Status     psql.WhereMod[Q, string]
	UserID     psql.WhereNullMod[Q, string]
	ExpiresAt  psql.WhereMod[Q, time.Time]
	CreatedAt  psql.WhereMod[Q, time.Time]
	Data       psql.WhereMod[Q, json.RawMessage]
}

func (iamDeviceCodeWhere[Q]) AliasedAs(alias string) iamDeviceCodeWhere[Q] {
	return buildIamDeviceCodeWhere[Q](buildIamDeviceCodeColumns(alias))
}

func buildIamDeviceCodeWhere[Q psql.Filterable](cols iamDeviceCodeColumns) iamDeviceCodeWhere[Q] {
	return iamDeviceCodeWhere[Q]{
		ID:         psql.Where[Q, string](cols.ID.Expression),
		ProjectID:  psql.Where[Q, string](cols.ProjectID.Expression),
		DeviceCode: psql.Where[Q, string](cols.DeviceCode.Expression),
		UserCode:   psql.Where[Q, string](cols.UserCode.Expression),
		Status:     psql.Where[Q, string](cols.Status.Expression),
		UserID:     psql.WhereNull[Q, string](cols.UserID.Expression),
		ExpiresAt:  psql.Where[Q, time.Time](cols.ExpiresAt.Expression),
		CreatedAt:  psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:       psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
