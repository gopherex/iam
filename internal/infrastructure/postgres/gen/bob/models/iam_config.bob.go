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

// IamConfig is an object representing the database table.
type IamConfig struct {
	ProjectID   string          `db:"project_id,pk" `
	Environment string          `db:"environment,pk" `
	Key         string          `db:"key,pk" `
	UpdatedAt   time.Time       `db:"updated_at" `
	Data        json.RawMessage `db:"data" `
}

// IamConfigSlice is an alias for a slice of pointers to IamConfig.
// This should almost always be used instead of []*IamConfig.
type IamConfigSlice []*IamConfig

// IamConfigs contains methods to work with the iam_config table
var IamConfigs = psql.NewTablex[*IamConfig, IamConfigSlice, *IamConfigSetter]("", "iam_config", buildIamConfigColumns("iam_config"))

// IamConfigsQuery is a query on the iam_config table
type IamConfigsQuery = *psql.ViewQuery[*IamConfig, IamConfigSlice]

func buildIamConfigColumns(tableName string) iamConfigColumns {
	columnsExpr := expr.NewColumnsExpr(
		"project_id", "environment", "key", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamConfigColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ProjectID:   buildIamConfigColumn(tableName, "project_id"),
		Environment: buildIamConfigColumn(tableName, "environment"),
		Key:         buildIamConfigColumn(tableName, "key"),
		UpdatedAt:   buildIamConfigColumn(tableName, "updated_at"),
		Data:        buildIamConfigColumn(tableName, "data"),
	}
}

type iamConfigColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ProjectID   iamConfigColumn
	Environment iamConfigColumn
	Key         iamConfigColumn
	UpdatedAt   iamConfigColumn
	Data        iamConfigColumn
}

// Alias returns the current table alias for the columns set.
func (c iamConfigColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamConfigColumns) AliasedAs(tableName string) iamConfigColumns {
	return buildIamConfigColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamConfigColumns) Unqualified() iamConfigColumns {
	return buildIamConfigColumns("")
}

func buildIamConfigColumn(alias, name string) iamConfigColumn {
	return iamConfigColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamConfigColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamConfigColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamConfigColumn) ShouldOmitParens() bool {
	return true
}

// IamConfigSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamConfigSetter struct {
	ProjectID   *string          `db:"project_id,pk" `
	Environment *string          `db:"environment,pk" `
	Key         *string          `db:"key,pk" `
	UpdatedAt   *time.Time       `db:"updated_at" `
	Data        *json.RawMessage `db:"data" `
}

func (s IamConfigSetter) SetColumns() []string {
	vals := make([]string, 0, 5)
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.Key != nil {
		vals = append(vals, "key")
	}
	if s.UpdatedAt != nil {
		vals = append(vals, "updated_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamConfigSetter) Overwrite(t *IamConfig) {
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
	if s.Key != nil {
		t.Key = func() string {
			if s.Key == nil {
				return *new(string)
			}
			return *s.Key
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

func (s *IamConfigSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamConfigs.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 5)
		if s.ProjectID != nil {
			vals[0] = psql.Arg(func() string {
				if s.ProjectID == nil {
					return *new(string)
				}
				return *s.ProjectID
			}())
		} else {
			vals[0] = psql.Raw("DEFAULT")
		}

		if s.Environment != nil {
			vals[1] = psql.Arg(func() string {
				if s.Environment == nil {
					return *new(string)
				}
				return *s.Environment
			}())
		} else {
			vals[1] = psql.Raw("DEFAULT")
		}

		if s.Key != nil {
			vals[2] = psql.Arg(func() string {
				if s.Key == nil {
					return *new(string)
				}
				return *s.Key
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.UpdatedAt != nil {
			vals[3] = psql.Arg(func() time.Time {
				if s.UpdatedAt == nil {
					return *new(time.Time)
				}
				return *s.UpdatedAt
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[4] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamConfigSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamConfigSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 5)

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

	if s.Key != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "key")...),
			psql.Arg(s.Key),
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

// FindIamConfig retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamConfig(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, KeyPK string, cols ...string) (*IamConfig, error) {
	if len(cols) == 0 {
		return IamConfigs.Query(
			sm.Where(IamConfigs.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
			sm.Where(IamConfigs.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
			sm.Where(IamConfigs.Columns.Key.EQ(psql.Arg(KeyPK))),
		).One(ctx, exec)
	}

	return IamConfigs.Query(
		sm.Where(IamConfigs.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamConfigs.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamConfigs.Columns.Key.EQ(psql.Arg(KeyPK))),
		sm.Columns(IamConfigs.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamConfigExists checks the presence of a single record by primary key
func IamConfigExists(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, KeyPK string) (bool, error) {
	return IamConfigs.Query(
		sm.Where(IamConfigs.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamConfigs.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamConfigs.Columns.Key.EQ(psql.Arg(KeyPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamConfig is retrieved from the database
func (o *IamConfig) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamConfigs.AfterSelectHooks.RunHooks(ctx, exec, IamConfigSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamConfigs.AfterInsertHooks.RunHooks(ctx, exec, IamConfigSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamConfigs.AfterUpdateHooks.RunHooks(ctx, exec, IamConfigSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamConfigs.AfterDeleteHooks.RunHooks(ctx, exec, IamConfigSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamConfigs.AfterMergeHooks.RunHooks(ctx, exec, IamConfigSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamConfig
func (o *IamConfig) primaryKeyVals() bob.Expression {
	return psql.ArgGroup(
		o.ProjectID,
		o.Environment,
		o.Key,
	)
}

func (o *IamConfig) pkEQ() dialect.Expression {
	return psql.Group(psql.Quote("iam_config", "project_id"), psql.Quote("iam_config", "environment"), psql.Quote("iam_config", "key")).EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamConfig
func (o *IamConfig) Update(ctx context.Context, exec bob.Executor, s *IamConfigSetter) error {
	v, err := IamConfigs.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamConfig record with an executor
func (o *IamConfig) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamConfigs.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamConfig using the executor
func (o *IamConfig) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamConfigs.Query(
		sm.Where(IamConfigs.Columns.ProjectID.EQ(psql.Arg(o.ProjectID))),
		sm.Where(IamConfigs.Columns.Environment.EQ(psql.Arg(o.Environment))),
		sm.Where(IamConfigs.Columns.Key.EQ(psql.Arg(o.Key))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamConfigSlice is retrieved from the database
func (o IamConfigSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamConfigs.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamConfigs.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamConfigs.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamConfigs.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamConfigs.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamConfigSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Group(psql.Quote("iam_config", "project_id"), psql.Quote("iam_config", "environment"), psql.Quote("iam_config", "key")).In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamConfigSlice) copyMatchingRows(from ...*IamConfig) {
	for i, old := range o {
		for _, new := range from {
			if new.ProjectID != old.ProjectID {
				continue
			}
			if new.Environment != old.Environment {
				continue
			}
			if new.Key != old.Key {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamConfigSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamConfigs.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamConfig:
				o.copyMatchingRows(retrieved)
			case []*IamConfig:
				o.copyMatchingRows(retrieved...)
			case IamConfigSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamConfig or a slice of IamConfig
				// then run the AfterUpdateHooks on the slice
				_, err = IamConfigs.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamConfigSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamConfigs.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamConfig:
				o.copyMatchingRows(retrieved)
			case []*IamConfig:
				o.copyMatchingRows(retrieved...)
			case IamConfigSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamConfig or a slice of IamConfig
				// then run the AfterDeleteHooks on the slice
				_, err = IamConfigs.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamConfigSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamConfigs.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamConfig:
				o.copyMatchingRows(retrieved)
			case []*IamConfig:
				o.copyMatchingRows(retrieved...)
			case IamConfigSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamConfig or a slice of IamConfig
				// then run the AfterMergeHooks on the slice
				_, err = IamConfigs.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamConfigSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamConfigSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamConfigs.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamConfigSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamConfigs.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamConfigSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamConfigs.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamConfigWhere[Q psql.Filterable] struct {
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	Key         psql.WhereMod[Q, string]
	UpdatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamConfigWhere[Q]) AliasedAs(alias string) iamConfigWhere[Q] {
	return buildIamConfigWhere[Q](buildIamConfigColumns(alias))
}

func buildIamConfigWhere[Q psql.Filterable](cols iamConfigColumns) iamConfigWhere[Q] {
	return iamConfigWhere[Q]{
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		Key:         psql.Where[Q, string](cols.Key.Expression),
		UpdatedAt:   psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
