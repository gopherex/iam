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

// IamRecoveryCode is an object representing the database table.
type IamRecoveryCode struct {
	ID          string    `db:"id,pk" `
	ProjectID   string    `db:"project_id" `
	Environment string    `db:"environment" `
	UserID      string    `db:"user_id" `
	Hash        string    `db:"hash" `
	Used        bool      `db:"used" `
	CreatedAt   time.Time `db:"created_at" `
}

// IamRecoveryCodeSlice is an alias for a slice of pointers to IamRecoveryCode.
// This should almost always be used instead of []*IamRecoveryCode.
type IamRecoveryCodeSlice []*IamRecoveryCode

// IamRecoveryCodes contains methods to work with the iam_recovery_codes table
var IamRecoveryCodes = psql.NewTablex[*IamRecoveryCode, IamRecoveryCodeSlice, *IamRecoveryCodeSetter]("", "iam_recovery_codes", buildIamRecoveryCodeColumns("iam_recovery_codes"))

// IamRecoveryCodesQuery is a query on the iam_recovery_codes table
type IamRecoveryCodesQuery = *psql.ViewQuery[*IamRecoveryCode, IamRecoveryCodeSlice]

func buildIamRecoveryCodeColumns(tableName string) iamRecoveryCodeColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "user_id", "hash", "used", "created_at",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamRecoveryCodeColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamRecoveryCodeColumn(tableName, "id"),
		ProjectID:   buildIamRecoveryCodeColumn(tableName, "project_id"),
		Environment: buildIamRecoveryCodeColumn(tableName, "environment"),
		UserID:      buildIamRecoveryCodeColumn(tableName, "user_id"),
		Hash:        buildIamRecoveryCodeColumn(tableName, "hash"),
		Used:        buildIamRecoveryCodeColumn(tableName, "used"),
		CreatedAt:   buildIamRecoveryCodeColumn(tableName, "created_at"),
	}
}

type iamRecoveryCodeColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamRecoveryCodeColumn
	ProjectID   iamRecoveryCodeColumn
	Environment iamRecoveryCodeColumn
	UserID      iamRecoveryCodeColumn
	Hash        iamRecoveryCodeColumn
	Used        iamRecoveryCodeColumn
	CreatedAt   iamRecoveryCodeColumn
}

// Alias returns the current table alias for the columns set.
func (c iamRecoveryCodeColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamRecoveryCodeColumns) AliasedAs(tableName string) iamRecoveryCodeColumns {
	return buildIamRecoveryCodeColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamRecoveryCodeColumns) Unqualified() iamRecoveryCodeColumns {
	return buildIamRecoveryCodeColumns("")
}

func buildIamRecoveryCodeColumn(alias, name string) iamRecoveryCodeColumn {
	return iamRecoveryCodeColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamRecoveryCodeColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamRecoveryCodeColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamRecoveryCodeColumn) ShouldOmitParens() bool {
	return true
}

// IamRecoveryCodeSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamRecoveryCodeSetter struct {
	ID          *string    `db:"id,pk" `
	ProjectID   *string    `db:"project_id" `
	Environment *string    `db:"environment" `
	UserID      *string    `db:"user_id" `
	Hash        *string    `db:"hash" `
	Used        *bool      `db:"used" `
	CreatedAt   *time.Time `db:"created_at" `
}

func (s IamRecoveryCodeSetter) SetColumns() []string {
	vals := make([]string, 0, 7)
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
	if s.Hash != nil {
		vals = append(vals, "hash")
	}
	if s.Used != nil {
		vals = append(vals, "used")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	return vals
}

func (s IamRecoveryCodeSetter) Overwrite(t *IamRecoveryCode) {
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
	if s.Hash != nil {
		t.Hash = func() string {
			if s.Hash == nil {
				return *new(string)
			}
			return *s.Hash
		}()
	}
	if s.Used != nil {
		t.Used = func() bool {
			if s.Used == nil {
				return *new(bool)
			}
			return *s.Used
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
}

func (s *IamRecoveryCodeSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamRecoveryCodes.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Hash != nil {
			vals[4] = psql.Arg(func() string {
				if s.Hash == nil {
					return *new(string)
				}
				return *s.Hash
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.Used != nil {
			vals[5] = psql.Arg(func() bool {
				if s.Used == nil {
					return *new(bool)
				}
				return *s.Used
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

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamRecoveryCodeSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamRecoveryCodeSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Hash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "hash")...),
			psql.Arg(s.Hash),
		}})
	}

	if s.Used != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "used")...),
			psql.Arg(s.Used),
		}})
	}

	if s.CreatedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "created_at")...),
			psql.Arg(s.CreatedAt),
		}})
	}

	return exprs
}

// FindIamRecoveryCode retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamRecoveryCode(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamRecoveryCode, error) {
	if len(cols) == 0 {
		return IamRecoveryCodes.Query(
			sm.Where(IamRecoveryCodes.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamRecoveryCodes.Query(
		sm.Where(IamRecoveryCodes.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamRecoveryCodes.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamRecoveryCodeExists checks the presence of a single record by primary key
func IamRecoveryCodeExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamRecoveryCodes.Query(
		sm.Where(IamRecoveryCodes.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamRecoveryCode is retrieved from the database
func (o *IamRecoveryCode) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamRecoveryCodes.AfterSelectHooks.RunHooks(ctx, exec, IamRecoveryCodeSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamRecoveryCodes.AfterInsertHooks.RunHooks(ctx, exec, IamRecoveryCodeSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamRecoveryCodes.AfterUpdateHooks.RunHooks(ctx, exec, IamRecoveryCodeSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamRecoveryCodes.AfterDeleteHooks.RunHooks(ctx, exec, IamRecoveryCodeSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamRecoveryCodes.AfterMergeHooks.RunHooks(ctx, exec, IamRecoveryCodeSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamRecoveryCode
func (o *IamRecoveryCode) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamRecoveryCode) pkEQ() dialect.Expression {
	return psql.Quote("iam_recovery_codes", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamRecoveryCode
func (o *IamRecoveryCode) Update(ctx context.Context, exec bob.Executor, s *IamRecoveryCodeSetter) error {
	v, err := IamRecoveryCodes.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamRecoveryCode record with an executor
func (o *IamRecoveryCode) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamRecoveryCodes.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamRecoveryCode using the executor
func (o *IamRecoveryCode) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamRecoveryCodes.Query(
		sm.Where(IamRecoveryCodes.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamRecoveryCodeSlice is retrieved from the database
func (o IamRecoveryCodeSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamRecoveryCodes.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamRecoveryCodes.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamRecoveryCodes.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamRecoveryCodes.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamRecoveryCodes.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamRecoveryCodeSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_recovery_codes", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamRecoveryCodeSlice) copyMatchingRows(from ...*IamRecoveryCode) {
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
func (o IamRecoveryCodeSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRecoveryCodes.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRecoveryCode:
				o.copyMatchingRows(retrieved)
			case []*IamRecoveryCode:
				o.copyMatchingRows(retrieved...)
			case IamRecoveryCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRecoveryCode or a slice of IamRecoveryCode
				// then run the AfterUpdateHooks on the slice
				_, err = IamRecoveryCodes.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamRecoveryCodeSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRecoveryCodes.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRecoveryCode:
				o.copyMatchingRows(retrieved)
			case []*IamRecoveryCode:
				o.copyMatchingRows(retrieved...)
			case IamRecoveryCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRecoveryCode or a slice of IamRecoveryCode
				// then run the AfterDeleteHooks on the slice
				_, err = IamRecoveryCodes.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamRecoveryCodeSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRecoveryCodes.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRecoveryCode:
				o.copyMatchingRows(retrieved)
			case []*IamRecoveryCode:
				o.copyMatchingRows(retrieved...)
			case IamRecoveryCodeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRecoveryCode or a slice of IamRecoveryCode
				// then run the AfterMergeHooks on the slice
				_, err = IamRecoveryCodes.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamRecoveryCodeSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamRecoveryCodeSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamRecoveryCodes.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamRecoveryCodeSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamRecoveryCodes.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamRecoveryCodeSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamRecoveryCodes.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamRecoveryCodeWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	UserID      psql.WhereMod[Q, string]
	Hash        psql.WhereMod[Q, string]
	Used        psql.WhereMod[Q, bool]
	CreatedAt   psql.WhereMod[Q, time.Time]
}

func (iamRecoveryCodeWhere[Q]) AliasedAs(alias string) iamRecoveryCodeWhere[Q] {
	return buildIamRecoveryCodeWhere[Q](buildIamRecoveryCodeColumns(alias))
}

func buildIamRecoveryCodeWhere[Q psql.Filterable](cols iamRecoveryCodeColumns) iamRecoveryCodeWhere[Q] {
	return iamRecoveryCodeWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		UserID:      psql.Where[Q, string](cols.UserID.Expression),
		Hash:        psql.Where[Q, string](cols.Hash.Expression),
		Used:        psql.Where[Q, bool](cols.Used.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
	}
}
