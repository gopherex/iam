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

// IamIdentity is an object representing the database table.
type IamIdentity struct {
	ID                string           `db:"id,pk" `
	ProjectID         string           `db:"project_id" `
	Environment       string           `db:"environment" `
	UserID            string           `db:"user_id" `
	Type              string           `db:"type" `
	Provider          null.Val[string] `db:"provider" `
	ProviderAccountID null.Val[string] `db:"provider_account_id" `
	Email             null.Val[string] `db:"email" `
	CreatedAt         time.Time        `db:"created_at" `
	Data              json.RawMessage  `db:"data" `
}

// IamIdentitySlice is an alias for a slice of pointers to IamIdentity.
// This should almost always be used instead of []*IamIdentity.
type IamIdentitySlice []*IamIdentity

// IamIdentities contains methods to work with the iam_identities table
var IamIdentities = psql.NewTablex[*IamIdentity, IamIdentitySlice, *IamIdentitySetter]("", "iam_identities", buildIamIdentityColumns("iam_identities"))

// IamIdentitiesQuery is a query on the iam_identities table
type IamIdentitiesQuery = *psql.ViewQuery[*IamIdentity, IamIdentitySlice]

func buildIamIdentityColumns(tableName string) iamIdentityColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "user_id", "type", "provider", "provider_account_id", "email", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamIdentityColumns{
		ColumnsExpr:       columnsExpr,
		tableAlias:        tableName,
		ID:                buildIamIdentityColumn(tableName, "id"),
		ProjectID:         buildIamIdentityColumn(tableName, "project_id"),
		Environment:       buildIamIdentityColumn(tableName, "environment"),
		UserID:            buildIamIdentityColumn(tableName, "user_id"),
		Type:              buildIamIdentityColumn(tableName, "type"),
		Provider:          buildIamIdentityColumn(tableName, "provider"),
		ProviderAccountID: buildIamIdentityColumn(tableName, "provider_account_id"),
		Email:             buildIamIdentityColumn(tableName, "email"),
		CreatedAt:         buildIamIdentityColumn(tableName, "created_at"),
		Data:              buildIamIdentityColumn(tableName, "data"),
	}
}

type iamIdentityColumns struct {
	expr.ColumnsExpr
	tableAlias        string
	ID                iamIdentityColumn
	ProjectID         iamIdentityColumn
	Environment       iamIdentityColumn
	UserID            iamIdentityColumn
	Type              iamIdentityColumn
	Provider          iamIdentityColumn
	ProviderAccountID iamIdentityColumn
	Email             iamIdentityColumn
	CreatedAt         iamIdentityColumn
	Data              iamIdentityColumn
}

// Alias returns the current table alias for the columns set.
func (c iamIdentityColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamIdentityColumns) AliasedAs(tableName string) iamIdentityColumns {
	return buildIamIdentityColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamIdentityColumns) Unqualified() iamIdentityColumns {
	return buildIamIdentityColumns("")
}

func buildIamIdentityColumn(alias, name string) iamIdentityColumn {
	return iamIdentityColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamIdentityColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamIdentityColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamIdentityColumn) ShouldOmitParens() bool {
	return true
}

// IamIdentitySetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamIdentitySetter struct {
	ID                *string           `db:"id,pk" `
	ProjectID         *string           `db:"project_id" `
	Environment       *string           `db:"environment" `
	UserID            *string           `db:"user_id" `
	Type              *string           `db:"type" `
	Provider          *null.Val[string] `db:"provider" `
	ProviderAccountID *null.Val[string] `db:"provider_account_id" `
	Email             *null.Val[string] `db:"email" `
	CreatedAt         *time.Time        `db:"created_at" `
	Data              *json.RawMessage  `db:"data" `
}

func (s IamIdentitySetter) SetColumns() []string {
	vals := make([]string, 0, 10)
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
	if s.Type != nil {
		vals = append(vals, "type")
	}
	if s.Provider != nil {
		vals = append(vals, "provider")
	}
	if s.ProviderAccountID != nil {
		vals = append(vals, "provider_account_id")
	}
	if s.Email != nil {
		vals = append(vals, "email")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamIdentitySetter) Overwrite(t *IamIdentity) {
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
	if s.Type != nil {
		t.Type = func() string {
			if s.Type == nil {
				return *new(string)
			}
			return *s.Type
		}()
	}
	if s.Provider != nil {
		t.Provider = func() null.Val[string] {
			if s.Provider == nil {
				return *new(null.Val[string])
			}
			v := s.Provider
			return *v
		}()
	}
	if s.ProviderAccountID != nil {
		t.ProviderAccountID = func() null.Val[string] {
			if s.ProviderAccountID == nil {
				return *new(null.Val[string])
			}
			v := s.ProviderAccountID
			return *v
		}()
	}
	if s.Email != nil {
		t.Email = func() null.Val[string] {
			if s.Email == nil {
				return *new(null.Val[string])
			}
			v := s.Email
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
	if s.Data != nil {
		t.Data = func() json.RawMessage {
			if s.Data == nil {
				return *new(json.RawMessage)
			}
			return *s.Data
		}()
	}
}

func (s *IamIdentitySetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamIdentities.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 10)
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

		if s.Provider != nil {
			vals[5] = psql.Arg(func() null.Val[string] {
				if s.Provider == nil {
					return *new(null.Val[string])
				}
				v := s.Provider
				return *v
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.ProviderAccountID != nil {
			vals[6] = psql.Arg(func() null.Val[string] {
				if s.ProviderAccountID == nil {
					return *new(null.Val[string])
				}
				v := s.ProviderAccountID
				return *v
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.Email != nil {
			vals[7] = psql.Arg(func() null.Val[string] {
				if s.Email == nil {
					return *new(null.Val[string])
				}
				v := s.Email
				return *v
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

		if s.Data != nil {
			vals[9] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[9] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamIdentitySetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamIdentitySetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 10)

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

	if s.Type != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "type")...),
			psql.Arg(s.Type),
		}})
	}

	if s.Provider != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "provider")...),
			psql.Arg(s.Provider),
		}})
	}

	if s.ProviderAccountID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "provider_account_id")...),
			psql.Arg(s.ProviderAccountID),
		}})
	}

	if s.Email != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "email")...),
			psql.Arg(s.Email),
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

// FindIamIdentity retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamIdentity(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamIdentity, error) {
	if len(cols) == 0 {
		return IamIdentities.Query(
			sm.Where(IamIdentities.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamIdentities.Query(
		sm.Where(IamIdentities.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamIdentities.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamIdentityExists checks the presence of a single record by primary key
func IamIdentityExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamIdentities.Query(
		sm.Where(IamIdentities.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamIdentity is retrieved from the database
func (o *IamIdentity) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamIdentities.AfterSelectHooks.RunHooks(ctx, exec, IamIdentitySlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamIdentities.AfterInsertHooks.RunHooks(ctx, exec, IamIdentitySlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamIdentities.AfterUpdateHooks.RunHooks(ctx, exec, IamIdentitySlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamIdentities.AfterDeleteHooks.RunHooks(ctx, exec, IamIdentitySlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamIdentities.AfterMergeHooks.RunHooks(ctx, exec, IamIdentitySlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamIdentity
func (o *IamIdentity) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamIdentity) pkEQ() dialect.Expression {
	return psql.Quote("iam_identities", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamIdentity
func (o *IamIdentity) Update(ctx context.Context, exec bob.Executor, s *IamIdentitySetter) error {
	v, err := IamIdentities.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamIdentity record with an executor
func (o *IamIdentity) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamIdentities.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamIdentity using the executor
func (o *IamIdentity) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamIdentities.Query(
		sm.Where(IamIdentities.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamIdentitySlice is retrieved from the database
func (o IamIdentitySlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamIdentities.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamIdentities.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamIdentities.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamIdentities.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamIdentities.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamIdentitySlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_identities", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamIdentitySlice) copyMatchingRows(from ...*IamIdentity) {
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
func (o IamIdentitySlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamIdentities.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamIdentity:
				o.copyMatchingRows(retrieved)
			case []*IamIdentity:
				o.copyMatchingRows(retrieved...)
			case IamIdentitySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamIdentity or a slice of IamIdentity
				// then run the AfterUpdateHooks on the slice
				_, err = IamIdentities.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamIdentitySlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamIdentities.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamIdentity:
				o.copyMatchingRows(retrieved)
			case []*IamIdentity:
				o.copyMatchingRows(retrieved...)
			case IamIdentitySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamIdentity or a slice of IamIdentity
				// then run the AfterDeleteHooks on the slice
				_, err = IamIdentities.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamIdentitySlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamIdentities.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamIdentity:
				o.copyMatchingRows(retrieved)
			case []*IamIdentity:
				o.copyMatchingRows(retrieved...)
			case IamIdentitySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamIdentity or a slice of IamIdentity
				// then run the AfterMergeHooks on the slice
				_, err = IamIdentities.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamIdentitySlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamIdentitySetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamIdentities.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamIdentitySlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamIdentities.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamIdentitySlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamIdentities.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamIdentityWhere[Q psql.Filterable] struct {
	ID                psql.WhereMod[Q, string]
	ProjectID         psql.WhereMod[Q, string]
	Environment       psql.WhereMod[Q, string]
	UserID            psql.WhereMod[Q, string]
	Type              psql.WhereMod[Q, string]
	Provider          psql.WhereNullMod[Q, string]
	ProviderAccountID psql.WhereNullMod[Q, string]
	Email             psql.WhereNullMod[Q, string]
	CreatedAt         psql.WhereMod[Q, time.Time]
	Data              psql.WhereMod[Q, json.RawMessage]
}

func (iamIdentityWhere[Q]) AliasedAs(alias string) iamIdentityWhere[Q] {
	return buildIamIdentityWhere[Q](buildIamIdentityColumns(alias))
}

func buildIamIdentityWhere[Q psql.Filterable](cols iamIdentityColumns) iamIdentityWhere[Q] {
	return iamIdentityWhere[Q]{
		ID:                psql.Where[Q, string](cols.ID.Expression),
		ProjectID:         psql.Where[Q, string](cols.ProjectID.Expression),
		Environment:       psql.Where[Q, string](cols.Environment.Expression),
		UserID:            psql.Where[Q, string](cols.UserID.Expression),
		Type:              psql.Where[Q, string](cols.Type.Expression),
		Provider:          psql.WhereNull[Q, string](cols.Provider.Expression),
		ProviderAccountID: psql.WhereNull[Q, string](cols.ProviderAccountID.Expression),
		Email:             psql.WhereNull[Q, string](cols.Email.Expression),
		CreatedAt:         psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:              psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
