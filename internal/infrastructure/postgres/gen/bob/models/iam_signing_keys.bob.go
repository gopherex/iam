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

// IamSigningKey is an object representing the database table.
type IamSigningKey struct {
	Kid         string           `db:"kid,pk" `
	ProjectID   string           `db:"project_id" `
	Environment string           `db:"environment" `
	Alg         string           `db:"alg" `
	Use         string           `db:"use" `
	Status      string           `db:"status" `
	PrivatePem  null.Val[string] `db:"private_pem" `
	CreatedAt   time.Time        `db:"created_at" `
	Data        json.RawMessage  `db:"data" `
}

// IamSigningKeySlice is an alias for a slice of pointers to IamSigningKey.
// This should almost always be used instead of []*IamSigningKey.
type IamSigningKeySlice []*IamSigningKey

// IamSigningKeys contains methods to work with the iam_signing_keys table
var IamSigningKeys = psql.NewTablex[*IamSigningKey, IamSigningKeySlice, *IamSigningKeySetter]("", "iam_signing_keys", buildIamSigningKeyColumns("iam_signing_keys"))

// IamSigningKeysQuery is a query on the iam_signing_keys table
type IamSigningKeysQuery = *psql.ViewQuery[*IamSigningKey, IamSigningKeySlice]

func buildIamSigningKeyColumns(tableName string) iamSigningKeyColumns {
	columnsExpr := expr.NewColumnsExpr(
		"kid", "project_id", "environment", "alg", "use", "status", "private_pem", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSigningKeyColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		Kid:         buildIamSigningKeyColumn(tableName, "kid"),
		ProjectID:   buildIamSigningKeyColumn(tableName, "project_id"),
		Environment: buildIamSigningKeyColumn(tableName, "environment"),
		Alg:         buildIamSigningKeyColumn(tableName, "alg"),
		Use:         buildIamSigningKeyColumn(tableName, "use"),
		Status:      buildIamSigningKeyColumn(tableName, "status"),
		PrivatePem:  buildIamSigningKeyColumn(tableName, "private_pem"),
		CreatedAt:   buildIamSigningKeyColumn(tableName, "created_at"),
		Data:        buildIamSigningKeyColumn(tableName, "data"),
	}
}

type iamSigningKeyColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	Kid         iamSigningKeyColumn
	ProjectID   iamSigningKeyColumn
	Environment iamSigningKeyColumn
	Alg         iamSigningKeyColumn
	Use         iamSigningKeyColumn
	Status      iamSigningKeyColumn
	PrivatePem  iamSigningKeyColumn
	CreatedAt   iamSigningKeyColumn
	Data        iamSigningKeyColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSigningKeyColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSigningKeyColumns) AliasedAs(tableName string) iamSigningKeyColumns {
	return buildIamSigningKeyColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSigningKeyColumns) Unqualified() iamSigningKeyColumns {
	return buildIamSigningKeyColumns("")
}

func buildIamSigningKeyColumn(alias, name string) iamSigningKeyColumn {
	return iamSigningKeyColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSigningKeyColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSigningKeyColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSigningKeyColumn) ShouldOmitParens() bool {
	return true
}

// IamSigningKeySetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSigningKeySetter struct {
	Kid         *string           `db:"kid,pk" `
	ProjectID   *string           `db:"project_id" `
	Environment *string           `db:"environment" `
	Alg         *string           `db:"alg" `
	Use         *string           `db:"use" `
	Status      *string           `db:"status" `
	PrivatePem  *null.Val[string] `db:"private_pem" `
	CreatedAt   *time.Time        `db:"created_at" `
	Data        *json.RawMessage  `db:"data" `
}

func (s IamSigningKeySetter) SetColumns() []string {
	vals := make([]string, 0, 9)
	if s.Kid != nil {
		vals = append(vals, "kid")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.Alg != nil {
		vals = append(vals, "alg")
	}
	if s.Use != nil {
		vals = append(vals, "use")
	}
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.PrivatePem != nil {
		vals = append(vals, "private_pem")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamSigningKeySetter) Overwrite(t *IamSigningKey) {
	if s.Kid != nil {
		t.Kid = func() string {
			if s.Kid == nil {
				return *new(string)
			}
			return *s.Kid
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
	if s.Alg != nil {
		t.Alg = func() string {
			if s.Alg == nil {
				return *new(string)
			}
			return *s.Alg
		}()
	}
	if s.Use != nil {
		t.Use = func() string {
			if s.Use == nil {
				return *new(string)
			}
			return *s.Use
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
	if s.PrivatePem != nil {
		t.PrivatePem = func() null.Val[string] {
			if s.PrivatePem == nil {
				return *new(null.Val[string])
			}
			v := s.PrivatePem
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
	if s.Data != nil {
		t.Data = func() json.RawMessage {
			if s.Data == nil {
				return *new(json.RawMessage)
			}
			return *s.Data
		}()
	}
}

func (s *IamSigningKeySetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSigningKeys.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 9)
		if s.Kid != nil {
			vals[0] = psql.Arg(func() string {
				if s.Kid == nil {
					return *new(string)
				}
				return *s.Kid
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

		if s.Alg != nil {
			vals[3] = psql.Arg(func() string {
				if s.Alg == nil {
					return *new(string)
				}
				return *s.Alg
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.Use != nil {
			vals[4] = psql.Arg(func() string {
				if s.Use == nil {
					return *new(string)
				}
				return *s.Use
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.Status != nil {
			vals[5] = psql.Arg(func() string {
				if s.Status == nil {
					return *new(string)
				}
				return *s.Status
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.PrivatePem != nil {
			vals[6] = psql.Arg(func() null.Val[string] {
				if s.PrivatePem == nil {
					return *new(null.Val[string])
				}
				v := s.PrivatePem
				return *v
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

func (s IamSigningKeySetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSigningKeySetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 9)

	if s.Kid != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "kid")...),
			psql.Arg(s.Kid),
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

	if s.Alg != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "alg")...),
			psql.Arg(s.Alg),
		}})
	}

	if s.Use != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "use")...),
			psql.Arg(s.Use),
		}})
	}

	if s.Status != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "status")...),
			psql.Arg(s.Status),
		}})
	}

	if s.PrivatePem != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "private_pem")...),
			psql.Arg(s.PrivatePem),
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

// FindIamSigningKey retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSigningKey(ctx context.Context, exec bob.Executor, KidPK string, cols ...string) (*IamSigningKey, error) {
	if len(cols) == 0 {
		return IamSigningKeys.Query(
			sm.Where(IamSigningKeys.Columns.Kid.EQ(psql.Arg(KidPK))),
		).One(ctx, exec)
	}

	return IamSigningKeys.Query(
		sm.Where(IamSigningKeys.Columns.Kid.EQ(psql.Arg(KidPK))),
		sm.Columns(IamSigningKeys.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSigningKeyExists checks the presence of a single record by primary key
func IamSigningKeyExists(ctx context.Context, exec bob.Executor, KidPK string) (bool, error) {
	return IamSigningKeys.Query(
		sm.Where(IamSigningKeys.Columns.Kid.EQ(psql.Arg(KidPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSigningKey is retrieved from the database
func (o *IamSigningKey) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSigningKeys.AfterSelectHooks.RunHooks(ctx, exec, IamSigningKeySlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSigningKeys.AfterInsertHooks.RunHooks(ctx, exec, IamSigningKeySlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSigningKeys.AfterUpdateHooks.RunHooks(ctx, exec, IamSigningKeySlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSigningKeys.AfterDeleteHooks.RunHooks(ctx, exec, IamSigningKeySlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSigningKeys.AfterMergeHooks.RunHooks(ctx, exec, IamSigningKeySlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSigningKey
func (o *IamSigningKey) primaryKeyVals() bob.Expression {
	return psql.Arg(o.Kid)
}

func (o *IamSigningKey) pkEQ() dialect.Expression {
	return psql.Quote("iam_signing_keys", "kid").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSigningKey
func (o *IamSigningKey) Update(ctx context.Context, exec bob.Executor, s *IamSigningKeySetter) error {
	v, err := IamSigningKeys.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSigningKey record with an executor
func (o *IamSigningKey) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSigningKeys.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSigningKey using the executor
func (o *IamSigningKey) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSigningKeys.Query(
		sm.Where(IamSigningKeys.Columns.Kid.EQ(psql.Arg(o.Kid))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSigningKeySlice is retrieved from the database
func (o IamSigningKeySlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSigningKeys.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSigningKeys.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSigningKeys.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSigningKeys.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSigningKeys.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSigningKeySlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_signing_keys", "kid").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSigningKeySlice) copyMatchingRows(from ...*IamSigningKey) {
	for i, old := range o {
		for _, new := range from {
			if new.Kid != old.Kid {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamSigningKeySlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSigningKeys.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSigningKey:
				o.copyMatchingRows(retrieved)
			case []*IamSigningKey:
				o.copyMatchingRows(retrieved...)
			case IamSigningKeySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSigningKey or a slice of IamSigningKey
				// then run the AfterUpdateHooks on the slice
				_, err = IamSigningKeys.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSigningKeySlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSigningKeys.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSigningKey:
				o.copyMatchingRows(retrieved)
			case []*IamSigningKey:
				o.copyMatchingRows(retrieved...)
			case IamSigningKeySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSigningKey or a slice of IamSigningKey
				// then run the AfterDeleteHooks on the slice
				_, err = IamSigningKeys.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSigningKeySlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSigningKeys.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSigningKey:
				o.copyMatchingRows(retrieved)
			case []*IamSigningKey:
				o.copyMatchingRows(retrieved...)
			case IamSigningKeySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSigningKey or a slice of IamSigningKey
				// then run the AfterMergeHooks on the slice
				_, err = IamSigningKeys.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSigningKeySlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSigningKeySetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSigningKeys.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSigningKeySlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSigningKeys.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSigningKeySlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSigningKeys.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSigningKeyWhere[Q psql.Filterable] struct {
	Kid         psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	Alg         psql.WhereMod[Q, string]
	Use         psql.WhereMod[Q, string]
	Status      psql.WhereMod[Q, string]
	PrivatePem  psql.WhereNullMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamSigningKeyWhere[Q]) AliasedAs(alias string) iamSigningKeyWhere[Q] {
	return buildIamSigningKeyWhere[Q](buildIamSigningKeyColumns(alias))
}

func buildIamSigningKeyWhere[Q psql.Filterable](cols iamSigningKeyColumns) iamSigningKeyWhere[Q] {
	return iamSigningKeyWhere[Q]{
		Kid:         psql.Where[Q, string](cols.Kid.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		Alg:         psql.Where[Q, string](cols.Alg.Expression),
		Use:         psql.Where[Q, string](cols.Use.Expression),
		Status:      psql.Where[Q, string](cols.Status.Expression),
		PrivatePem:  psql.WhereNull[Q, string](cols.PrivatePem.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
