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

// IamSsoConnection is an object representing the database table.
type IamSsoConnection struct {
	ID          string           `db:"id,pk" `
	ProjectID   string           `db:"project_id" `
	Type        string           `db:"type" `
	Status      string           `db:"status" `
	Name        string           `db:"name" `
	ExternalRef null.Val[string] `db:"external_ref" `
	CreatedAt   time.Time        `db:"created_at" `
	UpdatedAt   time.Time        `db:"updated_at" `
	Data        json.RawMessage  `db:"data" `
}

// IamSsoConnectionSlice is an alias for a slice of pointers to IamSsoConnection.
// This should almost always be used instead of []*IamSsoConnection.
type IamSsoConnectionSlice []*IamSsoConnection

// IamSsoConnections contains methods to work with the iam_sso_connections table
var IamSsoConnections = psql.NewTablex[*IamSsoConnection, IamSsoConnectionSlice, *IamSsoConnectionSetter]("", "iam_sso_connections", buildIamSsoConnectionColumns("iam_sso_connections"))

// IamSsoConnectionsQuery is a query on the iam_sso_connections table
type IamSsoConnectionsQuery = *psql.ViewQuery[*IamSsoConnection, IamSsoConnectionSlice]

func buildIamSsoConnectionColumns(tableName string) iamSsoConnectionColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "type", "status", "name", "external_ref", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSsoConnectionColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamSsoConnectionColumn(tableName, "id"),
		ProjectID:   buildIamSsoConnectionColumn(tableName, "project_id"),
		Type:        buildIamSsoConnectionColumn(tableName, "type"),
		Status:      buildIamSsoConnectionColumn(tableName, "status"),
		Name:        buildIamSsoConnectionColumn(tableName, "name"),
		ExternalRef: buildIamSsoConnectionColumn(tableName, "external_ref"),
		CreatedAt:   buildIamSsoConnectionColumn(tableName, "created_at"),
		UpdatedAt:   buildIamSsoConnectionColumn(tableName, "updated_at"),
		Data:        buildIamSsoConnectionColumn(tableName, "data"),
	}
}

type iamSsoConnectionColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamSsoConnectionColumn
	ProjectID   iamSsoConnectionColumn
	Type        iamSsoConnectionColumn
	Status      iamSsoConnectionColumn
	Name        iamSsoConnectionColumn
	ExternalRef iamSsoConnectionColumn
	CreatedAt   iamSsoConnectionColumn
	UpdatedAt   iamSsoConnectionColumn
	Data        iamSsoConnectionColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSsoConnectionColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSsoConnectionColumns) AliasedAs(tableName string) iamSsoConnectionColumns {
	return buildIamSsoConnectionColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSsoConnectionColumns) Unqualified() iamSsoConnectionColumns {
	return buildIamSsoConnectionColumns("")
}

func buildIamSsoConnectionColumn(alias, name string) iamSsoConnectionColumn {
	return iamSsoConnectionColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSsoConnectionColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSsoConnectionColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSsoConnectionColumn) ShouldOmitParens() bool {
	return true
}

// IamSsoConnectionSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSsoConnectionSetter struct {
	ID          *string           `db:"id,pk" `
	ProjectID   *string           `db:"project_id" `
	Type        *string           `db:"type" `
	Status      *string           `db:"status" `
	Name        *string           `db:"name" `
	ExternalRef *null.Val[string] `db:"external_ref" `
	CreatedAt   *time.Time        `db:"created_at" `
	UpdatedAt   *time.Time        `db:"updated_at" `
	Data        *json.RawMessage  `db:"data" `
}

func (s IamSsoConnectionSetter) SetColumns() []string {
	vals := make([]string, 0, 9)
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
	if s.Name != nil {
		vals = append(vals, "name")
	}
	if s.ExternalRef != nil {
		vals = append(vals, "external_ref")
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

func (s IamSsoConnectionSetter) Overwrite(t *IamSsoConnection) {
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
	if s.Name != nil {
		t.Name = func() string {
			if s.Name == nil {
				return *new(string)
			}
			return *s.Name
		}()
	}
	if s.ExternalRef != nil {
		t.ExternalRef = func() null.Val[string] {
			if s.ExternalRef == nil {
				return *new(null.Val[string])
			}
			v := s.ExternalRef
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

func (s *IamSsoConnectionSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSsoConnections.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Name != nil {
			vals[4] = psql.Arg(func() string {
				if s.Name == nil {
					return *new(string)
				}
				return *s.Name
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.ExternalRef != nil {
			vals[5] = psql.Arg(func() null.Val[string] {
				if s.ExternalRef == nil {
					return *new(null.Val[string])
				}
				v := s.ExternalRef
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

func (s IamSsoConnectionSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSsoConnectionSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Name != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "name")...),
			psql.Arg(s.Name),
		}})
	}

	if s.ExternalRef != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "external_ref")...),
			psql.Arg(s.ExternalRef),
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

// FindIamSsoConnection retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSsoConnection(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamSsoConnection, error) {
	if len(cols) == 0 {
		return IamSsoConnections.Query(
			sm.Where(IamSsoConnections.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamSsoConnections.Query(
		sm.Where(IamSsoConnections.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamSsoConnections.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSsoConnectionExists checks the presence of a single record by primary key
func IamSsoConnectionExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamSsoConnections.Query(
		sm.Where(IamSsoConnections.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSsoConnection is retrieved from the database
func (o *IamSsoConnection) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSsoConnections.AfterSelectHooks.RunHooks(ctx, exec, IamSsoConnectionSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSsoConnections.AfterInsertHooks.RunHooks(ctx, exec, IamSsoConnectionSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSsoConnections.AfterUpdateHooks.RunHooks(ctx, exec, IamSsoConnectionSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSsoConnections.AfterDeleteHooks.RunHooks(ctx, exec, IamSsoConnectionSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSsoConnections.AfterMergeHooks.RunHooks(ctx, exec, IamSsoConnectionSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSsoConnection
func (o *IamSsoConnection) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamSsoConnection) pkEQ() dialect.Expression {
	return psql.Quote("iam_sso_connections", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSsoConnection
func (o *IamSsoConnection) Update(ctx context.Context, exec bob.Executor, s *IamSsoConnectionSetter) error {
	v, err := IamSsoConnections.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSsoConnection record with an executor
func (o *IamSsoConnection) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSsoConnections.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSsoConnection using the executor
func (o *IamSsoConnection) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSsoConnections.Query(
		sm.Where(IamSsoConnections.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSsoConnectionSlice is retrieved from the database
func (o IamSsoConnectionSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSsoConnections.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSsoConnections.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSsoConnections.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSsoConnections.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSsoConnections.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSsoConnectionSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_sso_connections", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSsoConnectionSlice) copyMatchingRows(from ...*IamSsoConnection) {
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
func (o IamSsoConnectionSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSsoConnections.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSsoConnection:
				o.copyMatchingRows(retrieved)
			case []*IamSsoConnection:
				o.copyMatchingRows(retrieved...)
			case IamSsoConnectionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSsoConnection or a slice of IamSsoConnection
				// then run the AfterUpdateHooks on the slice
				_, err = IamSsoConnections.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSsoConnectionSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSsoConnections.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSsoConnection:
				o.copyMatchingRows(retrieved)
			case []*IamSsoConnection:
				o.copyMatchingRows(retrieved...)
			case IamSsoConnectionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSsoConnection or a slice of IamSsoConnection
				// then run the AfterDeleteHooks on the slice
				_, err = IamSsoConnections.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSsoConnectionSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSsoConnections.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSsoConnection:
				o.copyMatchingRows(retrieved)
			case []*IamSsoConnection:
				o.copyMatchingRows(retrieved...)
			case IamSsoConnectionSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSsoConnection or a slice of IamSsoConnection
				// then run the AfterMergeHooks on the slice
				_, err = IamSsoConnections.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSsoConnectionSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSsoConnectionSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSsoConnections.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSsoConnectionSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSsoConnections.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSsoConnectionSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSsoConnections.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSsoConnectionWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Type        psql.WhereMod[Q, string]
	Status      psql.WhereMod[Q, string]
	Name        psql.WhereMod[Q, string]
	ExternalRef psql.WhereNullMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
	UpdatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamSsoConnectionWhere[Q]) AliasedAs(alias string) iamSsoConnectionWhere[Q] {
	return buildIamSsoConnectionWhere[Q](buildIamSsoConnectionColumns(alias))
}

func buildIamSsoConnectionWhere[Q psql.Filterable](cols iamSsoConnectionColumns) iamSsoConnectionWhere[Q] {
	return iamSsoConnectionWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Type:        psql.Where[Q, string](cols.Type.Expression),
		Status:      psql.Where[Q, string](cols.Status.Expression),
		Name:        psql.Where[Q, string](cols.Name.Expression),
		ExternalRef: psql.WhereNull[Q, string](cols.ExternalRef.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt:   psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
