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

// IamAppClient is an object representing the database table.
type IamAppClient struct {
	ID          string          `db:"id,pk" `
	ProjectID   string          `db:"project_id" `
	Environment string          `db:"environment" `
	Name        string          `db:"name" `
	Type        string          `db:"type" `
	CreatedAt   time.Time       `db:"created_at" `
	UpdatedAt   time.Time       `db:"updated_at" `
	Data        json.RawMessage `db:"data" `
}

// IamAppClientSlice is an alias for a slice of pointers to IamAppClient.
// This should almost always be used instead of []*IamAppClient.
type IamAppClientSlice []*IamAppClient

// IamAppClients contains methods to work with the iam_app_clients table
var IamAppClients = psql.NewTablex[*IamAppClient, IamAppClientSlice, *IamAppClientSetter]("", "iam_app_clients", buildIamAppClientColumns("iam_app_clients"))

// IamAppClientsQuery is a query on the iam_app_clients table
type IamAppClientsQuery = *psql.ViewQuery[*IamAppClient, IamAppClientSlice]

func buildIamAppClientColumns(tableName string) iamAppClientColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "name", "type", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamAppClientColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamAppClientColumn(tableName, "id"),
		ProjectID:   buildIamAppClientColumn(tableName, "project_id"),
		Environment: buildIamAppClientColumn(tableName, "environment"),
		Name:        buildIamAppClientColumn(tableName, "name"),
		Type:        buildIamAppClientColumn(tableName, "type"),
		CreatedAt:   buildIamAppClientColumn(tableName, "created_at"),
		UpdatedAt:   buildIamAppClientColumn(tableName, "updated_at"),
		Data:        buildIamAppClientColumn(tableName, "data"),
	}
}

type iamAppClientColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamAppClientColumn
	ProjectID   iamAppClientColumn
	Environment iamAppClientColumn
	Name        iamAppClientColumn
	Type        iamAppClientColumn
	CreatedAt   iamAppClientColumn
	UpdatedAt   iamAppClientColumn
	Data        iamAppClientColumn
}

// Alias returns the current table alias for the columns set.
func (c iamAppClientColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamAppClientColumns) AliasedAs(tableName string) iamAppClientColumns {
	return buildIamAppClientColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamAppClientColumns) Unqualified() iamAppClientColumns {
	return buildIamAppClientColumns("")
}

func buildIamAppClientColumn(alias, name string) iamAppClientColumn {
	return iamAppClientColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamAppClientColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamAppClientColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamAppClientColumn) ShouldOmitParens() bool {
	return true
}

// IamAppClientSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamAppClientSetter struct {
	ID          *string          `db:"id,pk" `
	ProjectID   *string          `db:"project_id" `
	Environment *string          `db:"environment" `
	Name        *string          `db:"name" `
	Type        *string          `db:"type" `
	CreatedAt   *time.Time       `db:"created_at" `
	UpdatedAt   *time.Time       `db:"updated_at" `
	Data        *json.RawMessage `db:"data" `
}

func (s IamAppClientSetter) SetColumns() []string {
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
	if s.Name != nil {
		vals = append(vals, "name")
	}
	if s.Type != nil {
		vals = append(vals, "type")
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

func (s IamAppClientSetter) Overwrite(t *IamAppClient) {
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
	if s.Name != nil {
		t.Name = func() string {
			if s.Name == nil {
				return *new(string)
			}
			return *s.Name
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

func (s *IamAppClientSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamAppClients.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Name != nil {
			vals[3] = psql.Arg(func() string {
				if s.Name == nil {
					return *new(string)
				}
				return *s.Name
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.Type != nil {
			vals[4] = psql.Arg(func() string {
				if s.Type == nil {
					return *new(string)
				}
				return *s.Type
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

func (s IamAppClientSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamAppClientSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Name != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "name")...),
			psql.Arg(s.Name),
		}})
	}

	if s.Type != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "type")...),
			psql.Arg(s.Type),
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

// FindIamAppClient retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamAppClient(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamAppClient, error) {
	if len(cols) == 0 {
		return IamAppClients.Query(
			sm.Where(IamAppClients.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamAppClients.Query(
		sm.Where(IamAppClients.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamAppClients.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamAppClientExists checks the presence of a single record by primary key
func IamAppClientExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamAppClients.Query(
		sm.Where(IamAppClients.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamAppClient is retrieved from the database
func (o *IamAppClient) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAppClients.AfterSelectHooks.RunHooks(ctx, exec, IamAppClientSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamAppClients.AfterInsertHooks.RunHooks(ctx, exec, IamAppClientSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamAppClients.AfterUpdateHooks.RunHooks(ctx, exec, IamAppClientSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamAppClients.AfterDeleteHooks.RunHooks(ctx, exec, IamAppClientSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamAppClients.AfterMergeHooks.RunHooks(ctx, exec, IamAppClientSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamAppClient
func (o *IamAppClient) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamAppClient) pkEQ() dialect.Expression {
	return psql.Quote("iam_app_clients", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamAppClient
func (o *IamAppClient) Update(ctx context.Context, exec bob.Executor, s *IamAppClientSetter) error {
	v, err := IamAppClients.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamAppClient record with an executor
func (o *IamAppClient) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamAppClients.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamAppClient using the executor
func (o *IamAppClient) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamAppClients.Query(
		sm.Where(IamAppClients.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamAppClientSlice is retrieved from the database
func (o IamAppClientSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamAppClients.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamAppClients.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamAppClients.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamAppClients.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamAppClients.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamAppClientSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_app_clients", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamAppClientSlice) copyMatchingRows(from ...*IamAppClient) {
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
func (o IamAppClientSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAppClients.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAppClient:
				o.copyMatchingRows(retrieved)
			case []*IamAppClient:
				o.copyMatchingRows(retrieved...)
			case IamAppClientSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAppClient or a slice of IamAppClient
				// then run the AfterUpdateHooks on the slice
				_, err = IamAppClients.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamAppClientSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAppClients.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAppClient:
				o.copyMatchingRows(retrieved)
			case []*IamAppClient:
				o.copyMatchingRows(retrieved...)
			case IamAppClientSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAppClient or a slice of IamAppClient
				// then run the AfterDeleteHooks on the slice
				_, err = IamAppClients.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamAppClientSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamAppClients.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamAppClient:
				o.copyMatchingRows(retrieved)
			case []*IamAppClient:
				o.copyMatchingRows(retrieved...)
			case IamAppClientSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamAppClient or a slice of IamAppClient
				// then run the AfterMergeHooks on the slice
				_, err = IamAppClients.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamAppClientSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamAppClientSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAppClients.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamAppClientSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamAppClients.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamAppClientSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamAppClients.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamAppClientWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	Name        psql.WhereMod[Q, string]
	Type        psql.WhereMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
	UpdatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamAppClientWhere[Q]) AliasedAs(alias string) iamAppClientWhere[Q] {
	return buildIamAppClientWhere[Q](buildIamAppClientColumns(alias))
}

func buildIamAppClientWhere[Q psql.Filterable](cols iamAppClientColumns) iamAppClientWhere[Q] {
	return iamAppClientWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		Name:        psql.Where[Q, string](cols.Name.Expression),
		Type:        psql.Where[Q, string](cols.Type.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt:   psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
