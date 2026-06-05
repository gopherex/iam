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

// IamScimResource is an object representing the database table.
type IamScimResource struct {
	ID           string           `db:"id,pk" `
	ProjectID    string           `db:"project_id" `
	ConnectionID string           `db:"connection_id" `
	ResourceType string           `db:"resource_type" `
	ExternalID   null.Val[string] `db:"external_id" `
	UserID       null.Val[string] `db:"user_id" `
	CreatedAt    time.Time        `db:"created_at" `
	UpdatedAt    time.Time        `db:"updated_at" `
	Data         json.RawMessage  `db:"data" `
}

// IamScimResourceSlice is an alias for a slice of pointers to IamScimResource.
// This should almost always be used instead of []*IamScimResource.
type IamScimResourceSlice []*IamScimResource

// IamScimResources contains methods to work with the iam_scim_resources table
var IamScimResources = psql.NewTablex[*IamScimResource, IamScimResourceSlice, *IamScimResourceSetter]("", "iam_scim_resources", buildIamScimResourceColumns("iam_scim_resources"))

// IamScimResourcesQuery is a query on the iam_scim_resources table
type IamScimResourcesQuery = *psql.ViewQuery[*IamScimResource, IamScimResourceSlice]

func buildIamScimResourceColumns(tableName string) iamScimResourceColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "connection_id", "resource_type", "external_id", "user_id", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamScimResourceColumns{
		ColumnsExpr:  columnsExpr,
		tableAlias:   tableName,
		ID:           buildIamScimResourceColumn(tableName, "id"),
		ProjectID:    buildIamScimResourceColumn(tableName, "project_id"),
		ConnectionID: buildIamScimResourceColumn(tableName, "connection_id"),
		ResourceType: buildIamScimResourceColumn(tableName, "resource_type"),
		ExternalID:   buildIamScimResourceColumn(tableName, "external_id"),
		UserID:       buildIamScimResourceColumn(tableName, "user_id"),
		CreatedAt:    buildIamScimResourceColumn(tableName, "created_at"),
		UpdatedAt:    buildIamScimResourceColumn(tableName, "updated_at"),
		Data:         buildIamScimResourceColumn(tableName, "data"),
	}
}

type iamScimResourceColumns struct {
	expr.ColumnsExpr
	tableAlias   string
	ID           iamScimResourceColumn
	ProjectID    iamScimResourceColumn
	ConnectionID iamScimResourceColumn
	ResourceType iamScimResourceColumn
	ExternalID   iamScimResourceColumn
	UserID       iamScimResourceColumn
	CreatedAt    iamScimResourceColumn
	UpdatedAt    iamScimResourceColumn
	Data         iamScimResourceColumn
}

// Alias returns the current table alias for the columns set.
func (c iamScimResourceColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamScimResourceColumns) AliasedAs(tableName string) iamScimResourceColumns {
	return buildIamScimResourceColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamScimResourceColumns) Unqualified() iamScimResourceColumns {
	return buildIamScimResourceColumns("")
}

func buildIamScimResourceColumn(alias, name string) iamScimResourceColumn {
	return iamScimResourceColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamScimResourceColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamScimResourceColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamScimResourceColumn) ShouldOmitParens() bool {
	return true
}

// IamScimResourceSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamScimResourceSetter struct {
	ID           *string           `db:"id,pk" `
	ProjectID    *string           `db:"project_id" `
	ConnectionID *string           `db:"connection_id" `
	ResourceType *string           `db:"resource_type" `
	ExternalID   *null.Val[string] `db:"external_id" `
	UserID       *null.Val[string] `db:"user_id" `
	CreatedAt    *time.Time        `db:"created_at" `
	UpdatedAt    *time.Time        `db:"updated_at" `
	Data         *json.RawMessage  `db:"data" `
}

func (s IamScimResourceSetter) SetColumns() []string {
	vals := make([]string, 0, 9)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.ConnectionID != nil {
		vals = append(vals, "connection_id")
	}
	if s.ResourceType != nil {
		vals = append(vals, "resource_type")
	}
	if s.ExternalID != nil {
		vals = append(vals, "external_id")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
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

func (s IamScimResourceSetter) Overwrite(t *IamScimResource) {
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
	if s.ConnectionID != nil {
		t.ConnectionID = func() string {
			if s.ConnectionID == nil {
				return *new(string)
			}
			return *s.ConnectionID
		}()
	}
	if s.ResourceType != nil {
		t.ResourceType = func() string {
			if s.ResourceType == nil {
				return *new(string)
			}
			return *s.ResourceType
		}()
	}
	if s.ExternalID != nil {
		t.ExternalID = func() null.Val[string] {
			if s.ExternalID == nil {
				return *new(null.Val[string])
			}
			v := s.ExternalID
			return *v
		}()
	}
	if s.UserID != nil {
		t.UserID = func() null.Val[string] {
			if s.UserID == nil {
				return *new(null.Val[string])
			}
			v := s.UserID
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

func (s *IamScimResourceSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamScimResources.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.ConnectionID != nil {
			vals[2] = psql.Arg(func() string {
				if s.ConnectionID == nil {
					return *new(string)
				}
				return *s.ConnectionID
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.ResourceType != nil {
			vals[3] = psql.Arg(func() string {
				if s.ResourceType == nil {
					return *new(string)
				}
				return *s.ResourceType
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.ExternalID != nil {
			vals[4] = psql.Arg(func() null.Val[string] {
				if s.ExternalID == nil {
					return *new(null.Val[string])
				}
				v := s.ExternalID
				return *v
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.UserID != nil {
			vals[5] = psql.Arg(func() null.Val[string] {
				if s.UserID == nil {
					return *new(null.Val[string])
				}
				v := s.UserID
				return *v
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

		if s.UpdatedAt != nil {
			vals[7] = psql.Arg(func() time.Time {
				if s.UpdatedAt == nil {
					return *new(time.Time)
				}
				return *s.UpdatedAt
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

func (s IamScimResourceSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamScimResourceSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.ConnectionID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "connection_id")...),
			psql.Arg(s.ConnectionID),
		}})
	}

	if s.ResourceType != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "resource_type")...),
			psql.Arg(s.ResourceType),
		}})
	}

	if s.ExternalID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "external_id")...),
			psql.Arg(s.ExternalID),
		}})
	}

	if s.UserID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_id")...),
			psql.Arg(s.UserID),
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

// FindIamScimResource retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamScimResource(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamScimResource, error) {
	if len(cols) == 0 {
		return IamScimResources.Query(
			sm.Where(IamScimResources.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamScimResources.Query(
		sm.Where(IamScimResources.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamScimResources.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamScimResourceExists checks the presence of a single record by primary key
func IamScimResourceExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamScimResources.Query(
		sm.Where(IamScimResources.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamScimResource is retrieved from the database
func (o *IamScimResource) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamScimResources.AfterSelectHooks.RunHooks(ctx, exec, IamScimResourceSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamScimResources.AfterInsertHooks.RunHooks(ctx, exec, IamScimResourceSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamScimResources.AfterUpdateHooks.RunHooks(ctx, exec, IamScimResourceSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamScimResources.AfterDeleteHooks.RunHooks(ctx, exec, IamScimResourceSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamScimResources.AfterMergeHooks.RunHooks(ctx, exec, IamScimResourceSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamScimResource
func (o *IamScimResource) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamScimResource) pkEQ() dialect.Expression {
	return psql.Quote("iam_scim_resources", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamScimResource
func (o *IamScimResource) Update(ctx context.Context, exec bob.Executor, s *IamScimResourceSetter) error {
	v, err := IamScimResources.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamScimResource record with an executor
func (o *IamScimResource) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamScimResources.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamScimResource using the executor
func (o *IamScimResource) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamScimResources.Query(
		sm.Where(IamScimResources.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamScimResourceSlice is retrieved from the database
func (o IamScimResourceSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamScimResources.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamScimResources.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamScimResources.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamScimResources.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamScimResources.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamScimResourceSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_scim_resources", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamScimResourceSlice) copyMatchingRows(from ...*IamScimResource) {
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
func (o IamScimResourceSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamScimResources.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamScimResource:
				o.copyMatchingRows(retrieved)
			case []*IamScimResource:
				o.copyMatchingRows(retrieved...)
			case IamScimResourceSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamScimResource or a slice of IamScimResource
				// then run the AfterUpdateHooks on the slice
				_, err = IamScimResources.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamScimResourceSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamScimResources.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamScimResource:
				o.copyMatchingRows(retrieved)
			case []*IamScimResource:
				o.copyMatchingRows(retrieved...)
			case IamScimResourceSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamScimResource or a slice of IamScimResource
				// then run the AfterDeleteHooks on the slice
				_, err = IamScimResources.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamScimResourceSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamScimResources.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamScimResource:
				o.copyMatchingRows(retrieved)
			case []*IamScimResource:
				o.copyMatchingRows(retrieved...)
			case IamScimResourceSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamScimResource or a slice of IamScimResource
				// then run the AfterMergeHooks on the slice
				_, err = IamScimResources.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamScimResourceSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamScimResourceSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamScimResources.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamScimResourceSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamScimResources.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamScimResourceSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamScimResources.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamScimResourceWhere[Q psql.Filterable] struct {
	ID           psql.WhereMod[Q, string]
	ProjectID    psql.WhereMod[Q, string]
	ConnectionID psql.WhereMod[Q, string]
	ResourceType psql.WhereMod[Q, string]
	ExternalID   psql.WhereNullMod[Q, string]
	UserID       psql.WhereNullMod[Q, string]
	CreatedAt    psql.WhereMod[Q, time.Time]
	UpdatedAt    psql.WhereMod[Q, time.Time]
	Data         psql.WhereMod[Q, json.RawMessage]
}

func (iamScimResourceWhere[Q]) AliasedAs(alias string) iamScimResourceWhere[Q] {
	return buildIamScimResourceWhere[Q](buildIamScimResourceColumns(alias))
}

func buildIamScimResourceWhere[Q psql.Filterable](cols iamScimResourceColumns) iamScimResourceWhere[Q] {
	return iamScimResourceWhere[Q]{
		ID:           psql.Where[Q, string](cols.ID.Expression),
		ProjectID:    psql.Where[Q, string](cols.ProjectID.Expression),
		ConnectionID: psql.Where[Q, string](cols.ConnectionID.Expression),
		ResourceType: psql.Where[Q, string](cols.ResourceType.Expression),
		ExternalID:   psql.WhereNull[Q, string](cols.ExternalID.Expression),
		UserID:       psql.WhereNull[Q, string](cols.UserID.Expression),
		CreatedAt:    psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt:    psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:         psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
