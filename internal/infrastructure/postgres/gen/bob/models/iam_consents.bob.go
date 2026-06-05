// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
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

// IamConsent is an object representing the database table.
type IamConsent struct {
	ID         string           `db:"id,pk" `
	ProjectID  string           `db:"project_id" `
	UserID     string           `db:"user_id" `
	DocKey     string           `db:"doc_key" `
	Version    string           `db:"version" `
	Locale     null.Val[string] `db:"locale" `
	AcceptedAt time.Time        `db:"accepted_at" `
}

// IamConsentSlice is an alias for a slice of pointers to IamConsent.
// This should almost always be used instead of []*IamConsent.
type IamConsentSlice []*IamConsent

// IamConsents contains methods to work with the iam_consents table
var IamConsents = psql.NewTablex[*IamConsent, IamConsentSlice, *IamConsentSetter]("", "iam_consents", buildIamConsentColumns("iam_consents"))

// IamConsentsQuery is a query on the iam_consents table
type IamConsentsQuery = *psql.ViewQuery[*IamConsent, IamConsentSlice]

func buildIamConsentColumns(tableName string) iamConsentColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "user_id", "doc_key", "version", "locale", "accepted_at",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamConsentColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamConsentColumn(tableName, "id"),
		ProjectID:   buildIamConsentColumn(tableName, "project_id"),
		UserID:      buildIamConsentColumn(tableName, "user_id"),
		DocKey:      buildIamConsentColumn(tableName, "doc_key"),
		Version:     buildIamConsentColumn(tableName, "version"),
		Locale:      buildIamConsentColumn(tableName, "locale"),
		AcceptedAt:  buildIamConsentColumn(tableName, "accepted_at"),
	}
}

type iamConsentColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamConsentColumn
	ProjectID  iamConsentColumn
	UserID     iamConsentColumn
	DocKey     iamConsentColumn
	Version    iamConsentColumn
	Locale     iamConsentColumn
	AcceptedAt iamConsentColumn
}

// Alias returns the current table alias for the columns set.
func (c iamConsentColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamConsentColumns) AliasedAs(tableName string) iamConsentColumns {
	return buildIamConsentColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamConsentColumns) Unqualified() iamConsentColumns {
	return buildIamConsentColumns("")
}

func buildIamConsentColumn(alias, name string) iamConsentColumn {
	return iamConsentColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamConsentColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamConsentColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamConsentColumn) ShouldOmitParens() bool {
	return true
}

// IamConsentSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamConsentSetter struct {
	ID         *string           `db:"id,pk" `
	ProjectID  *string           `db:"project_id" `
	UserID     *string           `db:"user_id" `
	DocKey     *string           `db:"doc_key" `
	Version    *string           `db:"version" `
	Locale     *null.Val[string] `db:"locale" `
	AcceptedAt *time.Time        `db:"accepted_at" `
}

func (s IamConsentSetter) SetColumns() []string {
	vals := make([]string, 0, 7)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.UserID != nil {
		vals = append(vals, "user_id")
	}
	if s.DocKey != nil {
		vals = append(vals, "doc_key")
	}
	if s.Version != nil {
		vals = append(vals, "version")
	}
	if s.Locale != nil {
		vals = append(vals, "locale")
	}
	if s.AcceptedAt != nil {
		vals = append(vals, "accepted_at")
	}
	return vals
}

func (s IamConsentSetter) Overwrite(t *IamConsent) {
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
	if s.UserID != nil {
		t.UserID = func() string {
			if s.UserID == nil {
				return *new(string)
			}
			return *s.UserID
		}()
	}
	if s.DocKey != nil {
		t.DocKey = func() string {
			if s.DocKey == nil {
				return *new(string)
			}
			return *s.DocKey
		}()
	}
	if s.Version != nil {
		t.Version = func() string {
			if s.Version == nil {
				return *new(string)
			}
			return *s.Version
		}()
	}
	if s.Locale != nil {
		t.Locale = func() null.Val[string] {
			if s.Locale == nil {
				return *new(null.Val[string])
			}
			v := s.Locale
			return *v
		}()
	}
	if s.AcceptedAt != nil {
		t.AcceptedAt = func() time.Time {
			if s.AcceptedAt == nil {
				return *new(time.Time)
			}
			return *s.AcceptedAt
		}()
	}
}

func (s *IamConsentSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamConsents.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.DocKey != nil {
			vals[3] = psql.Arg(func() string {
				if s.DocKey == nil {
					return *new(string)
				}
				return *s.DocKey
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.Version != nil {
			vals[4] = psql.Arg(func() string {
				if s.Version == nil {
					return *new(string)
				}
				return *s.Version
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.Locale != nil {
			vals[5] = psql.Arg(func() null.Val[string] {
				if s.Locale == nil {
					return *new(null.Val[string])
				}
				v := s.Locale
				return *v
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.AcceptedAt != nil {
			vals[6] = psql.Arg(func() time.Time {
				if s.AcceptedAt == nil {
					return *new(time.Time)
				}
				return *s.AcceptedAt
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamConsentSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamConsentSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.UserID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "user_id")...),
			psql.Arg(s.UserID),
		}})
	}

	if s.DocKey != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "doc_key")...),
			psql.Arg(s.DocKey),
		}})
	}

	if s.Version != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "version")...),
			psql.Arg(s.Version),
		}})
	}

	if s.Locale != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "locale")...),
			psql.Arg(s.Locale),
		}})
	}

	if s.AcceptedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "accepted_at")...),
			psql.Arg(s.AcceptedAt),
		}})
	}

	return exprs
}

// FindIamConsent retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamConsent(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamConsent, error) {
	if len(cols) == 0 {
		return IamConsents.Query(
			sm.Where(IamConsents.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamConsents.Query(
		sm.Where(IamConsents.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamConsents.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamConsentExists checks the presence of a single record by primary key
func IamConsentExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamConsents.Query(
		sm.Where(IamConsents.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamConsent is retrieved from the database
func (o *IamConsent) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamConsents.AfterSelectHooks.RunHooks(ctx, exec, IamConsentSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamConsents.AfterInsertHooks.RunHooks(ctx, exec, IamConsentSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamConsents.AfterUpdateHooks.RunHooks(ctx, exec, IamConsentSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamConsents.AfterDeleteHooks.RunHooks(ctx, exec, IamConsentSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamConsents.AfterMergeHooks.RunHooks(ctx, exec, IamConsentSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamConsent
func (o *IamConsent) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamConsent) pkEQ() dialect.Expression {
	return psql.Quote("iam_consents", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamConsent
func (o *IamConsent) Update(ctx context.Context, exec bob.Executor, s *IamConsentSetter) error {
	v, err := IamConsents.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamConsent record with an executor
func (o *IamConsent) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamConsents.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamConsent using the executor
func (o *IamConsent) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamConsents.Query(
		sm.Where(IamConsents.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamConsentSlice is retrieved from the database
func (o IamConsentSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamConsents.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamConsents.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamConsents.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamConsents.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamConsents.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamConsentSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_consents", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamConsentSlice) copyMatchingRows(from ...*IamConsent) {
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
func (o IamConsentSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamConsents.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamConsent:
				o.copyMatchingRows(retrieved)
			case []*IamConsent:
				o.copyMatchingRows(retrieved...)
			case IamConsentSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamConsent or a slice of IamConsent
				// then run the AfterUpdateHooks on the slice
				_, err = IamConsents.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamConsentSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamConsents.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamConsent:
				o.copyMatchingRows(retrieved)
			case []*IamConsent:
				o.copyMatchingRows(retrieved...)
			case IamConsentSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamConsent or a slice of IamConsent
				// then run the AfterDeleteHooks on the slice
				_, err = IamConsents.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamConsentSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamConsents.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamConsent:
				o.copyMatchingRows(retrieved)
			case []*IamConsent:
				o.copyMatchingRows(retrieved...)
			case IamConsentSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamConsent or a slice of IamConsent
				// then run the AfterMergeHooks on the slice
				_, err = IamConsents.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamConsentSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamConsentSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamConsents.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamConsentSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamConsents.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamConsentSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamConsents.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamConsentWhere[Q psql.Filterable] struct {
	ID         psql.WhereMod[Q, string]
	ProjectID  psql.WhereMod[Q, string]
	UserID     psql.WhereMod[Q, string]
	DocKey     psql.WhereMod[Q, string]
	Version    psql.WhereMod[Q, string]
	Locale     psql.WhereNullMod[Q, string]
	AcceptedAt psql.WhereMod[Q, time.Time]
}

func (iamConsentWhere[Q]) AliasedAs(alias string) iamConsentWhere[Q] {
	return buildIamConsentWhere[Q](buildIamConsentColumns(alias))
}

func buildIamConsentWhere[Q psql.Filterable](cols iamConsentColumns) iamConsentWhere[Q] {
	return iamConsentWhere[Q]{
		ID:         psql.Where[Q, string](cols.ID.Expression),
		ProjectID:  psql.Where[Q, string](cols.ProjectID.Expression),
		UserID:     psql.Where[Q, string](cols.UserID.Expression),
		DocKey:     psql.Where[Q, string](cols.DocKey.Expression),
		Version:    psql.Where[Q, string](cols.Version.Expression),
		Locale:     psql.WhereNull[Q, string](cols.Locale.Expression),
		AcceptedAt: psql.Where[Q, time.Time](cols.AcceptedAt.Expression),
	}
}
