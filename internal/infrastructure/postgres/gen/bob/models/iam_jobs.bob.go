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

// IamJob is an object representing the database table.
type IamJob struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	Type      string          `db:"type" `
	Status    string          `db:"status" `
	CreatedAt time.Time       `db:"created_at" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamJobSlice is an alias for a slice of pointers to IamJob.
// This should almost always be used instead of []*IamJob.
type IamJobSlice []*IamJob

// IamJobs contains methods to work with the iam_jobs table
var IamJobs = psql.NewTablex[*IamJob, IamJobSlice, *IamJobSetter]("", "iam_jobs", buildIamJobColumns("iam_jobs"))

// IamJobsQuery is a query on the iam_jobs table
type IamJobsQuery = *psql.ViewQuery[*IamJob, IamJobSlice]

func buildIamJobColumns(tableName string) iamJobColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "type", "status", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamJobColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamJobColumn(tableName, "id"),
		ProjectID:   buildIamJobColumn(tableName, "project_id"),
		Type:        buildIamJobColumn(tableName, "type"),
		Status:      buildIamJobColumn(tableName, "status"),
		CreatedAt:   buildIamJobColumn(tableName, "created_at"),
		UpdatedAt:   buildIamJobColumn(tableName, "updated_at"),
		Data:        buildIamJobColumn(tableName, "data"),
	}
}

type iamJobColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamJobColumn
	ProjectID  iamJobColumn
	Type       iamJobColumn
	Status     iamJobColumn
	CreatedAt  iamJobColumn
	UpdatedAt  iamJobColumn
	Data       iamJobColumn
}

// Alias returns the current table alias for the columns set.
func (c iamJobColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamJobColumns) AliasedAs(tableName string) iamJobColumns {
	return buildIamJobColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamJobColumns) Unqualified() iamJobColumns {
	return buildIamJobColumns("")
}

func buildIamJobColumn(alias, name string) iamJobColumn {
	return iamJobColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamJobColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamJobColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamJobColumn) ShouldOmitParens() bool {
	return true
}

// IamJobSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamJobSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	Type      *string          `db:"type" `
	Status    *string          `db:"status" `
	CreatedAt *time.Time       `db:"created_at" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamJobSetter) SetColumns() []string {
	vals := make([]string, 0, 7)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Type != nil {
		vals = append(vals, "type")
	}
	if s.Status != nil {
		vals = append(vals, "status")
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

func (s IamJobSetter) Overwrite(t *IamJob) {
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
	if s.Type != nil {
		t.Type = func() string {
			if s.Type == nil {
				return *new(string)
			}
			return *s.Type
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

func (s *IamJobSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamJobs.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Type != nil {
			vals[2] = psql.Arg(func() string {
				if s.Type == nil {
					return *new(string)
				}
				return *s.Type
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Status != nil {
			vals[3] = psql.Arg(func() string {
				if s.Status == nil {
					return *new(string)
				}
				return *s.Status
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

func (s IamJobSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamJobSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Type != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "type")...),
			psql.Arg(s.Type),
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

// FindIamJob retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamJob(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamJob, error) {
	if len(cols) == 0 {
		return IamJobs.Query(
			sm.Where(IamJobs.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamJobs.Query(
		sm.Where(IamJobs.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamJobs.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamJobExists checks the presence of a single record by primary key
func IamJobExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamJobs.Query(
		sm.Where(IamJobs.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamJob is retrieved from the database
func (o *IamJob) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamJobs.AfterSelectHooks.RunHooks(ctx, exec, IamJobSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamJobs.AfterInsertHooks.RunHooks(ctx, exec, IamJobSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamJobs.AfterUpdateHooks.RunHooks(ctx, exec, IamJobSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamJobs.AfterDeleteHooks.RunHooks(ctx, exec, IamJobSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamJobs.AfterMergeHooks.RunHooks(ctx, exec, IamJobSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamJob
func (o *IamJob) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamJob) pkEQ() dialect.Expression {
	return psql.Quote("iam_jobs", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamJob
func (o *IamJob) Update(ctx context.Context, exec bob.Executor, s *IamJobSetter) error {
	v, err := IamJobs.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamJob record with an executor
func (o *IamJob) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamJobs.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamJob using the executor
func (o *IamJob) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamJobs.Query(
		sm.Where(IamJobs.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamJobSlice is retrieved from the database
func (o IamJobSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamJobs.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamJobs.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamJobs.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamJobs.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamJobs.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamJobSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_jobs", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamJobSlice) copyMatchingRows(from ...*IamJob) {
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
func (o IamJobSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamJobs.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamJob:
				o.copyMatchingRows(retrieved)
			case []*IamJob:
				o.copyMatchingRows(retrieved...)
			case IamJobSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamJob or a slice of IamJob
				// then run the AfterUpdateHooks on the slice
				_, err = IamJobs.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamJobSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamJobs.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamJob:
				o.copyMatchingRows(retrieved)
			case []*IamJob:
				o.copyMatchingRows(retrieved...)
			case IamJobSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamJob or a slice of IamJob
				// then run the AfterDeleteHooks on the slice
				_, err = IamJobs.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamJobSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamJobs.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamJob:
				o.copyMatchingRows(retrieved)
			case []*IamJob:
				o.copyMatchingRows(retrieved...)
			case IamJobSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamJob or a slice of IamJob
				// then run the AfterMergeHooks on the slice
				_, err = IamJobs.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamJobSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamJobSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamJobs.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamJobSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamJobs.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamJobSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamJobs.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamJobWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Type      psql.WhereMod[Q, string]
	Status    psql.WhereMod[Q, string]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamJobWhere[Q]) AliasedAs(alias string) iamJobWhere[Q] {
	return buildIamJobWhere[Q](buildIamJobColumns(alias))
}

func buildIamJobWhere[Q psql.Filterable](cols iamJobColumns) iamJobWhere[Q] {
	return iamJobWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Type:      psql.Where[Q, string](cols.Type.Expression),
		Status:    psql.Where[Q, string](cols.Status.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
