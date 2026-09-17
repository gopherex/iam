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

// IamSecurityContinuation is an object representing the database table.
type IamSecurityContinuation struct {
	ExchangeData string    `db:"exchange_data" `
	TokenHash    string    `db:"token_hash,pk" `
	ProjectID    string    `db:"project_id" `
	Environment  string    `db:"environment" `
	UserID       string    `db:"user_id" `
	IncidentID   string    `db:"incident_id" `
	CaseID       string    `db:"case_id" `
	Purpose      string    `db:"purpose" `
	ExpiresAt    time.Time `db:"expires_at" `
	Consumed     bool      `db:"consumed" `
}

// IamSecurityContinuationSlice is an alias for a slice of pointers to IamSecurityContinuation.
// This should almost always be used instead of []*IamSecurityContinuation.
type IamSecurityContinuationSlice []*IamSecurityContinuation

// IamSecurityContinuations contains methods to work with the iam_security_continuations table
var IamSecurityContinuations = psql.NewTablex[*IamSecurityContinuation, IamSecurityContinuationSlice, *IamSecurityContinuationSetter]("", "iam_security_continuations", buildIamSecurityContinuationColumns("iam_security_continuations"))

// IamSecurityContinuationsQuery is a query on the iam_security_continuations table
type IamSecurityContinuationsQuery = *psql.ViewQuery[*IamSecurityContinuation, IamSecurityContinuationSlice]

func buildIamSecurityContinuationColumns(tableName string) iamSecurityContinuationColumns {
	columnsExpr := expr.NewColumnsExpr(
		"exchange_data", "token_hash", "project_id", "environment", "user_id", "incident_id", "case_id", "purpose", "expires_at", "consumed",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSecurityContinuationColumns{
		ColumnsExpr:  columnsExpr,
		tableAlias:   tableName,
		ExchangeData: buildIamSecurityContinuationColumn(tableName, "exchange_data"),
		TokenHash:    buildIamSecurityContinuationColumn(tableName, "token_hash"),
		ProjectID:    buildIamSecurityContinuationColumn(tableName, "project_id"),
		Environment:  buildIamSecurityContinuationColumn(tableName, "environment"),
		UserID:       buildIamSecurityContinuationColumn(tableName, "user_id"),
		IncidentID:   buildIamSecurityContinuationColumn(tableName, "incident_id"),
		CaseID:       buildIamSecurityContinuationColumn(tableName, "case_id"),
		Purpose:      buildIamSecurityContinuationColumn(tableName, "purpose"),
		ExpiresAt:    buildIamSecurityContinuationColumn(tableName, "expires_at"),
		Consumed:     buildIamSecurityContinuationColumn(tableName, "consumed"),
	}
}

type iamSecurityContinuationColumns struct {
	expr.ColumnsExpr
	tableAlias   string
	ExchangeData iamSecurityContinuationColumn
	TokenHash    iamSecurityContinuationColumn
	ProjectID    iamSecurityContinuationColumn
	Environment  iamSecurityContinuationColumn
	UserID       iamSecurityContinuationColumn
	IncidentID   iamSecurityContinuationColumn
	CaseID       iamSecurityContinuationColumn
	Purpose      iamSecurityContinuationColumn
	ExpiresAt    iamSecurityContinuationColumn
	Consumed     iamSecurityContinuationColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSecurityContinuationColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSecurityContinuationColumns) AliasedAs(tableName string) iamSecurityContinuationColumns {
	return buildIamSecurityContinuationColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSecurityContinuationColumns) Unqualified() iamSecurityContinuationColumns {
	return buildIamSecurityContinuationColumns("")
}

func buildIamSecurityContinuationColumn(alias, name string) iamSecurityContinuationColumn {
	return iamSecurityContinuationColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSecurityContinuationColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSecurityContinuationColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSecurityContinuationColumn) ShouldOmitParens() bool {
	return true
}

// IamSecurityContinuationSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSecurityContinuationSetter struct {
	ExchangeData *string    `db:"exchange_data" `
	TokenHash    *string    `db:"token_hash,pk" `
	ProjectID    *string    `db:"project_id" `
	Environment  *string    `db:"environment" `
	UserID       *string    `db:"user_id" `
	IncidentID   *string    `db:"incident_id" `
	CaseID       *string    `db:"case_id" `
	Purpose      *string    `db:"purpose" `
	ExpiresAt    *time.Time `db:"expires_at" `
	Consumed     *bool      `db:"consumed" `
}

func (s IamSecurityContinuationSetter) SetColumns() []string {
	vals := make([]string, 0, 10)
	if s.ExchangeData != nil {
		vals = append(vals, "exchange_data")
	}
	if s.TokenHash != nil {
		vals = append(vals, "token_hash")
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
	if s.IncidentID != nil {
		vals = append(vals, "incident_id")
	}
	if s.CaseID != nil {
		vals = append(vals, "case_id")
	}
	if s.Purpose != nil {
		vals = append(vals, "purpose")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
	}
	if s.Consumed != nil {
		vals = append(vals, "consumed")
	}
	return vals
}

func (s IamSecurityContinuationSetter) Overwrite(t *IamSecurityContinuation) {
	if s.ExchangeData != nil {
		t.ExchangeData = func() string {
			if s.ExchangeData == nil {
				return *new(string)
			}
			return *s.ExchangeData
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
	if s.IncidentID != nil {
		t.IncidentID = func() string {
			if s.IncidentID == nil {
				return *new(string)
			}
			return *s.IncidentID
		}()
	}
	if s.CaseID != nil {
		t.CaseID = func() string {
			if s.CaseID == nil {
				return *new(string)
			}
			return *s.CaseID
		}()
	}
	if s.Purpose != nil {
		t.Purpose = func() string {
			if s.Purpose == nil {
				return *new(string)
			}
			return *s.Purpose
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
	if s.Consumed != nil {
		t.Consumed = func() bool {
			if s.Consumed == nil {
				return *new(bool)
			}
			return *s.Consumed
		}()
	}
}

func (s *IamSecurityContinuationSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSecurityContinuations.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 10)
		if s.ExchangeData != nil {
			vals[0] = psql.Arg(func() string {
				if s.ExchangeData == nil {
					return *new(string)
				}
				return *s.ExchangeData
			}())
		} else {
			vals[0] = psql.Raw("DEFAULT")
		}

		if s.TokenHash != nil {
			vals[1] = psql.Arg(func() string {
				if s.TokenHash == nil {
					return *new(string)
				}
				return *s.TokenHash
			}())
		} else {
			vals[1] = psql.Raw("DEFAULT")
		}

		if s.ProjectID != nil {
			vals[2] = psql.Arg(func() string {
				if s.ProjectID == nil {
					return *new(string)
				}
				return *s.ProjectID
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Environment != nil {
			vals[3] = psql.Arg(func() string {
				if s.Environment == nil {
					return *new(string)
				}
				return *s.Environment
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.UserID != nil {
			vals[4] = psql.Arg(func() string {
				if s.UserID == nil {
					return *new(string)
				}
				return *s.UserID
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.IncidentID != nil {
			vals[5] = psql.Arg(func() string {
				if s.IncidentID == nil {
					return *new(string)
				}
				return *s.IncidentID
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.CaseID != nil {
			vals[6] = psql.Arg(func() string {
				if s.CaseID == nil {
					return *new(string)
				}
				return *s.CaseID
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.Purpose != nil {
			vals[7] = psql.Arg(func() string {
				if s.Purpose == nil {
					return *new(string)
				}
				return *s.Purpose
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[8] = psql.Arg(func() time.Time {
				if s.ExpiresAt == nil {
					return *new(time.Time)
				}
				return *s.ExpiresAt
			}())
		} else {
			vals[8] = psql.Raw("DEFAULT")
		}

		if s.Consumed != nil {
			vals[9] = psql.Arg(func() bool {
				if s.Consumed == nil {
					return *new(bool)
				}
				return *s.Consumed
			}())
		} else {
			vals[9] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamSecurityContinuationSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSecurityContinuationSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 10)

	if s.ExchangeData != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "exchange_data")...),
			psql.Arg(s.ExchangeData),
		}})
	}

	if s.TokenHash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "token_hash")...),
			psql.Arg(s.TokenHash),
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

	if s.IncidentID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "incident_id")...),
			psql.Arg(s.IncidentID),
		}})
	}

	if s.CaseID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "case_id")...),
			psql.Arg(s.CaseID),
		}})
	}

	if s.Purpose != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "purpose")...),
			psql.Arg(s.Purpose),
		}})
	}

	if s.ExpiresAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "expires_at")...),
			psql.Arg(s.ExpiresAt),
		}})
	}

	if s.Consumed != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "consumed")...),
			psql.Arg(s.Consumed),
		}})
	}

	return exprs
}

// FindIamSecurityContinuation retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSecurityContinuation(ctx context.Context, exec bob.Executor, TokenHashPK string, cols ...string) (*IamSecurityContinuation, error) {
	if len(cols) == 0 {
		return IamSecurityContinuations.Query(
			sm.Where(IamSecurityContinuations.Columns.TokenHash.EQ(psql.Arg(TokenHashPK))),
		).One(ctx, exec)
	}

	return IamSecurityContinuations.Query(
		sm.Where(IamSecurityContinuations.Columns.TokenHash.EQ(psql.Arg(TokenHashPK))),
		sm.Columns(IamSecurityContinuations.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSecurityContinuationExists checks the presence of a single record by primary key
func IamSecurityContinuationExists(ctx context.Context, exec bob.Executor, TokenHashPK string) (bool, error) {
	return IamSecurityContinuations.Query(
		sm.Where(IamSecurityContinuations.Columns.TokenHash.EQ(psql.Arg(TokenHashPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSecurityContinuation is retrieved from the database
func (o *IamSecurityContinuation) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityContinuations.AfterSelectHooks.RunHooks(ctx, exec, IamSecurityContinuationSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityContinuations.AfterInsertHooks.RunHooks(ctx, exec, IamSecurityContinuationSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityContinuations.AfterUpdateHooks.RunHooks(ctx, exec, IamSecurityContinuationSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityContinuations.AfterDeleteHooks.RunHooks(ctx, exec, IamSecurityContinuationSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityContinuations.AfterMergeHooks.RunHooks(ctx, exec, IamSecurityContinuationSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSecurityContinuation
func (o *IamSecurityContinuation) primaryKeyVals() bob.Expression {
	return psql.Arg(o.TokenHash)
}

func (o *IamSecurityContinuation) pkEQ() dialect.Expression {
	return psql.Quote("iam_security_continuations", "token_hash").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSecurityContinuation
func (o *IamSecurityContinuation) Update(ctx context.Context, exec bob.Executor, s *IamSecurityContinuationSetter) error {
	v, err := IamSecurityContinuations.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSecurityContinuation record with an executor
func (o *IamSecurityContinuation) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSecurityContinuations.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSecurityContinuation using the executor
func (o *IamSecurityContinuation) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSecurityContinuations.Query(
		sm.Where(IamSecurityContinuations.Columns.TokenHash.EQ(psql.Arg(o.TokenHash))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSecurityContinuationSlice is retrieved from the database
func (o IamSecurityContinuationSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityContinuations.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityContinuations.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityContinuations.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityContinuations.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityContinuations.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSecurityContinuationSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_security_continuations", "token_hash").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSecurityContinuationSlice) copyMatchingRows(from ...*IamSecurityContinuation) {
	for i, old := range o {
		for _, new := range from {
			if new.TokenHash != old.TokenHash {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamSecurityContinuationSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityContinuations.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityContinuation:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityContinuation:
				o.copyMatchingRows(retrieved...)
			case IamSecurityContinuationSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityContinuation or a slice of IamSecurityContinuation
				// then run the AfterUpdateHooks on the slice
				_, err = IamSecurityContinuations.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSecurityContinuationSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityContinuations.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityContinuation:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityContinuation:
				o.copyMatchingRows(retrieved...)
			case IamSecurityContinuationSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityContinuation or a slice of IamSecurityContinuation
				// then run the AfterDeleteHooks on the slice
				_, err = IamSecurityContinuations.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSecurityContinuationSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityContinuations.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityContinuation:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityContinuation:
				o.copyMatchingRows(retrieved...)
			case IamSecurityContinuationSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityContinuation or a slice of IamSecurityContinuation
				// then run the AfterMergeHooks on the slice
				_, err = IamSecurityContinuations.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSecurityContinuationSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSecurityContinuationSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityContinuations.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSecurityContinuationSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityContinuations.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSecurityContinuationSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSecurityContinuations.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSecurityContinuationWhere[Q psql.Filterable] struct {
	ExchangeData psql.WhereMod[Q, string]
	TokenHash    psql.WhereMod[Q, string]
	ProjectID    psql.WhereMod[Q, string]
	Environment  psql.WhereMod[Q, string]
	UserID       psql.WhereMod[Q, string]
	IncidentID   psql.WhereMod[Q, string]
	CaseID       psql.WhereMod[Q, string]
	Purpose      psql.WhereMod[Q, string]
	ExpiresAt    psql.WhereMod[Q, time.Time]
	Consumed     psql.WhereMod[Q, bool]
}

func (iamSecurityContinuationWhere[Q]) AliasedAs(alias string) iamSecurityContinuationWhere[Q] {
	return buildIamSecurityContinuationWhere[Q](buildIamSecurityContinuationColumns(alias))
}

func buildIamSecurityContinuationWhere[Q psql.Filterable](cols iamSecurityContinuationColumns) iamSecurityContinuationWhere[Q] {
	return iamSecurityContinuationWhere[Q]{
		ExchangeData: psql.Where[Q, string](cols.ExchangeData.Expression),
		TokenHash:    psql.Where[Q, string](cols.TokenHash.Expression),
		ProjectID:    psql.Where[Q, string](cols.ProjectID.Expression),
		Environment:  psql.Where[Q, string](cols.Environment.Expression),
		UserID:       psql.Where[Q, string](cols.UserID.Expression),
		IncidentID:   psql.Where[Q, string](cols.IncidentID.Expression),
		CaseID:       psql.Where[Q, string](cols.CaseID.Expression),
		Purpose:      psql.Where[Q, string](cols.Purpose.Expression),
		ExpiresAt:    psql.Where[Q, time.Time](cols.ExpiresAt.Expression),
		Consumed:     psql.Where[Q, bool](cols.Consumed.Expression),
	}
}
