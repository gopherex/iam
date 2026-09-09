// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
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

// IamUserRole is an object representing the database table.
type IamUserRole struct {
	ProjectID   string    `db:"project_id,pk" `
	Environment string    `db:"environment,pk" `
	UserID      string    `db:"user_id,pk" `
	Role        string    `db:"role,pk" `
	CreatedAt   time.Time `db:"created_at" `
}

// IamUserRoleSlice is an alias for a slice of pointers to IamUserRole.
// This should almost always be used instead of []*IamUserRole.
type IamUserRoleSlice []*IamUserRole

// IamUserRoles contains methods to work with the iam_user_roles table
var IamUserRoles = psql.NewTablex[*IamUserRole, IamUserRoleSlice, *IamUserRoleSetter]("", "iam_user_roles", buildIamUserRoleColumns("iam_user_roles"))

// IamUserRolesQuery is a query on the iam_user_roles table
type IamUserRolesQuery = *psql.ViewQuery[*IamUserRole, IamUserRoleSlice]

func buildIamUserRoleColumns(tableName string) iamUserRoleColumns {
	columnsExpr := expr.NewColumnsExpr(
		"project_id", "environment", "user_id", "role", "created_at",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamUserRoleColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ProjectID:   buildIamUserRoleColumn(tableName, "project_id"),
		Environment: buildIamUserRoleColumn(tableName, "environment"),
		UserID:      buildIamUserRoleColumn(tableName, "user_id"),
		Role:        buildIamUserRoleColumn(tableName, "role"),
		CreatedAt:   buildIamUserRoleColumn(tableName, "created_at"),
	}
}

type iamUserRoleColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ProjectID   iamUserRoleColumn
	Environment iamUserRoleColumn
	UserID      iamUserRoleColumn
	Role        iamUserRoleColumn
	CreatedAt   iamUserRoleColumn
}

// Alias returns the current table alias for the columns set.
func (c iamUserRoleColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamUserRoleColumns) AliasedAs(tableName string) iamUserRoleColumns {
	return buildIamUserRoleColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamUserRoleColumns) Unqualified() iamUserRoleColumns {
	return buildIamUserRoleColumns("")
}

func buildIamUserRoleColumn(alias, name string) iamUserRoleColumn {
	return iamUserRoleColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamUserRoleColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamUserRoleColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamUserRoleColumn) ShouldOmitParens() bool {
	return true
}

// IamUserRoleSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamUserRoleSetter struct {
	ProjectID   *string    `db:"project_id,pk" `
	Environment *string    `db:"environment,pk" `
	UserID      *string    `db:"user_id,pk" `
	Role        *string    `db:"role,pk" `
	CreatedAt   *time.Time `db:"created_at" `
}

func (s IamUserRoleSetter) SetColumns() []string {
	vals := make([]string, 0, 5)
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
	}
	if s.Role != nil {
		vals = append(vals, "role")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	return vals
}

func (s IamUserRoleSetter) Overwrite(t *IamUserRole) {
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
	if s.Role != nil {
		t.Role = func() string {
			if s.Role == nil {
				return *new(string)
			}
			return *s.Role
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
}

func (s *IamUserRoleSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamUserRoles.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.UserID != nil {
			vals[2] = psql.Arg(func() string {
				if s.UserID == nil {
					return *new(string)
				}
				return *s.UserID
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Role != nil {
			vals[3] = psql.Arg(func() string {
				if s.Role == nil {
					return *new(string)
				}
				return *s.Role
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[4] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamUserRoleSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamUserRoleSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.UserID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_id")...),
			psql.Arg(s.UserID),
		}})
	}

	if s.Role != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "role")...),
			psql.Arg(s.Role),
		}})
	}

	if s.CreatedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "created_at")...),
			psql.Arg(s.CreatedAt),
		}})
	}

	return exprs
}

// FindIamUserRole retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamUserRole(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, UserIDPK string, RolePK string, cols ...string) (*IamUserRole, error) {
	if len(cols) == 0 {
		return IamUserRoles.Query(
			sm.Where(IamUserRoles.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
			sm.Where(IamUserRoles.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
			sm.Where(IamUserRoles.Columns.UserID.EQ(psql.Arg(UserIDPK))),
			sm.Where(IamUserRoles.Columns.Role.EQ(psql.Arg(RolePK))),
		).One(ctx, exec)
	}

	return IamUserRoles.Query(
		sm.Where(IamUserRoles.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamUserRoles.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamUserRoles.Columns.UserID.EQ(psql.Arg(UserIDPK))),
		sm.Where(IamUserRoles.Columns.Role.EQ(psql.Arg(RolePK))),
		sm.Columns(IamUserRoles.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamUserRoleExists checks the presence of a single record by primary key
func IamUserRoleExists(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, UserIDPK string, RolePK string) (bool, error) {
	return IamUserRoles.Query(
		sm.Where(IamUserRoles.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamUserRoles.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamUserRoles.Columns.UserID.EQ(psql.Arg(UserIDPK))),
		sm.Where(IamUserRoles.Columns.Role.EQ(psql.Arg(RolePK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamUserRole is retrieved from the database
func (o *IamUserRole) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamUserRoles.AfterSelectHooks.RunHooks(ctx, exec, IamUserRoleSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamUserRoles.AfterInsertHooks.RunHooks(ctx, exec, IamUserRoleSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamUserRoles.AfterUpdateHooks.RunHooks(ctx, exec, IamUserRoleSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamUserRoles.AfterDeleteHooks.RunHooks(ctx, exec, IamUserRoleSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamUserRoles.AfterMergeHooks.RunHooks(ctx, exec, IamUserRoleSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamUserRole
func (o *IamUserRole) primaryKeyVals() bob.Expression {
	return psql.ArgGroup(
		o.ProjectID,
		o.Environment,
		o.UserID,
		o.Role,
	)
}

func (o *IamUserRole) pkEQ() dialect.Expression {
	return psql.Group(psql.Quote("iam_user_roles", "project_id"), psql.Quote("iam_user_roles", "environment"), psql.Quote("iam_user_roles", "user_id"), psql.Quote("iam_user_roles", "role")).EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamUserRole
func (o *IamUserRole) Update(ctx context.Context, exec bob.Executor, s *IamUserRoleSetter) error {
	v, err := IamUserRoles.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamUserRole record with an executor
func (o *IamUserRole) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamUserRoles.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamUserRole using the executor
func (o *IamUserRole) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamUserRoles.Query(
		sm.Where(IamUserRoles.Columns.ProjectID.EQ(psql.Arg(o.ProjectID))),
		sm.Where(IamUserRoles.Columns.Environment.EQ(psql.Arg(o.Environment))),
		sm.Where(IamUserRoles.Columns.UserID.EQ(psql.Arg(o.UserID))),
		sm.Where(IamUserRoles.Columns.Role.EQ(psql.Arg(o.Role))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamUserRoleSlice is retrieved from the database
func (o IamUserRoleSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamUserRoles.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamUserRoles.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamUserRoles.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamUserRoles.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamUserRoles.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamUserRoleSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Group(psql.Quote("iam_user_roles", "project_id"), psql.Quote("iam_user_roles", "environment"), psql.Quote("iam_user_roles", "user_id"), psql.Quote("iam_user_roles", "role")).In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamUserRoleSlice) copyMatchingRows(from ...*IamUserRole) {
	for i, old := range o {
		for _, new := range from {
			if new.ProjectID != old.ProjectID {
				continue
			}
			if new.Environment != old.Environment {
				continue
			}
			if new.UserID != old.UserID {
				continue
			}
			if new.Role != old.Role {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamUserRoleSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamUserRoles.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamUserRole:
				o.copyMatchingRows(retrieved)
			case []*IamUserRole:
				o.copyMatchingRows(retrieved...)
			case IamUserRoleSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamUserRole or a slice of IamUserRole
				// then run the AfterUpdateHooks on the slice
				_, err = IamUserRoles.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamUserRoleSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamUserRoles.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamUserRole:
				o.copyMatchingRows(retrieved)
			case []*IamUserRole:
				o.copyMatchingRows(retrieved...)
			case IamUserRoleSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamUserRole or a slice of IamUserRole
				// then run the AfterDeleteHooks on the slice
				_, err = IamUserRoles.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamUserRoleSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamUserRoles.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamUserRole:
				o.copyMatchingRows(retrieved)
			case []*IamUserRole:
				o.copyMatchingRows(retrieved...)
			case IamUserRoleSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamUserRole or a slice of IamUserRole
				// then run the AfterMergeHooks on the slice
				_, err = IamUserRoles.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamUserRoleSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamUserRoleSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamUserRoles.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamUserRoleSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamUserRoles.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamUserRoleSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamUserRoles.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamUserRoleWhere[Q psql.Filterable] struct {
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	UserID      psql.WhereMod[Q, string]
	Role        psql.WhereMod[Q, string]
	CreatedAt   psql.WhereMod[Q, time.Time]
}

func (iamUserRoleWhere[Q]) AliasedAs(alias string) iamUserRoleWhere[Q] {
	return buildIamUserRoleWhere[Q](buildIamUserRoleColumns(alias))
}

func buildIamUserRoleWhere[Q psql.Filterable](cols iamUserRoleColumns) iamUserRoleWhere[Q] {
	return iamUserRoleWhere[Q]{
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		UserID:      psql.Where[Q, string](cols.UserID.Expression),
		Role:        psql.Where[Q, string](cols.Role.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
	}
}
