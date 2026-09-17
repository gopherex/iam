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

// IamAccountDeletion is an object representing the database table.
type IamAccountDeletion struct {
	ProjectID   string          `db:"project_id,pk" `
	Environment string          `db:"environment,pk" `
	UserID      string          `db:"user_id,pk" `
	RequestID   string          `db:"request_id" `
	Status      string          `db:"status" `
	DeleteAt    time.Time       `db:"delete_at" `
	Data        json.RawMessage `db:"data" `
}

// IamAccountDeletionSlice is an alias for a slice of pointers to IamAccountDeletion.
// This should almost always be used instead of []*IamAccountDeletion.
type IamAccountDeletionSlice []*IamAccountDeletion

// IamAccountDeletions contains methods to work with the iam_account_deletions table
var IamAccountDeletions = psql.NewTablex[*IamAccountDeletion, IamAccountDeletionSlice, *IamAccountDeletionSetter]("", "iam_account_deletions", buildIamAccountDeletionColumns("iam_account_deletions"))

// IamAccountDeletionsQuery is a query on the iam_account_deletions table
type IamAccountDeletionsQuery = *psql.ViewQuery[*IamAccountDeletion, IamAccountDeletionSlice]

func buildIamAccountDeletionColumns(tableName string) iamAccountDeletionColumns {
	columnsExpr := expr.NewColumnsExpr(
		"project_id", "environment", "user_id", "request_id", "status", "delete_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamAccountDeletionColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ProjectID:   buildIamAccountDeletionColumn(tableName, "project_id"),
		Environment: buildIamAccountDeletionColumn(tableName, "environment"),
		UserID:      buildIamAccountDeletionColumn(tableName, "user_id"),
		RequestID:   buildIamAccountDeletionColumn(tableName, "request_id"),
		Status:      buildIamAccountDeletionColumn(tableName, "status"),
		DeleteAt:    buildIamAccountDeletionColumn(tableName, "delete_at"),
		Data:        buildIamAccountDeletionColumn(tableName, "data"),
	}
}

type iamAccountDeletionColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ProjectID   iamAccountDeletionColumn
	Environment iamAccountDeletionColumn
	UserID      iamAccountDeletionColumn
	RequestID   iamAccountDeletionColumn
	Status      iamAccountDeletionColumn
	DeleteAt    iamAccountDeletionColumn
	Data        iamAccountDeletionColumn
}

// Alias returns the current table alias for the columns set.
func (c iamAccountDeletionColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamAccountDeletionColumns) AliasedAs(tableName string) iamAccountDeletionColumns {
	return buildIamAccountDeletionColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamAccountDeletionColumns) Unqualified() iamAccountDeletionColumns {
	return buildIamAccountDeletionColumns("")
}

func buildIamAccountDeletionColumn(alias, name string) iamAccountDeletionColumn {
	return iamAccountDeletionColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamAccountDeletionColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamAccountDeletionColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamAccountDeletionColumn) ShouldOmitParens() bool {
	return true
}

// IamAccountDeletionSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamAccountDeletionSetter struct {
	ProjectID   *string          `db:"project_id,pk" `
	Environment *string          `db:"environment,pk" `
	UserID      *string          `db:"user_id,pk" `
	RequestID   *string          `db:"request_id" `
	Status      *string          `db:"status" `
	DeleteAt    *time.Time       `db:"delete_at" `
	Data        *json.RawMessage `db:"data" `
}

func (s IamAccountDeletionSetter) SetColumns() []string {
	vals := make([]string, 0, 7)
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
	}
	if s.RequestID != nil {
		vals = append(vals, "request_id")
	}
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.DeleteAt != nil {
		vals = append(vals, "delete_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamAccountDeletionSetter) Overwrite(t *IamAccountDeletion) {
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
	if s.RequestID != nil {
		t.RequestID = func() string {
			if s.RequestID == nil {
				return *new(string)
			}
			return *s.RequestID
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
	if s.DeleteAt != nil {
		t.DeleteAt = func() time.Time {
			if s.DeleteAt == nil {
				return *new(time.Time)
			}
			return *s.DeleteAt
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

func (s *IamAccountDeletionSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamAccountDeletions.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 7)
		if s.ProjectID != nil {
			vals[0] = psql.Arg(func() string {
				if s.ProjectID == nil {
					return *new(string)
				}
				return *s.ProjectID
			}())
		} else {
			vals[0] = psql.Raw("DEFAULT")
		}

		if s.Environment != nil {
			vals[1] = psql.Arg(func() string {
				if s.Environment == nil {
					return *new(string)
				}
				return *s.Environment
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

		if s.RequestID != nil {
			vals[3] = psql.Arg(func() string {
				if s.RequestID == nil {
					return *new(string)
				}
				return *s.RequestID
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

		if s.DeleteAt != nil {
			vals[5] = psql.Arg(func() time.Time {
				if s.DeleteAt == nil {
					return *new(time.Time)
				}
				return *s.DeleteAt
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

func (s IamAccountDeletionSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamAccountDeletionSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 7)

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

	if s.RequestID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "request_id")...),
			psql.Arg(s.RequestID),
		}})
	}

	if s.Status != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "status")...),
			psql.Arg(s.Status),
		}})
	}

	if s.DeleteAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "delete_at")...),
			psql.Arg(s.DeleteAt),
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

// FindIamAccountDeletion retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamAccountDeletion(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, UserIDPK string, cols ...string) (*IamAccountDeletion, error) {
	if len(cols) == 0 {
		return IamAccountDeletions.Query(
			sm.Where(IamAccountDeletions.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
			sm.Where(IamAccountDeletions.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
			sm.Where(IamAccountDeletions.Columns.UserID.EQ(psql.Arg(UserIDPK))),
		).One(ctx, exec)
	}

	return IamAccountDeletions.Query(
		sm.Where(IamAccountDeletions.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamAccountDeletions.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamAccountDeletions.Columns.UserID.EQ(psql.Arg(UserIDPK))),
		sm.Columns(IamAccountDeletions.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamAccountDeletionExists checks the presence of a single record by primary key
func IamAccountDeletionExists(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, UserIDPK string) (bool, error) {
	return IamAccountDeletions.Query(
		sm.Where(IamAccountDeletions.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamAccountDeletions.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamAccountDeletions.Columns.UserID.EQ(psql.Arg(UserIDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamAccountDeletion is retrieved from the database
func (o *IamAccountDeletion) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAccountDeletions.AfterSelectHooks.RunHooks(ctx, exec, IamAccountDeletionSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamAccountDeletions.AfterInsertHooks.RunHooks(ctx, exec, IamAccountDeletionSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamAccountDeletions.AfterUpdateHooks.RunHooks(ctx, exec, IamAccountDeletionSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamAccountDeletions.AfterDeleteHooks.RunHooks(ctx, exec, IamAccountDeletionSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamAccountDeletions.AfterMergeHooks.RunHooks(ctx, exec, IamAccountDeletionSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamAccountDeletion
func (o *IamAccountDeletion) primaryKeyVals() bob.Expression {
	return psql.ArgGroup(
		o.ProjectID,
		o.Environment,
		o.UserID,
	)
}

func (o *IamAccountDeletion) pkEQ() dialect.Expression {
	return psql.Group(psql.Quote("iam_account_deletions", "project_id"), psql.Quote("iam_account_deletions", "environment"), psql.Quote("iam_account_deletions", "user_id")).EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamAccountDeletion
func (o *IamAccountDeletion) Update(ctx context.Context, exec bob.Executor, s *IamAccountDeletionSetter) error {
	v, err := IamAccountDeletions.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamAccountDeletion record with an executor
func (o *IamAccountDeletion) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamAccountDeletions.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamAccountDeletion using the executor
func (o *IamAccountDeletion) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamAccountDeletions.Query(
		sm.Where(IamAccountDeletions.Columns.ProjectID.EQ(psql.Arg(o.ProjectID))),
		sm.Where(IamAccountDeletions.Columns.Environment.EQ(psql.Arg(o.Environment))),
		sm.Where(IamAccountDeletions.Columns.UserID.EQ(psql.Arg(o.UserID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamAccountDeletionSlice is retrieved from the database
func (o IamAccountDeletionSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAccountDeletions.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamAccountDeletions.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamAccountDeletions.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamAccountDeletions.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamAccountDeletions.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamAccountDeletionSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Group(psql.Quote("iam_account_deletions", "project_id"), psql.Quote("iam_account_deletions", "environment"), psql.Quote("iam_account_deletions", "user_id")).In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamAccountDeletionSlice) copyMatchingRows(from ...*IamAccountDeletion) {
	for i, old := range o {
		for _, new := range from {
			if new.ProjectID != old.ProjectID {
				continue
			}
			if new.Environment != old.Environment {
				continue
			}
			if new.UserID != old.UserID {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamAccountDeletionSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAccountDeletions.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAccountDeletion:
				o.copyMatchingRows(retrieved)
			case []*IamAccountDeletion:
				o.copyMatchingRows(retrieved...)
			case IamAccountDeletionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAccountDeletion or a slice of IamAccountDeletion
				// then run the AfterUpdateHooks on the slice
				_, err = IamAccountDeletions.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamAccountDeletionSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAccountDeletions.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAccountDeletion:
				o.copyMatchingRows(retrieved)
			case []*IamAccountDeletion:
				o.copyMatchingRows(retrieved...)
			case IamAccountDeletionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAccountDeletion or a slice of IamAccountDeletion
				// then run the AfterDeleteHooks on the slice
				_, err = IamAccountDeletions.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamAccountDeletionSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAccountDeletions.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAccountDeletion:
				o.copyMatchingRows(retrieved)
			case []*IamAccountDeletion:
				o.copyMatchingRows(retrieved...)
			case IamAccountDeletionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAccountDeletion or a slice of IamAccountDeletion
				// then run the AfterMergeHooks on the slice
				_, err = IamAccountDeletions.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamAccountDeletionSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamAccountDeletionSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAccountDeletions.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamAccountDeletionSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAccountDeletions.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamAccountDeletionSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamAccountDeletions.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamAccountDeletionWhere[Q psql.Filterable] struct {
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	UserID      psql.WhereMod[Q, string]
	RequestID   psql.WhereMod[Q, string]
	Status      psql.WhereMod[Q, string]
	DeleteAt    psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamAccountDeletionWhere[Q]) AliasedAs(alias string) iamAccountDeletionWhere[Q] {
	return buildIamAccountDeletionWhere[Q](buildIamAccountDeletionColumns(alias))
}

func buildIamAccountDeletionWhere[Q psql.Filterable](cols iamAccountDeletionColumns) iamAccountDeletionWhere[Q] {
	return iamAccountDeletionWhere[Q]{
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		UserID:      psql.Where[Q, string](cols.UserID.Expression),
		RequestID:   psql.Where[Q, string](cols.RequestID.Expression),
		Status:      psql.Where[Q, string](cols.Status.Expression),
		DeleteAt:    psql.Where[Q, time.Time](cols.DeleteAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
