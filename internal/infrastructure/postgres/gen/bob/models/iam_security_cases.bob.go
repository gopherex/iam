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

// IamSecurityCase is an object representing the database table.
type IamSecurityCase struct {
	ID          string          `db:"id,pk" `
	ProjectID   string          `db:"project_id" `
	Environment string          `db:"environment" `
	UserID      string          `db:"user_id" `
	Status      string          `db:"status" `
	CreatedAt   time.Time       `db:"created_at" `
	Data        json.RawMessage `db:"data" `
	PrivateData string          `db:"private_data" `
}

// IamSecurityCaseSlice is an alias for a slice of pointers to IamSecurityCase.
// This should almost always be used instead of []*IamSecurityCase.
type IamSecurityCaseSlice []*IamSecurityCase

// IamSecurityCases contains methods to work with the iam_security_cases table
var IamSecurityCases = psql.NewTablex[*IamSecurityCase, IamSecurityCaseSlice, *IamSecurityCaseSetter]("", "iam_security_cases", buildIamSecurityCaseColumns("iam_security_cases"))

// IamSecurityCasesQuery is a query on the iam_security_cases table
type IamSecurityCasesQuery = *psql.ViewQuery[*IamSecurityCase, IamSecurityCaseSlice]

func buildIamSecurityCaseColumns(tableName string) iamSecurityCaseColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "user_id", "status", "created_at", "data", "private_data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSecurityCaseColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamSecurityCaseColumn(tableName, "id"),
		ProjectID:   buildIamSecurityCaseColumn(tableName, "project_id"),
		Environment: buildIamSecurityCaseColumn(tableName, "environment"),
		UserID:      buildIamSecurityCaseColumn(tableName, "user_id"),
		Status:      buildIamSecurityCaseColumn(tableName, "status"),
		CreatedAt:   buildIamSecurityCaseColumn(tableName, "created_at"),
		Data:        buildIamSecurityCaseColumn(tableName, "data"),
		PrivateData: buildIamSecurityCaseColumn(tableName, "private_data"),
	}
}

type iamSecurityCaseColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamSecurityCaseColumn
	ProjectID   iamSecurityCaseColumn
	Environment iamSecurityCaseColumn
	UserID      iamSecurityCaseColumn
	Status      iamSecurityCaseColumn
	CreatedAt   iamSecurityCaseColumn
	Data        iamSecurityCaseColumn
	PrivateData iamSecurityCaseColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSecurityCaseColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSecurityCaseColumns) AliasedAs(tableName string) iamSecurityCaseColumns {
	return buildIamSecurityCaseColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSecurityCaseColumns) Unqualified() iamSecurityCaseColumns {
	return buildIamSecurityCaseColumns("")
}

func buildIamSecurityCaseColumn(alias, name string) iamSecurityCaseColumn {
	return iamSecurityCaseColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSecurityCaseColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSecurityCaseColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSecurityCaseColumn) ShouldOmitParens() bool {
	return true
}

// IamSecurityCaseSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSecurityCaseSetter struct {
	ID          *string          `db:"id,pk" `
	ProjectID   *string          `db:"project_id" `
	Environment *string          `db:"environment" `
	UserID      *string          `db:"user_id" `
	Status      *string          `db:"status" `
	CreatedAt   *time.Time       `db:"created_at" `
	Data        *json.RawMessage `db:"data" `
	PrivateData *string          `db:"private_data" `
}

func (s IamSecurityCaseSetter) SetColumns() []string {
	vals := make([]string, 0, 8)
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
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	if s.PrivateData != nil {
		vals = append(vals, "private_data")
	}
	return vals
}

func (s IamSecurityCaseSetter) Overwrite(t *IamSecurityCase) {
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
	if s.Status != nil {
		t.Status = func() string {
			if s.Status == nil {
				return *new(string)
			}
			return *s.Status
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
	if s.PrivateData != nil {
		t.PrivateData = func() string {
			if s.PrivateData == nil {
				return *new(string)
			}
			return *s.PrivateData
		}()
	}
}

func (s *IamSecurityCaseSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSecurityCases.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.CreatedAt != nil {
			vals[5] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
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

		if s.PrivateData != nil {
			vals[7] = psql.Arg(func() string {
				if s.PrivateData == nil {
					return *new(string)
				}
				return *s.PrivateData
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamSecurityCaseSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSecurityCaseSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Status != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "status")...),
			psql.Arg(s.Status),
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

	if s.PrivateData != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "private_data")...),
			psql.Arg(s.PrivateData),
		}})
	}

	return exprs
}

// FindIamSecurityCase retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSecurityCase(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamSecurityCase, error) {
	if len(cols) == 0 {
		return IamSecurityCases.Query(
			sm.Where(IamSecurityCases.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamSecurityCases.Query(
		sm.Where(IamSecurityCases.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamSecurityCases.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSecurityCaseExists checks the presence of a single record by primary key
func IamSecurityCaseExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamSecurityCases.Query(
		sm.Where(IamSecurityCases.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSecurityCase is retrieved from the database
func (o *IamSecurityCase) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityCases.AfterSelectHooks.RunHooks(ctx, exec, IamSecurityCaseSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityCases.AfterInsertHooks.RunHooks(ctx, exec, IamSecurityCaseSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityCases.AfterUpdateHooks.RunHooks(ctx, exec, IamSecurityCaseSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityCases.AfterDeleteHooks.RunHooks(ctx, exec, IamSecurityCaseSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityCases.AfterMergeHooks.RunHooks(ctx, exec, IamSecurityCaseSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSecurityCase
func (o *IamSecurityCase) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamSecurityCase) pkEQ() dialect.Expression {
	return psql.Quote("iam_security_cases", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSecurityCase
func (o *IamSecurityCase) Update(ctx context.Context, exec bob.Executor, s *IamSecurityCaseSetter) error {
	v, err := IamSecurityCases.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSecurityCase record with an executor
func (o *IamSecurityCase) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSecurityCases.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSecurityCase using the executor
func (o *IamSecurityCase) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSecurityCases.Query(
		sm.Where(IamSecurityCases.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSecurityCaseSlice is retrieved from the database
func (o IamSecurityCaseSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityCases.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityCases.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityCases.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityCases.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityCases.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSecurityCaseSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_security_cases", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSecurityCaseSlice) copyMatchingRows(from ...*IamSecurityCase) {
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
func (o IamSecurityCaseSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityCases.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityCase:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityCase:
				o.copyMatchingRows(retrieved...)
			case IamSecurityCaseSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityCase or a slice of IamSecurityCase
				// then run the AfterUpdateHooks on the slice
				_, err = IamSecurityCases.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSecurityCaseSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityCases.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityCase:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityCase:
				o.copyMatchingRows(retrieved...)
			case IamSecurityCaseSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityCase or a slice of IamSecurityCase
				// then run the AfterDeleteHooks on the slice
				_, err = IamSecurityCases.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSecurityCaseSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityCases.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityCase:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityCase:
				o.copyMatchingRows(retrieved...)
			case IamSecurityCaseSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityCase or a slice of IamSecurityCase
				// then run the AfterMergeHooks on the slice
				_, err = IamSecurityCases.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSecurityCaseSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSecurityCaseSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityCases.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSecurityCaseSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityCases.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSecurityCaseSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSecurityCases.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSecurityCaseWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	UserID      psql.WhereMod[Q, string]
	Status      psql.WhereMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
	PrivateData psql.WhereMod[Q, string]
}

func (iamSecurityCaseWhere[Q]) AliasedAs(alias string) iamSecurityCaseWhere[Q] {
	return buildIamSecurityCaseWhere[Q](buildIamSecurityCaseColumns(alias))
}

func buildIamSecurityCaseWhere[Q psql.Filterable](cols iamSecurityCaseColumns) iamSecurityCaseWhere[Q] {
	return iamSecurityCaseWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		UserID:      psql.Where[Q, string](cols.UserID.Expression),
		Status:      psql.Where[Q, string](cols.Status.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
		PrivateData: psql.Where[Q, string](cols.PrivateData.Expression),
	}
}
