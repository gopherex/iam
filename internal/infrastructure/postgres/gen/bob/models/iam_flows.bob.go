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

// IamFlow is an object representing the database table.
type IamFlow struct {
	ID        string           `db:"id,pk" `
	ProjectID string           `db:"project_id" `
	TokenHash string           `db:"token_hash" `
	Kind      string           `db:"kind" `
	Status    string           `db:"status" `
	Step      string           `db:"step" `
	UserID    null.Val[string] `db:"user_id" `
	ExpiresAt time.Time        `db:"expires_at" `
	CreatedAt time.Time        `db:"created_at" `
	UpdatedAt time.Time        `db:"updated_at" `
	Data      json.RawMessage  `db:"data" `
}

// IamFlowSlice is an alias for a slice of pointers to IamFlow.
// This should almost always be used instead of []*IamFlow.
type IamFlowSlice []*IamFlow

// IamFlows contains methods to work with the iam_flows table
var IamFlows = psql.NewTablex[*IamFlow, IamFlowSlice, *IamFlowSetter]("", "iam_flows", buildIamFlowColumns("iam_flows"))

// IamFlowsQuery is a query on the iam_flows table
type IamFlowsQuery = *psql.ViewQuery[*IamFlow, IamFlowSlice]

func buildIamFlowColumns(tableName string) iamFlowColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "token_hash", "kind", "status", "step", "user_id", "expires_at", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamFlowColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamFlowColumn(tableName, "id"),
		ProjectID:   buildIamFlowColumn(tableName, "project_id"),
		TokenHash:   buildIamFlowColumn(tableName, "token_hash"),
		Kind:        buildIamFlowColumn(tableName, "kind"),
		Status:      buildIamFlowColumn(tableName, "status"),
		Step:        buildIamFlowColumn(tableName, "step"),
		UserID:      buildIamFlowColumn(tableName, "user_id"),
		ExpiresAt:   buildIamFlowColumn(tableName, "expires_at"),
		CreatedAt:   buildIamFlowColumn(tableName, "created_at"),
		UpdatedAt:   buildIamFlowColumn(tableName, "updated_at"),
		Data:        buildIamFlowColumn(tableName, "data"),
	}
}

type iamFlowColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamFlowColumn
	ProjectID  iamFlowColumn
	TokenHash  iamFlowColumn
	Kind       iamFlowColumn
	Status     iamFlowColumn
	Step       iamFlowColumn
	UserID     iamFlowColumn
	ExpiresAt  iamFlowColumn
	CreatedAt  iamFlowColumn
	UpdatedAt  iamFlowColumn
	Data       iamFlowColumn
}

// Alias returns the current table alias for the columns set.
func (c iamFlowColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamFlowColumns) AliasedAs(tableName string) iamFlowColumns {
	return buildIamFlowColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamFlowColumns) Unqualified() iamFlowColumns {
	return buildIamFlowColumns("")
}

func buildIamFlowColumn(alias, name string) iamFlowColumn {
	return iamFlowColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamFlowColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamFlowColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamFlowColumn) ShouldOmitParens() bool {
	return true
}

// IamFlowSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamFlowSetter struct {
	ID        *string           `db:"id,pk" `
	ProjectID *string           `db:"project_id" `
	TokenHash *string           `db:"token_hash" `
	Kind      *string           `db:"kind" `
	Status    *string           `db:"status" `
	Step      *string           `db:"step" `
	UserID    *null.Val[string] `db:"user_id" `
	ExpiresAt *time.Time        `db:"expires_at" `
	CreatedAt *time.Time        `db:"created_at" `
	UpdatedAt *time.Time        `db:"updated_at" `
	Data      *json.RawMessage  `db:"data" `
}

func (s IamFlowSetter) SetColumns() []string {
	vals := make([]string, 0, 11)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.TokenHash != nil {
		vals = append(vals, "token_hash")
	}
	if s.Kind != nil {
		vals = append(vals, "kind")
	}
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.Step != nil {
		vals = append(vals, "step")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
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

func (s IamFlowSetter) Overwrite(t *IamFlow) {
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
	if s.TokenHash != nil {
		t.TokenHash = func() string {
			if s.TokenHash == nil {
				return *new(string)
			}
			return *s.TokenHash
		}()
	}
	if s.Kind != nil {
		t.Kind = func() string {
			if s.Kind == nil {
				return *new(string)
			}
			return *s.Kind
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
	if s.Step != nil {
		t.Step = func() string {
			if s.Step == nil {
				return *new(string)
			}
			return *s.Step
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

func (s *IamFlowSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamFlows.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 11)
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

		if s.TokenHash != nil {
			vals[2] = psql.Arg(func() string {
				if s.TokenHash == nil {
					return *new(string)
				}
				return *s.TokenHash
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Kind != nil {
			vals[3] = psql.Arg(func() string {
				if s.Kind == nil {
					return *new(string)
				}
				return *s.Kind
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

		if s.Step != nil {
			vals[5] = psql.Arg(func() string {
				if s.Step == nil {
					return *new(string)
				}
				return *s.Step
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.UserID != nil {
			vals[6] = psql.Arg(func() null.Val[string] {
				if s.UserID == nil {
					return *new(null.Val[string])
				}
				v := s.UserID
				return *v
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[7] = psql.Arg(func() time.Time {
				if s.ExpiresAt == nil {
					return *new(time.Time)
				}
				return *s.ExpiresAt
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[8] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[8] = psql.Raw("DEFAULT")
		}

		if s.UpdatedAt != nil {
			vals[9] = psql.Arg(func() time.Time {
				if s.UpdatedAt == nil {
					return *new(time.Time)
				}
				return *s.UpdatedAt
			}())
		} else {
			vals[9] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[10] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[10] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamFlowSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamFlowSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 11)

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

	if s.TokenHash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "token_hash")...),
			psql.Arg(s.TokenHash),
		}})
	}

	if s.Kind != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "kind")...),
			psql.Arg(s.Kind),
		}})
	}

	if s.Status != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "status")...),
			psql.Arg(s.Status),
		}})
	}

	if s.Step != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "step")...),
			psql.Arg(s.Step),
		}})
	}

	if s.UserID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_id")...),
			psql.Arg(s.UserID),
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

// FindIamFlow retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamFlow(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamFlow, error) {
	if len(cols) == 0 {
		return IamFlows.Query(
			sm.Where(IamFlows.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamFlows.Query(
		sm.Where(IamFlows.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamFlows.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamFlowExists checks the presence of a single record by primary key
func IamFlowExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamFlows.Query(
		sm.Where(IamFlows.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamFlow is retrieved from the database
func (o *IamFlow) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamFlows.AfterSelectHooks.RunHooks(ctx, exec, IamFlowSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamFlows.AfterInsertHooks.RunHooks(ctx, exec, IamFlowSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamFlows.AfterUpdateHooks.RunHooks(ctx, exec, IamFlowSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamFlows.AfterDeleteHooks.RunHooks(ctx, exec, IamFlowSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamFlows.AfterMergeHooks.RunHooks(ctx, exec, IamFlowSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamFlow
func (o *IamFlow) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamFlow) pkEQ() dialect.Expression {
	return psql.Quote("iam_flows", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamFlow
func (o *IamFlow) Update(ctx context.Context, exec bob.Executor, s *IamFlowSetter) error {
	v, err := IamFlows.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamFlow record with an executor
func (o *IamFlow) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamFlows.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamFlow using the executor
func (o *IamFlow) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamFlows.Query(
		sm.Where(IamFlows.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamFlowSlice is retrieved from the database
func (o IamFlowSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamFlows.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamFlows.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamFlows.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamFlows.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamFlows.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamFlowSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_flows", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamFlowSlice) copyMatchingRows(from ...*IamFlow) {
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
func (o IamFlowSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamFlows.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamFlow:
				o.copyMatchingRows(retrieved)
			case []*IamFlow:
				o.copyMatchingRows(retrieved...)
			case IamFlowSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamFlow or a slice of IamFlow
				// then run the AfterUpdateHooks on the slice
				_, err = IamFlows.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamFlowSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamFlows.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamFlow:
				o.copyMatchingRows(retrieved)
			case []*IamFlow:
				o.copyMatchingRows(retrieved...)
			case IamFlowSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamFlow or a slice of IamFlow
				// then run the AfterDeleteHooks on the slice
				_, err = IamFlows.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamFlowSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamFlows.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamFlow:
				o.copyMatchingRows(retrieved)
			case []*IamFlow:
				o.copyMatchingRows(retrieved...)
			case IamFlowSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamFlow or a slice of IamFlow
				// then run the AfterMergeHooks on the slice
				_, err = IamFlows.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamFlowSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamFlowSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamFlows.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamFlowSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamFlows.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamFlowSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamFlows.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamFlowWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	TokenHash psql.WhereMod[Q, string]
	Kind      psql.WhereMod[Q, string]
	Status    psql.WhereMod[Q, string]
	Step      psql.WhereMod[Q, string]
	UserID    psql.WhereNullMod[Q, string]
	ExpiresAt psql.WhereMod[Q, time.Time]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamFlowWhere[Q]) AliasedAs(alias string) iamFlowWhere[Q] {
	return buildIamFlowWhere[Q](buildIamFlowColumns(alias))
}

func buildIamFlowWhere[Q psql.Filterable](cols iamFlowColumns) iamFlowWhere[Q] {
	return iamFlowWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		TokenHash: psql.Where[Q, string](cols.TokenHash.Expression),
		Kind:      psql.Where[Q, string](cols.Kind.Expression),
		Status:    psql.Where[Q, string](cols.Status.Expression),
		Step:      psql.Where[Q, string](cols.Step.Expression),
		UserID:    psql.WhereNull[Q, string](cols.UserID.Expression),
		ExpiresAt: psql.Where[Q, time.Time](cols.ExpiresAt.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
