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

// IamFactor is an object representing the database table.
type IamFactor struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	UserID    string          `db:"user_id" `
	Type      string          `db:"type" `
	Status    string          `db:"status" `
	Secret    string          `db:"secret" `
	CreatedAt time.Time       `db:"created_at" `
	Data      json.RawMessage `db:"data" `
}

// IamFactorSlice is an alias for a slice of pointers to IamFactor.
// This should almost always be used instead of []*IamFactor.
type IamFactorSlice []*IamFactor

// IamFactors contains methods to work with the iam_factors table
var IamFactors = psql.NewTablex[*IamFactor, IamFactorSlice, *IamFactorSetter]("", "iam_factors", buildIamFactorColumns("iam_factors"))

// IamFactorsQuery is a query on the iam_factors table
type IamFactorsQuery = *psql.ViewQuery[*IamFactor, IamFactorSlice]

func buildIamFactorColumns(tableName string) iamFactorColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "user_id", "type", "status", "secret", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamFactorColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamFactorColumn(tableName, "id"),
		ProjectID:   buildIamFactorColumn(tableName, "project_id"),
		UserID:      buildIamFactorColumn(tableName, "user_id"),
		Type:        buildIamFactorColumn(tableName, "type"),
		Status:      buildIamFactorColumn(tableName, "status"),
		Secret:      buildIamFactorColumn(tableName, "secret"),
		CreatedAt:   buildIamFactorColumn(tableName, "created_at"),
		Data:        buildIamFactorColumn(tableName, "data"),
	}
}

type iamFactorColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamFactorColumn
	ProjectID  iamFactorColumn
	UserID     iamFactorColumn
	Type       iamFactorColumn
	Status     iamFactorColumn
	Secret     iamFactorColumn
	CreatedAt  iamFactorColumn
	Data       iamFactorColumn
}

// Alias returns the current table alias for the columns set.
func (c iamFactorColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamFactorColumns) AliasedAs(tableName string) iamFactorColumns {
	return buildIamFactorColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamFactorColumns) Unqualified() iamFactorColumns {
	return buildIamFactorColumns("")
}

func buildIamFactorColumn(alias, name string) iamFactorColumn {
	return iamFactorColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamFactorColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamFactorColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamFactorColumn) ShouldOmitParens() bool {
	return true
}

// IamFactorSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamFactorSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	UserID    *string          `db:"user_id" `
	Type      *string          `db:"type" `
	Status    *string          `db:"status" `
	Secret    *string          `db:"secret" `
	CreatedAt *time.Time       `db:"created_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamFactorSetter) SetColumns() []string {
	vals := make([]string, 0, 8)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
	}
	if s.Type != nil {
		vals = append(vals, "type")
	}
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.Secret != nil {
		vals = append(vals, "secret")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamFactorSetter) Overwrite(t *IamFactor) {
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
	if s.UserID != nil {
		t.UserID = func() string {
			if s.UserID == nil {
				return *new(string)
			}
			return *s.UserID
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
	if s.Secret != nil {
		t.Secret = func() string {
			if s.Secret == nil {
				return *new(string)
			}
			return *s.Secret
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

func (s *IamFactorSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamFactors.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Type != nil {
			vals[3] = psql.Arg(func() string {
				if s.Type == nil {
					return *new(string)
				}
				return *s.Type
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

		if s.Secret != nil {
			vals[5] = psql.Arg(func() string {
				if s.Secret == nil {
					return *new(string)
				}
				return *s.Secret
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

		if s.Data != nil {
			vals[7] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamFactorSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamFactorSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.UserID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_id")...),
			psql.Arg(s.UserID),
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

	if s.Secret != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "secret")...),
			psql.Arg(s.Secret),
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

// FindIamFactor retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamFactor(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamFactor, error) {
	if len(cols) == 0 {
		return IamFactors.Query(
			sm.Where(IamFactors.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamFactors.Query(
		sm.Where(IamFactors.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamFactors.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamFactorExists checks the presence of a single record by primary key
func IamFactorExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamFactors.Query(
		sm.Where(IamFactors.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamFactor is retrieved from the database
func (o *IamFactor) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamFactors.AfterSelectHooks.RunHooks(ctx, exec, IamFactorSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamFactors.AfterInsertHooks.RunHooks(ctx, exec, IamFactorSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamFactors.AfterUpdateHooks.RunHooks(ctx, exec, IamFactorSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamFactors.AfterDeleteHooks.RunHooks(ctx, exec, IamFactorSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamFactors.AfterMergeHooks.RunHooks(ctx, exec, IamFactorSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamFactor
func (o *IamFactor) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamFactor) pkEQ() dialect.Expression {
	return psql.Quote("iam_factors", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamFactor
func (o *IamFactor) Update(ctx context.Context, exec bob.Executor, s *IamFactorSetter) error {
	v, err := IamFactors.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamFactor record with an executor
func (o *IamFactor) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamFactors.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamFactor using the executor
func (o *IamFactor) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamFactors.Query(
		sm.Where(IamFactors.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamFactorSlice is retrieved from the database
func (o IamFactorSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamFactors.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamFactors.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamFactors.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamFactors.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamFactors.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamFactorSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_factors", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamFactorSlice) copyMatchingRows(from ...*IamFactor) {
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
func (o IamFactorSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamFactors.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamFactor:
				o.copyMatchingRows(retrieved)
			case []*IamFactor:
				o.copyMatchingRows(retrieved...)
			case IamFactorSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamFactor or a slice of IamFactor
				// then run the AfterUpdateHooks on the slice
				_, err = IamFactors.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamFactorSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamFactors.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamFactor:
				o.copyMatchingRows(retrieved)
			case []*IamFactor:
				o.copyMatchingRows(retrieved...)
			case IamFactorSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamFactor or a slice of IamFactor
				// then run the AfterDeleteHooks on the slice
				_, err = IamFactors.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamFactorSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamFactors.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamFactor:
				o.copyMatchingRows(retrieved)
			case []*IamFactor:
				o.copyMatchingRows(retrieved...)
			case IamFactorSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamFactor or a slice of IamFactor
				// then run the AfterMergeHooks on the slice
				_, err = IamFactors.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamFactorSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamFactorSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamFactors.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamFactorSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamFactors.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamFactorSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamFactors.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamFactorWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	UserID    psql.WhereMod[Q, string]
	Type      psql.WhereMod[Q, string]
	Status    psql.WhereMod[Q, string]
	Secret    psql.WhereMod[Q, string]
	CreatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamFactorWhere[Q]) AliasedAs(alias string) iamFactorWhere[Q] {
	return buildIamFactorWhere[Q](buildIamFactorColumns(alias))
}

func buildIamFactorWhere[Q psql.Filterable](cols iamFactorColumns) iamFactorWhere[Q] {
	return iamFactorWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		UserID:    psql.Where[Q, string](cols.UserID.Expression),
		Type:      psql.Where[Q, string](cols.Type.Expression),
		Status:    psql.Where[Q, string](cols.Status.Expression),
		Secret:    psql.Where[Q, string](cols.Secret.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
