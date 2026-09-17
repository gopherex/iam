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

// IamSecurityDelivery is an object representing the database table.
type IamSecurityDelivery struct {
	ID          string          `db:"id,pk" `
	ProjectID   string          `db:"project_id" `
	Environment string          `db:"environment" `
	UserID      string          `db:"user_id" `
	DedupKey    string          `db:"dedup_key" `
	Status      string          `db:"status" `
	CreatedAt   time.Time       `db:"created_at" `
	Data        json.RawMessage `db:"data" `
	PrivateData string          `db:"private_data" `
}

// IamSecurityDeliverySlice is an alias for a slice of pointers to IamSecurityDelivery.
// This should almost always be used instead of []*IamSecurityDelivery.
type IamSecurityDeliverySlice []*IamSecurityDelivery

// IamSecurityDeliveries contains methods to work with the iam_security_deliveries table
var IamSecurityDeliveries = psql.NewTablex[*IamSecurityDelivery, IamSecurityDeliverySlice, *IamSecurityDeliverySetter]("", "iam_security_deliveries", buildIamSecurityDeliveryColumns("iam_security_deliveries"))

// IamSecurityDeliveriesQuery is a query on the iam_security_deliveries table
type IamSecurityDeliveriesQuery = *psql.ViewQuery[*IamSecurityDelivery, IamSecurityDeliverySlice]

func buildIamSecurityDeliveryColumns(tableName string) iamSecurityDeliveryColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "user_id", "dedup_key", "status", "created_at", "data", "private_data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSecurityDeliveryColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamSecurityDeliveryColumn(tableName, "id"),
		ProjectID:   buildIamSecurityDeliveryColumn(tableName, "project_id"),
		Environment: buildIamSecurityDeliveryColumn(tableName, "environment"),
		UserID:      buildIamSecurityDeliveryColumn(tableName, "user_id"),
		DedupKey:    buildIamSecurityDeliveryColumn(tableName, "dedup_key"),
		Status:      buildIamSecurityDeliveryColumn(tableName, "status"),
		CreatedAt:   buildIamSecurityDeliveryColumn(tableName, "created_at"),
		Data:        buildIamSecurityDeliveryColumn(tableName, "data"),
		PrivateData: buildIamSecurityDeliveryColumn(tableName, "private_data"),
	}
}

type iamSecurityDeliveryColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamSecurityDeliveryColumn
	ProjectID   iamSecurityDeliveryColumn
	Environment iamSecurityDeliveryColumn
	UserID      iamSecurityDeliveryColumn
	DedupKey    iamSecurityDeliveryColumn
	Status      iamSecurityDeliveryColumn
	CreatedAt   iamSecurityDeliveryColumn
	Data        iamSecurityDeliveryColumn
	PrivateData iamSecurityDeliveryColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSecurityDeliveryColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSecurityDeliveryColumns) AliasedAs(tableName string) iamSecurityDeliveryColumns {
	return buildIamSecurityDeliveryColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSecurityDeliveryColumns) Unqualified() iamSecurityDeliveryColumns {
	return buildIamSecurityDeliveryColumns("")
}

func buildIamSecurityDeliveryColumn(alias, name string) iamSecurityDeliveryColumn {
	return iamSecurityDeliveryColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSecurityDeliveryColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSecurityDeliveryColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSecurityDeliveryColumn) ShouldOmitParens() bool {
	return true
}

// IamSecurityDeliverySetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSecurityDeliverySetter struct {
	ID          *string          `db:"id,pk" `
	ProjectID   *string          `db:"project_id" `
	Environment *string          `db:"environment" `
	UserID      *string          `db:"user_id" `
	DedupKey    *string          `db:"dedup_key" `
	Status      *string          `db:"status" `
	CreatedAt   *time.Time       `db:"created_at" `
	Data        *json.RawMessage `db:"data" `
	PrivateData *string          `db:"private_data" `
}

func (s IamSecurityDeliverySetter) SetColumns() []string {
	vals := make([]string, 0, 9)
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
	if s.DedupKey != nil {
		vals = append(vals, "dedup_key")
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

func (s IamSecurityDeliverySetter) Overwrite(t *IamSecurityDelivery) {
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
	if s.DedupKey != nil {
		t.DedupKey = func() string {
			if s.DedupKey == nil {
				return *new(string)
			}
			return *s.DedupKey
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

func (s *IamSecurityDeliverySetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSecurityDeliveries.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.DedupKey != nil {
			vals[4] = psql.Arg(func() string {
				if s.DedupKey == nil {
					return *new(string)
				}
				return *s.DedupKey
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

		if s.PrivateData != nil {
			vals[8] = psql.Arg(func() string {
				if s.PrivateData == nil {
					return *new(string)
				}
				return *s.PrivateData
			}())
		} else {
			vals[8] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamSecurityDeliverySetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSecurityDeliverySetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.DedupKey != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "dedup_key")...),
			psql.Arg(s.DedupKey),
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

// FindIamSecurityDelivery retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSecurityDelivery(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamSecurityDelivery, error) {
	if len(cols) == 0 {
		return IamSecurityDeliveries.Query(
			sm.Where(IamSecurityDeliveries.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamSecurityDeliveries.Query(
		sm.Where(IamSecurityDeliveries.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamSecurityDeliveries.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSecurityDeliveryExists checks the presence of a single record by primary key
func IamSecurityDeliveryExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamSecurityDeliveries.Query(
		sm.Where(IamSecurityDeliveries.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSecurityDelivery is retrieved from the database
func (o *IamSecurityDelivery) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityDeliveries.AfterSelectHooks.RunHooks(ctx, exec, IamSecurityDeliverySlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityDeliveries.AfterInsertHooks.RunHooks(ctx, exec, IamSecurityDeliverySlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityDeliveries.AfterUpdateHooks.RunHooks(ctx, exec, IamSecurityDeliverySlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityDeliveries.AfterDeleteHooks.RunHooks(ctx, exec, IamSecurityDeliverySlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityDeliveries.AfterMergeHooks.RunHooks(ctx, exec, IamSecurityDeliverySlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSecurityDelivery
func (o *IamSecurityDelivery) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamSecurityDelivery) pkEQ() dialect.Expression {
	return psql.Quote("iam_security_deliveries", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSecurityDelivery
func (o *IamSecurityDelivery) Update(ctx context.Context, exec bob.Executor, s *IamSecurityDeliverySetter) error {
	v, err := IamSecurityDeliveries.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSecurityDelivery record with an executor
func (o *IamSecurityDelivery) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSecurityDeliveries.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSecurityDelivery using the executor
func (o *IamSecurityDelivery) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSecurityDeliveries.Query(
		sm.Where(IamSecurityDeliveries.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSecurityDeliverySlice is retrieved from the database
func (o IamSecurityDeliverySlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityDeliveries.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityDeliveries.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityDeliveries.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityDeliveries.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityDeliveries.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSecurityDeliverySlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_security_deliveries", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSecurityDeliverySlice) copyMatchingRows(from ...*IamSecurityDelivery) {
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
func (o IamSecurityDeliverySlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityDeliveries.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityDelivery:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityDelivery:
				o.copyMatchingRows(retrieved...)
			case IamSecurityDeliverySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityDelivery or a slice of IamSecurityDelivery
				// then run the AfterUpdateHooks on the slice
				_, err = IamSecurityDeliveries.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSecurityDeliverySlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityDeliveries.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityDelivery:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityDelivery:
				o.copyMatchingRows(retrieved...)
			case IamSecurityDeliverySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityDelivery or a slice of IamSecurityDelivery
				// then run the AfterDeleteHooks on the slice
				_, err = IamSecurityDeliveries.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSecurityDeliverySlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityDeliveries.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityDelivery:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityDelivery:
				o.copyMatchingRows(retrieved...)
			case IamSecurityDeliverySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityDelivery or a slice of IamSecurityDelivery
				// then run the AfterMergeHooks on the slice
				_, err = IamSecurityDeliveries.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSecurityDeliverySlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSecurityDeliverySetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityDeliveries.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSecurityDeliverySlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityDeliveries.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSecurityDeliverySlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSecurityDeliveries.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSecurityDeliveryWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	UserID      psql.WhereMod[Q, string]
	DedupKey    psql.WhereMod[Q, string]
	Status      psql.WhereMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
	PrivateData psql.WhereMod[Q, string]
}

func (iamSecurityDeliveryWhere[Q]) AliasedAs(alias string) iamSecurityDeliveryWhere[Q] {
	return buildIamSecurityDeliveryWhere[Q](buildIamSecurityDeliveryColumns(alias))
}

func buildIamSecurityDeliveryWhere[Q psql.Filterable](cols iamSecurityDeliveryColumns) iamSecurityDeliveryWhere[Q] {
	return iamSecurityDeliveryWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		UserID:      psql.Where[Q, string](cols.UserID.Expression),
		DedupKey:    psql.Where[Q, string](cols.DedupKey.Expression),
		Status:      psql.Where[Q, string](cols.Status.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
		PrivateData: psql.Where[Q, string](cols.PrivateData.Expression),
	}
}
