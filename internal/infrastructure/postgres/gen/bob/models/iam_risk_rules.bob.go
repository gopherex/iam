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

// IamRiskRule is an object representing the database table.
type IamRiskRule struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	Enabled   bool            `db:"enabled" `
	CreatedAt time.Time       `db:"created_at" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamRiskRuleSlice is an alias for a slice of pointers to IamRiskRule.
// This should almost always be used instead of []*IamRiskRule.
type IamRiskRuleSlice []*IamRiskRule

// IamRiskRules contains methods to work with the iam_risk_rules table
var IamRiskRules = psql.NewTablex[*IamRiskRule, IamRiskRuleSlice, *IamRiskRuleSetter]("", "iam_risk_rules", buildIamRiskRuleColumns("iam_risk_rules"))

// IamRiskRulesQuery is a query on the iam_risk_rules table
type IamRiskRulesQuery = *psql.ViewQuery[*IamRiskRule, IamRiskRuleSlice]

func buildIamRiskRuleColumns(tableName string) iamRiskRuleColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "enabled", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamRiskRuleColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamRiskRuleColumn(tableName, "id"),
		ProjectID:   buildIamRiskRuleColumn(tableName, "project_id"),
		Enabled:     buildIamRiskRuleColumn(tableName, "enabled"),
		CreatedAt:   buildIamRiskRuleColumn(tableName, "created_at"),
		UpdatedAt:   buildIamRiskRuleColumn(tableName, "updated_at"),
		Data:        buildIamRiskRuleColumn(tableName, "data"),
	}
}

type iamRiskRuleColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamRiskRuleColumn
	ProjectID  iamRiskRuleColumn
	Enabled    iamRiskRuleColumn
	CreatedAt  iamRiskRuleColumn
	UpdatedAt  iamRiskRuleColumn
	Data       iamRiskRuleColumn
}

// Alias returns the current table alias for the columns set.
func (c iamRiskRuleColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamRiskRuleColumns) AliasedAs(tableName string) iamRiskRuleColumns {
	return buildIamRiskRuleColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamRiskRuleColumns) Unqualified() iamRiskRuleColumns {
	return buildIamRiskRuleColumns("")
}

func buildIamRiskRuleColumn(alias, name string) iamRiskRuleColumn {
	return iamRiskRuleColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamRiskRuleColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamRiskRuleColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamRiskRuleColumn) ShouldOmitParens() bool {
	return true
}

// IamRiskRuleSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamRiskRuleSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	Enabled   *bool            `db:"enabled" `
	CreatedAt *time.Time       `db:"created_at" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamRiskRuleSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Enabled != nil {
		vals = append(vals, "enabled")
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

func (s IamRiskRuleSetter) Overwrite(t *IamRiskRule) {
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
	if s.Enabled != nil {
		t.Enabled = func() bool {
			if s.Enabled == nil {
				return *new(bool)
			}
			return *s.Enabled
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

func (s *IamRiskRuleSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamRiskRules.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Enabled != nil {
			vals[2] = psql.Arg(func() bool {
				if s.Enabled == nil {
					return *new(bool)
				}
				return *s.Enabled
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[3] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
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

func (s IamRiskRuleSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamRiskRuleSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Enabled != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "enabled")...),
			psql.Arg(s.Enabled),
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

// FindIamRiskRule retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamRiskRule(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamRiskRule, error) {
	if len(cols) == 0 {
		return IamRiskRules.Query(
			sm.Where(IamRiskRules.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamRiskRules.Query(
		sm.Where(IamRiskRules.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamRiskRules.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamRiskRuleExists checks the presence of a single record by primary key
func IamRiskRuleExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamRiskRules.Query(
		sm.Where(IamRiskRules.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamRiskRule is retrieved from the database
func (o *IamRiskRule) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamRiskRules.AfterSelectHooks.RunHooks(ctx, exec, IamRiskRuleSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamRiskRules.AfterInsertHooks.RunHooks(ctx, exec, IamRiskRuleSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamRiskRules.AfterUpdateHooks.RunHooks(ctx, exec, IamRiskRuleSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamRiskRules.AfterDeleteHooks.RunHooks(ctx, exec, IamRiskRuleSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamRiskRules.AfterMergeHooks.RunHooks(ctx, exec, IamRiskRuleSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamRiskRule
func (o *IamRiskRule) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamRiskRule) pkEQ() dialect.Expression {
	return psql.Quote("iam_risk_rules", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamRiskRule
func (o *IamRiskRule) Update(ctx context.Context, exec bob.Executor, s *IamRiskRuleSetter) error {
	v, err := IamRiskRules.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamRiskRule record with an executor
func (o *IamRiskRule) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamRiskRules.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamRiskRule using the executor
func (o *IamRiskRule) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamRiskRules.Query(
		sm.Where(IamRiskRules.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamRiskRuleSlice is retrieved from the database
func (o IamRiskRuleSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamRiskRules.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamRiskRules.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamRiskRules.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamRiskRules.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamRiskRules.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamRiskRuleSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_risk_rules", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamRiskRuleSlice) copyMatchingRows(from ...*IamRiskRule) {
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
func (o IamRiskRuleSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRiskRules.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRiskRule:
				o.copyMatchingRows(retrieved)
			case []*IamRiskRule:
				o.copyMatchingRows(retrieved...)
			case IamRiskRuleSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRiskRule or a slice of IamRiskRule
				// then run the AfterUpdateHooks on the slice
				_, err = IamRiskRules.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamRiskRuleSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRiskRules.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRiskRule:
				o.copyMatchingRows(retrieved)
			case []*IamRiskRule:
				o.copyMatchingRows(retrieved...)
			case IamRiskRuleSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRiskRule or a slice of IamRiskRule
				// then run the AfterDeleteHooks on the slice
				_, err = IamRiskRules.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamRiskRuleSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamRiskRules.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamRiskRule:
				o.copyMatchingRows(retrieved)
			case []*IamRiskRule:
				o.copyMatchingRows(retrieved...)
			case IamRiskRuleSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamRiskRule or a slice of IamRiskRule
				// then run the AfterMergeHooks on the slice
				_, err = IamRiskRules.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamRiskRuleSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamRiskRuleSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamRiskRules.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamRiskRuleSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamRiskRules.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamRiskRuleSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamRiskRules.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamRiskRuleWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Enabled   psql.WhereMod[Q, bool]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamRiskRuleWhere[Q]) AliasedAs(alias string) iamRiskRuleWhere[Q] {
	return buildIamRiskRuleWhere[Q](buildIamRiskRuleColumns(alias))
}

func buildIamRiskRuleWhere[Q psql.Filterable](cols iamRiskRuleColumns) iamRiskRuleWhere[Q] {
	return iamRiskRuleWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Enabled:   psql.Where[Q, bool](cols.Enabled.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
