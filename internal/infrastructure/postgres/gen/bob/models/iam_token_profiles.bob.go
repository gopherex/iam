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

// IamTokenProfile is an object representing the database table.
type IamTokenProfile struct {
	ID        string          `db:"id,pk" `
	ProjectID string          `db:"project_id" `
	Name      string          `db:"name" `
	CreatedAt time.Time       `db:"created_at" `
	UpdatedAt time.Time       `db:"updated_at" `
	Data      json.RawMessage `db:"data" `
}

// IamTokenProfileSlice is an alias for a slice of pointers to IamTokenProfile.
// This should almost always be used instead of []*IamTokenProfile.
type IamTokenProfileSlice []*IamTokenProfile

// IamTokenProfiles contains methods to work with the iam_token_profiles table
var IamTokenProfiles = psql.NewTablex[*IamTokenProfile, IamTokenProfileSlice, *IamTokenProfileSetter]("", "iam_token_profiles", buildIamTokenProfileColumns("iam_token_profiles"))

// IamTokenProfilesQuery is a query on the iam_token_profiles table
type IamTokenProfilesQuery = *psql.ViewQuery[*IamTokenProfile, IamTokenProfileSlice]

func buildIamTokenProfileColumns(tableName string) iamTokenProfileColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "name", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamTokenProfileColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamTokenProfileColumn(tableName, "id"),
		ProjectID:   buildIamTokenProfileColumn(tableName, "project_id"),
		Name:        buildIamTokenProfileColumn(tableName, "name"),
		CreatedAt:   buildIamTokenProfileColumn(tableName, "created_at"),
		UpdatedAt:   buildIamTokenProfileColumn(tableName, "updated_at"),
		Data:        buildIamTokenProfileColumn(tableName, "data"),
	}
}

type iamTokenProfileColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamTokenProfileColumn
	ProjectID  iamTokenProfileColumn
	Name       iamTokenProfileColumn
	CreatedAt  iamTokenProfileColumn
	UpdatedAt  iamTokenProfileColumn
	Data       iamTokenProfileColumn
}

// Alias returns the current table alias for the columns set.
func (c iamTokenProfileColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamTokenProfileColumns) AliasedAs(tableName string) iamTokenProfileColumns {
	return buildIamTokenProfileColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamTokenProfileColumns) Unqualified() iamTokenProfileColumns {
	return buildIamTokenProfileColumns("")
}

func buildIamTokenProfileColumn(alias, name string) iamTokenProfileColumn {
	return iamTokenProfileColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamTokenProfileColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamTokenProfileColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamTokenProfileColumn) ShouldOmitParens() bool {
	return true
}

// IamTokenProfileSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamTokenProfileSetter struct {
	ID        *string          `db:"id,pk" `
	ProjectID *string          `db:"project_id" `
	Name      *string          `db:"name" `
	CreatedAt *time.Time       `db:"created_at" `
	UpdatedAt *time.Time       `db:"updated_at" `
	Data      *json.RawMessage `db:"data" `
}

func (s IamTokenProfileSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Name != nil {
		vals = append(vals, "name")
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

func (s IamTokenProfileSetter) Overwrite(t *IamTokenProfile) {
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
	if s.Name != nil {
		t.Name = func() string {
			if s.Name == nil {
				return *new(string)
			}
			return *s.Name
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

func (s *IamTokenProfileSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamTokenProfiles.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Name != nil {
			vals[2] = psql.Arg(func() string {
				if s.Name == nil {
					return *new(string)
				}
				return *s.Name
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

func (s IamTokenProfileSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamTokenProfileSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Name != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "name")...),
			psql.Arg(s.Name),
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

// FindIamTokenProfile retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamTokenProfile(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamTokenProfile, error) {
	if len(cols) == 0 {
		return IamTokenProfiles.Query(
			sm.Where(IamTokenProfiles.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamTokenProfiles.Query(
		sm.Where(IamTokenProfiles.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamTokenProfiles.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamTokenProfileExists checks the presence of a single record by primary key
func IamTokenProfileExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamTokenProfiles.Query(
		sm.Where(IamTokenProfiles.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamTokenProfile is retrieved from the database
func (o *IamTokenProfile) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamTokenProfiles.AfterSelectHooks.RunHooks(ctx, exec, IamTokenProfileSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamTokenProfiles.AfterInsertHooks.RunHooks(ctx, exec, IamTokenProfileSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamTokenProfiles.AfterUpdateHooks.RunHooks(ctx, exec, IamTokenProfileSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamTokenProfiles.AfterDeleteHooks.RunHooks(ctx, exec, IamTokenProfileSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamTokenProfiles.AfterMergeHooks.RunHooks(ctx, exec, IamTokenProfileSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamTokenProfile
func (o *IamTokenProfile) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamTokenProfile) pkEQ() dialect.Expression {
	return psql.Quote("iam_token_profiles", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamTokenProfile
func (o *IamTokenProfile) Update(ctx context.Context, exec bob.Executor, s *IamTokenProfileSetter) error {
	v, err := IamTokenProfiles.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamTokenProfile record with an executor
func (o *IamTokenProfile) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamTokenProfiles.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamTokenProfile using the executor
func (o *IamTokenProfile) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamTokenProfiles.Query(
		sm.Where(IamTokenProfiles.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamTokenProfileSlice is retrieved from the database
func (o IamTokenProfileSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamTokenProfiles.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamTokenProfiles.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamTokenProfiles.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamTokenProfiles.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamTokenProfiles.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamTokenProfileSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_token_profiles", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamTokenProfileSlice) copyMatchingRows(from ...*IamTokenProfile) {
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
func (o IamTokenProfileSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamTokenProfiles.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamTokenProfile:
				o.copyMatchingRows(retrieved)
			case []*IamTokenProfile:
				o.copyMatchingRows(retrieved...)
			case IamTokenProfileSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamTokenProfile or a slice of IamTokenProfile
				// then run the AfterUpdateHooks on the slice
				_, err = IamTokenProfiles.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamTokenProfileSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamTokenProfiles.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamTokenProfile:
				o.copyMatchingRows(retrieved)
			case []*IamTokenProfile:
				o.copyMatchingRows(retrieved...)
			case IamTokenProfileSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamTokenProfile or a slice of IamTokenProfile
				// then run the AfterDeleteHooks on the slice
				_, err = IamTokenProfiles.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamTokenProfileSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamTokenProfiles.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamTokenProfile:
				o.copyMatchingRows(retrieved)
			case []*IamTokenProfile:
				o.copyMatchingRows(retrieved...)
			case IamTokenProfileSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamTokenProfile or a slice of IamTokenProfile
				// then run the AfterMergeHooks on the slice
				_, err = IamTokenProfiles.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamTokenProfileSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamTokenProfileSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamTokenProfiles.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamTokenProfileSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamTokenProfiles.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamTokenProfileSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamTokenProfiles.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamTokenProfileWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Name      psql.WhereMod[Q, string]
	CreatedAt psql.WhereMod[Q, time.Time]
	UpdatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamTokenProfileWhere[Q]) AliasedAs(alias string) iamTokenProfileWhere[Q] {
	return buildIamTokenProfileWhere[Q](buildIamTokenProfileColumns(alias))
}

func buildIamTokenProfileWhere[Q psql.Filterable](cols iamTokenProfileColumns) iamTokenProfileWhere[Q] {
	return iamTokenProfileWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Name:      psql.Where[Q, string](cols.Name.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt: psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
