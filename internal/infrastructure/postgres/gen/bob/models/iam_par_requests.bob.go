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

// IamParRequest is an object representing the database table.
type IamParRequest struct {
	ID         string           `db:"id,pk" `
	ProjectID  string           `db:"project_id" `
	RequestURI string           `db:"request_uri" `
	ClientID   null.Val[string] `db:"client_id" `
	ExpiresAt  time.Time        `db:"expires_at" `
	CreatedAt  time.Time        `db:"created_at" `
	Data       json.RawMessage  `db:"data" `
}

// IamParRequestSlice is an alias for a slice of pointers to IamParRequest.
// This should almost always be used instead of []*IamParRequest.
type IamParRequestSlice []*IamParRequest

// IamParRequests contains methods to work with the iam_par_requests table
var IamParRequests = psql.NewTablex[*IamParRequest, IamParRequestSlice, *IamParRequestSetter]("", "iam_par_requests", buildIamParRequestColumns("iam_par_requests"))

// IamParRequestsQuery is a query on the iam_par_requests table
type IamParRequestsQuery = *psql.ViewQuery[*IamParRequest, IamParRequestSlice]

func buildIamParRequestColumns(tableName string) iamParRequestColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "request_uri", "client_id", "expires_at", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamParRequestColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamParRequestColumn(tableName, "id"),
		ProjectID:   buildIamParRequestColumn(tableName, "project_id"),
		RequestURI:  buildIamParRequestColumn(tableName, "request_uri"),
		ClientID:    buildIamParRequestColumn(tableName, "client_id"),
		ExpiresAt:   buildIamParRequestColumn(tableName, "expires_at"),
		CreatedAt:   buildIamParRequestColumn(tableName, "created_at"),
		Data:        buildIamParRequestColumn(tableName, "data"),
	}
}

type iamParRequestColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamParRequestColumn
	ProjectID  iamParRequestColumn
	RequestURI iamParRequestColumn
	ClientID   iamParRequestColumn
	ExpiresAt  iamParRequestColumn
	CreatedAt  iamParRequestColumn
	Data       iamParRequestColumn
}

// Alias returns the current table alias for the columns set.
func (c iamParRequestColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamParRequestColumns) AliasedAs(tableName string) iamParRequestColumns {
	return buildIamParRequestColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamParRequestColumns) Unqualified() iamParRequestColumns {
	return buildIamParRequestColumns("")
}

func buildIamParRequestColumn(alias, name string) iamParRequestColumn {
	return iamParRequestColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamParRequestColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamParRequestColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamParRequestColumn) ShouldOmitParens() bool {
	return true
}

// IamParRequestSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamParRequestSetter struct {
	ID         *string           `db:"id,pk" `
	ProjectID  *string           `db:"project_id" `
	RequestURI *string           `db:"request_uri" `
	ClientID   *null.Val[string] `db:"client_id" `
	ExpiresAt  *time.Time        `db:"expires_at" `
	CreatedAt  *time.Time        `db:"created_at" `
	Data       *json.RawMessage  `db:"data" `
}

func (s IamParRequestSetter) SetColumns() []string {
	vals := make([]string, 0, 7)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.RequestURI != nil {
		vals = append(vals, "request_uri")
	}
	if s.ClientID != nil {
		vals = append(vals, "client_id")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamParRequestSetter) Overwrite(t *IamParRequest) {
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
	if s.RequestURI != nil {
		t.RequestURI = func() string {
			if s.RequestURI == nil {
				return *new(string)
			}
			return *s.RequestURI
		}()
	}
	if s.ClientID != nil {
		t.ClientID = func() null.Val[string] {
			if s.ClientID == nil {
				return *new(null.Val[string])
			}
			v := s.ClientID
			return *v
		}()
	}
	if s.ExpiresAt != nil {
		t.ExpiresAt = func() time.Time {
			if s.ExpiresAt == nil {
				return *new(time.Time)
			}
			return *s.ExpiresAt
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

func (s *IamParRequestSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamParRequests.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.RequestURI != nil {
			vals[2] = psql.Arg(func() string {
				if s.RequestURI == nil {
					return *new(string)
				}
				return *s.RequestURI
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.ClientID != nil {
			vals[3] = psql.Arg(func() null.Val[string] {
				if s.ClientID == nil {
					return *new(null.Val[string])
				}
				v := s.ClientID
				return *v
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[4] = psql.Arg(func() time.Time {
				if s.ExpiresAt == nil {
					return *new(time.Time)
				}
				return *s.ExpiresAt
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

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamParRequestSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamParRequestSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.RequestURI != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "request_uri")...),
			psql.Arg(s.RequestURI),
		}})
	}

	if s.ClientID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "client_id")...),
			psql.Arg(s.ClientID),
		}})
	}

	if s.ExpiresAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "expires_at")...),
			psql.Arg(s.ExpiresAt),
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

// FindIamParRequest retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamParRequest(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamParRequest, error) {
	if len(cols) == 0 {
		return IamParRequests.Query(
			sm.Where(IamParRequests.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamParRequests.Query(
		sm.Where(IamParRequests.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamParRequests.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamParRequestExists checks the presence of a single record by primary key
func IamParRequestExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamParRequests.Query(
		sm.Where(IamParRequests.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamParRequest is retrieved from the database
func (o *IamParRequest) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamParRequests.AfterSelectHooks.RunHooks(ctx, exec, IamParRequestSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamParRequests.AfterInsertHooks.RunHooks(ctx, exec, IamParRequestSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamParRequests.AfterUpdateHooks.RunHooks(ctx, exec, IamParRequestSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamParRequests.AfterDeleteHooks.RunHooks(ctx, exec, IamParRequestSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamParRequests.AfterMergeHooks.RunHooks(ctx, exec, IamParRequestSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamParRequest
func (o *IamParRequest) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamParRequest) pkEQ() dialect.Expression {
	return psql.Quote("iam_par_requests", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamParRequest
func (o *IamParRequest) Update(ctx context.Context, exec bob.Executor, s *IamParRequestSetter) error {
	v, err := IamParRequests.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamParRequest record with an executor
func (o *IamParRequest) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamParRequests.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamParRequest using the executor
func (o *IamParRequest) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamParRequests.Query(
		sm.Where(IamParRequests.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamParRequestSlice is retrieved from the database
func (o IamParRequestSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamParRequests.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamParRequests.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamParRequests.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamParRequests.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamParRequests.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamParRequestSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_par_requests", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamParRequestSlice) copyMatchingRows(from ...*IamParRequest) {
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
func (o IamParRequestSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamParRequests.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamParRequest:
				o.copyMatchingRows(retrieved)
			case []*IamParRequest:
				o.copyMatchingRows(retrieved...)
			case IamParRequestSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamParRequest or a slice of IamParRequest
				// then run the AfterUpdateHooks on the slice
				_, err = IamParRequests.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamParRequestSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamParRequests.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamParRequest:
				o.copyMatchingRows(retrieved)
			case []*IamParRequest:
				o.copyMatchingRows(retrieved...)
			case IamParRequestSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamParRequest or a slice of IamParRequest
				// then run the AfterDeleteHooks on the slice
				_, err = IamParRequests.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamParRequestSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamParRequests.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamParRequest:
				o.copyMatchingRows(retrieved)
			case []*IamParRequest:
				o.copyMatchingRows(retrieved...)
			case IamParRequestSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamParRequest or a slice of IamParRequest
				// then run the AfterMergeHooks on the slice
				_, err = IamParRequests.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamParRequestSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamParRequestSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamParRequests.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamParRequestSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamParRequests.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamParRequestSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamParRequests.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamParRequestWhere[Q psql.Filterable] struct {
	ID         psql.WhereMod[Q, string]
	ProjectID  psql.WhereMod[Q, string]
	RequestURI psql.WhereMod[Q, string]
	ClientID   psql.WhereNullMod[Q, string]
	ExpiresAt  psql.WhereMod[Q, time.Time]
	CreatedAt  psql.WhereMod[Q, time.Time]
	Data       psql.WhereMod[Q, json.RawMessage]
}

func (iamParRequestWhere[Q]) AliasedAs(alias string) iamParRequestWhere[Q] {
	return buildIamParRequestWhere[Q](buildIamParRequestColumns(alias))
}

func buildIamParRequestWhere[Q psql.Filterable](cols iamParRequestColumns) iamParRequestWhere[Q] {
	return iamParRequestWhere[Q]{
		ID:         psql.Where[Q, string](cols.ID.Expression),
		ProjectID:  psql.Where[Q, string](cols.ProjectID.Expression),
		RequestURI: psql.Where[Q, string](cols.RequestURI.Expression),
		ClientID:   psql.WhereNull[Q, string](cols.ClientID.Expression),
		ExpiresAt:  psql.Where[Q, time.Time](cols.ExpiresAt.Expression),
		CreatedAt:  psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:       psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
