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

// IamAccessRequest is an object representing the database table.
type IamAccessRequest struct {
	ID          string          `db:"id,pk" `
	ProjectID   string          `db:"project_id" `
	Environment string          `db:"environment" `
	Email       string          `db:"email" `
	Status      string          `db:"status" `
	CreatedAt   time.Time       `db:"created_at" `
	UpdatedAt   time.Time       `db:"updated_at" `
	Data        json.RawMessage `db:"data" `
}

// IamAccessRequestSlice is an alias for a slice of pointers to IamAccessRequest.
// This should almost always be used instead of []*IamAccessRequest.
type IamAccessRequestSlice []*IamAccessRequest

// IamAccessRequests contains methods to work with the iam_access_requests table
var IamAccessRequests = psql.NewTablex[*IamAccessRequest, IamAccessRequestSlice, *IamAccessRequestSetter]("", "iam_access_requests", buildIamAccessRequestColumns("iam_access_requests"))

// IamAccessRequestsQuery is a query on the iam_access_requests table
type IamAccessRequestsQuery = *psql.ViewQuery[*IamAccessRequest, IamAccessRequestSlice]

func buildIamAccessRequestColumns(tableName string) iamAccessRequestColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "email", "status", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamAccessRequestColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamAccessRequestColumn(tableName, "id"),
		ProjectID:   buildIamAccessRequestColumn(tableName, "project_id"),
		Environment: buildIamAccessRequestColumn(tableName, "environment"),
		Email:       buildIamAccessRequestColumn(tableName, "email"),
		Status:      buildIamAccessRequestColumn(tableName, "status"),
		CreatedAt:   buildIamAccessRequestColumn(tableName, "created_at"),
		UpdatedAt:   buildIamAccessRequestColumn(tableName, "updated_at"),
		Data:        buildIamAccessRequestColumn(tableName, "data"),
	}
}

type iamAccessRequestColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamAccessRequestColumn
	ProjectID   iamAccessRequestColumn
	Environment iamAccessRequestColumn
	Email       iamAccessRequestColumn
	Status      iamAccessRequestColumn
	CreatedAt   iamAccessRequestColumn
	UpdatedAt   iamAccessRequestColumn
	Data        iamAccessRequestColumn
}

// Alias returns the current table alias for the columns set.
func (c iamAccessRequestColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamAccessRequestColumns) AliasedAs(tableName string) iamAccessRequestColumns {
	return buildIamAccessRequestColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamAccessRequestColumns) Unqualified() iamAccessRequestColumns {
	return buildIamAccessRequestColumns("")
}

func buildIamAccessRequestColumn(alias, name string) iamAccessRequestColumn {
	return iamAccessRequestColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamAccessRequestColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamAccessRequestColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamAccessRequestColumn) ShouldOmitParens() bool {
	return true
}

// IamAccessRequestSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamAccessRequestSetter struct {
	ID          *string          `db:"id,pk" `
	ProjectID   *string          `db:"project_id" `
	Environment *string          `db:"environment" `
	Email       *string          `db:"email" `
	Status      *string          `db:"status" `
	CreatedAt   *time.Time       `db:"created_at" `
	UpdatedAt   *time.Time       `db:"updated_at" `
	Data        *json.RawMessage `db:"data" `
}

func (s IamAccessRequestSetter) SetColumns() []string {
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
	if s.Email != nil {
		vals = append(vals, "email")
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

func (s IamAccessRequestSetter) Overwrite(t *IamAccessRequest) {
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
	if s.Email != nil {
		t.Email = func() string {
			if s.Email == nil {
				return *new(string)
			}
			return *s.Email
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

func (s *IamAccessRequestSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamAccessRequests.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Email != nil {
			vals[3] = psql.Arg(func() string {
				if s.Email == nil {
					return *new(string)
				}
				return *s.Email
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

		if s.UpdatedAt != nil {
			vals[6] = psql.Arg(func() time.Time {
				if s.UpdatedAt == nil {
					return *new(time.Time)
				}
				return *s.UpdatedAt
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

func (s IamAccessRequestSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamAccessRequestSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Email != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "email")...),
			psql.Arg(s.Email),
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

// FindIamAccessRequest retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamAccessRequest(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamAccessRequest, error) {
	if len(cols) == 0 {
		return IamAccessRequests.Query(
			sm.Where(IamAccessRequests.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamAccessRequests.Query(
		sm.Where(IamAccessRequests.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamAccessRequests.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamAccessRequestExists checks the presence of a single record by primary key
func IamAccessRequestExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamAccessRequests.Query(
		sm.Where(IamAccessRequests.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamAccessRequest is retrieved from the database
func (o *IamAccessRequest) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAccessRequests.AfterSelectHooks.RunHooks(ctx, exec, IamAccessRequestSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamAccessRequests.AfterInsertHooks.RunHooks(ctx, exec, IamAccessRequestSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamAccessRequests.AfterUpdateHooks.RunHooks(ctx, exec, IamAccessRequestSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamAccessRequests.AfterDeleteHooks.RunHooks(ctx, exec, IamAccessRequestSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamAccessRequests.AfterMergeHooks.RunHooks(ctx, exec, IamAccessRequestSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamAccessRequest
func (o *IamAccessRequest) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamAccessRequest) pkEQ() dialect.Expression {
	return psql.Quote("iam_access_requests", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamAccessRequest
func (o *IamAccessRequest) Update(ctx context.Context, exec bob.Executor, s *IamAccessRequestSetter) error {
	v, err := IamAccessRequests.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamAccessRequest record with an executor
func (o *IamAccessRequest) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamAccessRequests.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamAccessRequest using the executor
func (o *IamAccessRequest) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamAccessRequests.Query(
		sm.Where(IamAccessRequests.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamAccessRequestSlice is retrieved from the database
func (o IamAccessRequestSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAccessRequests.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamAccessRequests.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamAccessRequests.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamAccessRequests.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamAccessRequests.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamAccessRequestSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_access_requests", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamAccessRequestSlice) copyMatchingRows(from ...*IamAccessRequest) {
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
func (o IamAccessRequestSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAccessRequests.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAccessRequest:
				o.copyMatchingRows(retrieved)
			case []*IamAccessRequest:
				o.copyMatchingRows(retrieved...)
			case IamAccessRequestSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAccessRequest or a slice of IamAccessRequest
				// then run the AfterUpdateHooks on the slice
				_, err = IamAccessRequests.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamAccessRequestSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAccessRequests.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAccessRequest:
				o.copyMatchingRows(retrieved)
			case []*IamAccessRequest:
				o.copyMatchingRows(retrieved...)
			case IamAccessRequestSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAccessRequest or a slice of IamAccessRequest
				// then run the AfterDeleteHooks on the slice
				_, err = IamAccessRequests.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamAccessRequestSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAccessRequests.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAccessRequest:
				o.copyMatchingRows(retrieved)
			case []*IamAccessRequest:
				o.copyMatchingRows(retrieved...)
			case IamAccessRequestSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAccessRequest or a slice of IamAccessRequest
				// then run the AfterMergeHooks on the slice
				_, err = IamAccessRequests.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamAccessRequestSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamAccessRequestSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAccessRequests.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamAccessRequestSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAccessRequests.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamAccessRequestSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamAccessRequests.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamAccessRequestWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	Email       psql.WhereMod[Q, string]
	Status      psql.WhereMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
	UpdatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamAccessRequestWhere[Q]) AliasedAs(alias string) iamAccessRequestWhere[Q] {
	return buildIamAccessRequestWhere[Q](buildIamAccessRequestColumns(alias))
}

func buildIamAccessRequestWhere[Q psql.Filterable](cols iamAccessRequestColumns) iamAccessRequestWhere[Q] {
	return iamAccessRequestWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		Email:       psql.Where[Q, string](cols.Email.Expression),
		Status:      psql.Where[Q, string](cols.Status.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt:   psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
