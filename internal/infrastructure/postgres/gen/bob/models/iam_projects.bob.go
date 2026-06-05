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

// IamProject is an object representing the database table.
type IamProject struct {
	ID        string          `db:"id,pk" `
	Slug      string          `db:"slug" `
	Name      string          `db:"name" `
	CreatedAt time.Time       `db:"created_at" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamProjectSlice is an alias for a slice of pointers to IamProject.
// This should almost always be used instead of []*IamProject.
type IamProjectSlice []*IamProject

// IamProjects contains methods to work with the iam_projects table
var IamProjects = psql.NewTablex[*IamProject, IamProjectSlice, *IamProjectSetter]("", "iam_projects", buildIamProjectColumns("iam_projects"))

// IamProjectsQuery is a query on the iam_projects table
type IamProjectsQuery = *psql.ViewQuery[*IamProject, IamProjectSlice]

func buildIamProjectColumns(tableName string) iamProjectColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "slug", "name", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamProjectColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamProjectColumn(tableName, "id"),
		Slug:        buildIamProjectColumn(tableName, "slug"),
		Name:        buildIamProjectColumn(tableName, "name"),
		CreatedAt:   buildIamProjectColumn(tableName, "created_at"),
		UpdatedAt:   buildIamProjectColumn(tableName, "updated_at"),
		Data:        buildIamProjectColumn(tableName, "data"),
	}
}

type iamProjectColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamProjectColumn
	Slug       iamProjectColumn
	Name       iamProjectColumn
	CreatedAt  iamProjectColumn
	UpdatedAt  iamProjectColumn
	Data       iamProjectColumn
}

// Alias returns the current table alias for the columns set.
func (c iamProjectColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamProjectColumns) AliasedAs(tableName string) iamProjectColumns {
	return buildIamProjectColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamProjectColumns) Unqualified() iamProjectColumns {
	return buildIamProjectColumns("")
}

func buildIamProjectColumn(alias, name string) iamProjectColumn {
	return iamProjectColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamProjectColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamProjectColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamProjectColumn) ShouldOmitParens() bool {
	return true
}

// IamProjectSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamProjectSetter struct {
	ID        *string          `db:"id,pk" `
	Slug      *string          `db:"slug" `
	Name      *string          `db:"name" `
	CreatedAt *time.Time       `db:"created_at" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamProjectSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.Slug != nil {
		vals = append(vals, "slug")
	}
	if s.Name != nil {
		vals = append(vals, "name")
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

func (s IamProjectSetter) Overwrite(t *IamProject) {
	if s.ID != nil {
		t.ID = func() string {
			if s.ID == nil {
				return *new(string)
			}
			return *s.ID
		}()
	}
	if s.Slug != nil {
		t.Slug = func() string {
			if s.Slug == nil {
				return *new(string)
			}
			return *s.Slug
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

func (s *IamProjectSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamProjects.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Slug != nil {
			vals[1] = psql.Arg(func() string {
				if s.Slug == nil {
					return *new(string)
				}
				return *s.Slug
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

func (s IamProjectSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamProjectSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 6)

	if s.ID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "id")...),
			psql.Arg(s.ID),
		}})
	}

	if s.Slug != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "slug")...),
			psql.Arg(s.Slug),
		}})
	}

	if s.Name != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "name")...),
			psql.Arg(s.Name),
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

// FindIamProject retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamProject(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamProject, error) {
	if len(cols) == 0 {
		return IamProjects.Query(
			sm.Where(IamProjects.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamProjects.Query(
		sm.Where(IamProjects.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamProjects.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamProjectExists checks the presence of a single record by primary key
func IamProjectExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamProjects.Query(
		sm.Where(IamProjects.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamProject is retrieved from the database
func (o *IamProject) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamProjects.AfterSelectHooks.RunHooks(ctx, exec, IamProjectSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamProjects.AfterInsertHooks.RunHooks(ctx, exec, IamProjectSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamProjects.AfterUpdateHooks.RunHooks(ctx, exec, IamProjectSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamProjects.AfterDeleteHooks.RunHooks(ctx, exec, IamProjectSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamProjects.AfterMergeHooks.RunHooks(ctx, exec, IamProjectSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamProject
func (o *IamProject) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamProject) pkEQ() dialect.Expression {
	return psql.Quote("iam_projects", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamProject
func (o *IamProject) Update(ctx context.Context, exec bob.Executor, s *IamProjectSetter) error {
	v, err := IamProjects.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamProject record with an executor
func (o *IamProject) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamProjects.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamProject using the executor
func (o *IamProject) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamProjects.Query(
		sm.Where(IamProjects.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamProjectSlice is retrieved from the database
func (o IamProjectSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamProjects.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamProjects.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamProjects.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamProjects.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamProjects.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamProjectSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_projects", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamProjectSlice) copyMatchingRows(from ...*IamProject) {
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
func (o IamProjectSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamProjects.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamProject:
				o.copyMatchingRows(retrieved)
			case []*IamProject:
				o.copyMatchingRows(retrieved...)
			case IamProjectSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamProject or a slice of IamProject
				// then run the AfterUpdateHooks on the slice
				_, err = IamProjects.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamProjectSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamProjects.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamProject:
				o.copyMatchingRows(retrieved)
			case []*IamProject:
				o.copyMatchingRows(retrieved...)
			case IamProjectSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamProject or a slice of IamProject
				// then run the AfterDeleteHooks on the slice
				_, err = IamProjects.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamProjectSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamProjects.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamProject:
				o.copyMatchingRows(retrieved)
			case []*IamProject:
				o.copyMatchingRows(retrieved...)
			case IamProjectSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamProject or a slice of IamProject
				// then run the AfterMergeHooks on the slice
				_, err = IamProjects.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamProjectSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamProjectSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamProjects.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamProjectSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamProjects.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamProjectSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamProjects.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamProjectWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	Slug      psql.WhereMod[Q, string]
	Name      psql.WhereMod[Q, string]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamProjectWhere[Q]) AliasedAs(alias string) iamProjectWhere[Q] {
	return buildIamProjectWhere[Q](buildIamProjectColumns(alias))
}

func buildIamProjectWhere[Q psql.Filterable](cols iamProjectColumns) iamProjectWhere[Q] {
	return iamProjectWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		Slug:      psql.Where[Q, string](cols.Slug.Expression),
		Name:      psql.Where[Q, string](cols.Name.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
