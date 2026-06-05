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

// IamEmailTemplate is an object representing the database table.
type IamEmailTemplate struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	Key       string          `db:"key" `
	Locale    string          `db:"locale" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamEmailTemplateSlice is an alias for a slice of pointers to IamEmailTemplate.
// This should almost always be used instead of []*IamEmailTemplate.
type IamEmailTemplateSlice []*IamEmailTemplate

// IamEmailTemplates contains methods to work with the iam_email_templates table
var IamEmailTemplates = psql.NewTablex[*IamEmailTemplate, IamEmailTemplateSlice, *IamEmailTemplateSetter]("", "iam_email_templates", buildIamEmailTemplateColumns("iam_email_templates"))

// IamEmailTemplatesQuery is a query on the iam_email_templates table
type IamEmailTemplatesQuery = *psql.ViewQuery[*IamEmailTemplate, IamEmailTemplateSlice]

func buildIamEmailTemplateColumns(tableName string) iamEmailTemplateColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "key", "locale", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamEmailTemplateColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamEmailTemplateColumn(tableName, "id"),
		ProjectID:   buildIamEmailTemplateColumn(tableName, "project_id"),
		Key:         buildIamEmailTemplateColumn(tableName, "key"),
		Locale:      buildIamEmailTemplateColumn(tableName, "locale"),
		UpdatedAt:   buildIamEmailTemplateColumn(tableName, "updated_at"),
		Data:        buildIamEmailTemplateColumn(tableName, "data"),
	}
}

type iamEmailTemplateColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamEmailTemplateColumn
	ProjectID  iamEmailTemplateColumn
	Key        iamEmailTemplateColumn
	Locale     iamEmailTemplateColumn
	UpdatedAt  iamEmailTemplateColumn
	Data       iamEmailTemplateColumn
}

// Alias returns the current table alias for the columns set.
func (c iamEmailTemplateColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamEmailTemplateColumns) AliasedAs(tableName string) iamEmailTemplateColumns {
	return buildIamEmailTemplateColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamEmailTemplateColumns) Unqualified() iamEmailTemplateColumns {
	return buildIamEmailTemplateColumns("")
}

func buildIamEmailTemplateColumn(alias, name string) iamEmailTemplateColumn {
	return iamEmailTemplateColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamEmailTemplateColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamEmailTemplateColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamEmailTemplateColumn) ShouldOmitParens() bool {
	return true
}

// IamEmailTemplateSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamEmailTemplateSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	Key       *string          `db:"key" `
	Locale    *string          `db:"locale" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamEmailTemplateSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Key != nil {
		vals = append(vals, "key")
	}
	if s.Locale != nil {
		vals = append(vals, "locale")
	}
	if s.UpdatedAt != nil {
		vals = append(vals, "updated_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamEmailTemplateSetter) Overwrite(t *IamEmailTemplate) {
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
	if s.Key != nil {
		t.Key = func() string {
			if s.Key == nil {
				return *new(string)
			}
			return *s.Key
		}()
	}
	if s.Locale != nil {
		t.Locale = func() string {
			if s.Locale == nil {
				return *new(string)
			}
			return *s.Locale
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

func (s *IamEmailTemplateSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamEmailTemplates.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Key != nil {
			vals[2] = psql.Arg(func() string {
				if s.Key == nil {
					return *new(string)
				}
				return *s.Key
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Locale != nil {
			vals[3] = psql.Arg(func() string {
				if s.Locale == nil {
					return *new(string)
				}
				return *s.Locale
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

func (s IamEmailTemplateSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamEmailTemplateSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Key != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "key")...),
			psql.Arg(s.Key),
		}})
	}

	if s.Locale != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "locale")...),
			psql.Arg(s.Locale),
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

// FindIamEmailTemplate retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamEmailTemplate(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamEmailTemplate, error) {
	if len(cols) == 0 {
		return IamEmailTemplates.Query(
			sm.Where(IamEmailTemplates.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamEmailTemplates.Query(
		sm.Where(IamEmailTemplates.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamEmailTemplates.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamEmailTemplateExists checks the presence of a single record by primary key
func IamEmailTemplateExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamEmailTemplates.Query(
		sm.Where(IamEmailTemplates.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamEmailTemplate is retrieved from the database
func (o *IamEmailTemplate) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamEmailTemplates.AfterSelectHooks.RunHooks(ctx, exec, IamEmailTemplateSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamEmailTemplates.AfterInsertHooks.RunHooks(ctx, exec, IamEmailTemplateSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamEmailTemplates.AfterUpdateHooks.RunHooks(ctx, exec, IamEmailTemplateSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamEmailTemplates.AfterDeleteHooks.RunHooks(ctx, exec, IamEmailTemplateSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamEmailTemplates.AfterMergeHooks.RunHooks(ctx, exec, IamEmailTemplateSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamEmailTemplate
func (o *IamEmailTemplate) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamEmailTemplate) pkEQ() dialect.Expression {
	return psql.Quote("iam_email_templates", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamEmailTemplate
func (o *IamEmailTemplate) Update(ctx context.Context, exec bob.Executor, s *IamEmailTemplateSetter) error {
	v, err := IamEmailTemplates.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamEmailTemplate record with an executor
func (o *IamEmailTemplate) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamEmailTemplates.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamEmailTemplate using the executor
func (o *IamEmailTemplate) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamEmailTemplates.Query(
		sm.Where(IamEmailTemplates.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamEmailTemplateSlice is retrieved from the database
func (o IamEmailTemplateSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamEmailTemplates.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamEmailTemplates.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamEmailTemplates.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamEmailTemplates.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamEmailTemplates.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamEmailTemplateSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_email_templates", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamEmailTemplateSlice) copyMatchingRows(from ...*IamEmailTemplate) {
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
func (o IamEmailTemplateSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEmailTemplates.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEmailTemplate:
				o.copyMatchingRows(retrieved)
			case []*IamEmailTemplate:
				o.copyMatchingRows(retrieved...)
			case IamEmailTemplateSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEmailTemplate or a slice of IamEmailTemplate
				// then run the AfterUpdateHooks on the slice
				_, err = IamEmailTemplates.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamEmailTemplateSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEmailTemplates.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEmailTemplate:
				o.copyMatchingRows(retrieved)
			case []*IamEmailTemplate:
				o.copyMatchingRows(retrieved...)
			case IamEmailTemplateSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEmailTemplate or a slice of IamEmailTemplate
				// then run the AfterDeleteHooks on the slice
				_, err = IamEmailTemplates.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamEmailTemplateSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamEmailTemplates.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamEmailTemplate:
				o.copyMatchingRows(retrieved)
			case []*IamEmailTemplate:
				o.copyMatchingRows(retrieved...)
			case IamEmailTemplateSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamEmailTemplate or a slice of IamEmailTemplate
				// then run the AfterMergeHooks on the slice
				_, err = IamEmailTemplates.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamEmailTemplateSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamEmailTemplateSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamEmailTemplates.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamEmailTemplateSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamEmailTemplates.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamEmailTemplateSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamEmailTemplates.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamEmailTemplateWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Key       psql.WhereMod[Q, string]
	Locale    psql.WhereMod[Q, string]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamEmailTemplateWhere[Q]) AliasedAs(alias string) iamEmailTemplateWhere[Q] {
	return buildIamEmailTemplateWhere[Q](buildIamEmailTemplateColumns(alias))
}

func buildIamEmailTemplateWhere[Q psql.Filterable](cols iamEmailTemplateColumns) iamEmailTemplateWhere[Q] {
	return iamEmailTemplateWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Key:       psql.Where[Q, string](cols.Key.Expression),
		Locale:    psql.Where[Q, string](cols.Locale.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
