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

// IamDomain is an object representing the database table.
type IamDomain struct {
	ID           string              `db:"id,pk" `
	ProjectID    string              `db:"project_id" `
	ConnectionID null.Val[string]    `db:"connection_id" `
	Domain       string              `db:"domain" `
	Status       string              `db:"status" `
	VerifiedAt   null.Val[time.Time] `db:"verified_at" `
	CreatedAt    time.Time           `db:"created_at" `
	Data         json.RawMessage     `db:"data" `
}

// IamDomainSlice is an alias for a slice of pointers to IamDomain.
// This should almost always be used instead of []*IamDomain.
type IamDomainSlice []*IamDomain

// IamDomains contains methods to work with the iam_domains table
var IamDomains = psql.NewTablex[*IamDomain, IamDomainSlice, *IamDomainSetter]("", "iam_domains", buildIamDomainColumns("iam_domains"))

// IamDomainsQuery is a query on the iam_domains table
type IamDomainsQuery = *psql.ViewQuery[*IamDomain, IamDomainSlice]

func buildIamDomainColumns(tableName string) iamDomainColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "connection_id", "domain", "status", "verified_at", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamDomainColumns{
		ColumnsExpr:  columnsExpr,
		tableAlias:   tableName,
		ID:           buildIamDomainColumn(tableName, "id"),
		ProjectID:    buildIamDomainColumn(tableName, "project_id"),
		ConnectionID: buildIamDomainColumn(tableName, "connection_id"),
		Domain:       buildIamDomainColumn(tableName, "domain"),
		Status:       buildIamDomainColumn(tableName, "status"),
		VerifiedAt:   buildIamDomainColumn(tableName, "verified_at"),
		CreatedAt:    buildIamDomainColumn(tableName, "created_at"),
		Data:         buildIamDomainColumn(tableName, "data"),
	}
}

type iamDomainColumns struct {
	expr.ColumnsExpr
	tableAlias   string
	ID           iamDomainColumn
	ProjectID    iamDomainColumn
	ConnectionID iamDomainColumn
	Domain       iamDomainColumn
	Status       iamDomainColumn
	VerifiedAt   iamDomainColumn
	CreatedAt    iamDomainColumn
	Data         iamDomainColumn
}

// Alias returns the current table alias for the columns set.
func (c iamDomainColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamDomainColumns) AliasedAs(tableName string) iamDomainColumns {
	return buildIamDomainColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamDomainColumns) Unqualified() iamDomainColumns {
	return buildIamDomainColumns("")
}

func buildIamDomainColumn(alias, name string) iamDomainColumn {
	return iamDomainColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamDomainColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamDomainColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamDomainColumn) ShouldOmitParens() bool {
	return true
}

// IamDomainSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamDomainSetter struct {
	ID           *string              `db:"id,pk" `
	ProjectID    *string              `db:"project_id" `
	ConnectionID *null.Val[string]    `db:"connection_id" `
	Domain       *string              `db:"domain" `
	Status       *string              `db:"status" `
	VerifiedAt   *null.Val[time.Time] `db:"verified_at" `
	CreatedAt    *time.Time           `db:"created_at" `
	Data         *json.RawMessage     `db:"data" `
}

func (s IamDomainSetter) SetColumns() []string {
	vals := make([]string, 0, 8)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.ConnectionID != nil {
		vals = append(vals, "connection_id")
	}
	if s.Domain != nil {
		vals = append(vals, "domain")
	}
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.VerifiedAt != nil {
		vals = append(vals, "verified_at")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamDomainSetter) Overwrite(t *IamDomain) {
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
		t.ConnectionID = func() null.Val[string] {
			if s.ConnectionID == nil {
				return *new(null.Val[string])
			}
			v := s.ConnectionID
			return *v
		}()
	}
	if s.Domain != nil {
		t.Domain = func() string {
			if s.Domain == nil {
				return *new(string)
			}
			return *s.Domain
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
	if s.VerifiedAt != nil {
		t.VerifiedAt = func() null.Val[time.Time] {
			if s.VerifiedAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.VerifiedAt
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

func (s *IamDomainSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamDomains.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 8)
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
			vals[2] = psql.Arg(func() null.Val[string] {
				if s.ConnectionID == nil {
					return *new(null.Val[string])
				}
				v := s.ConnectionID
				return *v
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Domain != nil {
			vals[3] = psql.Arg(func() string {
				if s.Domain == nil {
					return *new(string)
				}
				return *s.Domain
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

		if s.VerifiedAt != nil {
			vals[5] = psql.Arg(func() null.Val[time.Time] {
				if s.VerifiedAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.VerifiedAt
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

		if s.Data != nil {
			vals[7] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamDomainSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamDomainSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 8)

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

	if s.Domain != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "domain")...),
			psql.Arg(s.Domain),
		}})
	}

	if s.Status != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "status")...),
			psql.Arg(s.Status),
		}})
	}

	if s.VerifiedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "verified_at")...),
			psql.Arg(s.VerifiedAt),
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

// FindIamDomain retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamDomain(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamDomain, error) {
	if len(cols) == 0 {
		return IamDomains.Query(
			sm.Where(IamDomains.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamDomains.Query(
		sm.Where(IamDomains.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamDomains.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamDomainExists checks the presence of a single record by primary key
func IamDomainExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamDomains.Query(
		sm.Where(IamDomains.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamDomain is retrieved from the database
func (o *IamDomain) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamDomains.AfterSelectHooks.RunHooks(ctx, exec, IamDomainSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamDomains.AfterInsertHooks.RunHooks(ctx, exec, IamDomainSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamDomains.AfterUpdateHooks.RunHooks(ctx, exec, IamDomainSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamDomains.AfterDeleteHooks.RunHooks(ctx, exec, IamDomainSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamDomains.AfterMergeHooks.RunHooks(ctx, exec, IamDomainSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamDomain
func (o *IamDomain) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamDomain) pkEQ() dialect.Expression {
	return psql.Quote("iam_domains", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamDomain
func (o *IamDomain) Update(ctx context.Context, exec bob.Executor, s *IamDomainSetter) error {
	v, err := IamDomains.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamDomain record with an executor
func (o *IamDomain) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamDomains.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamDomain using the executor
func (o *IamDomain) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamDomains.Query(
		sm.Where(IamDomains.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamDomainSlice is retrieved from the database
func (o IamDomainSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamDomains.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamDomains.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamDomains.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamDomains.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamDomains.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamDomainSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_domains", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamDomainSlice) copyMatchingRows(from ...*IamDomain) {
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
func (o IamDomainSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamDomains.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamDomain:
				o.copyMatchingRows(retrieved)
			case []*IamDomain:
				o.copyMatchingRows(retrieved...)
			case IamDomainSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamDomain or a slice of IamDomain
				// then run the AfterUpdateHooks on the slice
				_, err = IamDomains.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamDomainSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamDomains.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamDomain:
				o.copyMatchingRows(retrieved)
			case []*IamDomain:
				o.copyMatchingRows(retrieved...)
			case IamDomainSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamDomain or a slice of IamDomain
				// then run the AfterDeleteHooks on the slice
				_, err = IamDomains.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamDomainSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamDomains.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamDomain:
				o.copyMatchingRows(retrieved)
			case []*IamDomain:
				o.copyMatchingRows(retrieved...)
			case IamDomainSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamDomain or a slice of IamDomain
				// then run the AfterMergeHooks on the slice
				_, err = IamDomains.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamDomainSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamDomainSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamDomains.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamDomainSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamDomains.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamDomainSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamDomains.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamDomainWhere[Q psql.Filterable] struct {
	ID           psql.WhereMod[Q, string]
	ProjectID    psql.WhereMod[Q, string]
	ConnectionID psql.WhereNullMod[Q, string]
	Domain       psql.WhereMod[Q, string]
	Status       psql.WhereMod[Q, string]
	VerifiedAt   psql.WhereNullMod[Q, time.Time]
	CreatedAt    psql.WhereMod[Q, time.Time]
	Data         psql.WhereMod[Q, json.RawMessage]
}

func (iamDomainWhere[Q]) AliasedAs(alias string) iamDomainWhere[Q] {
	return buildIamDomainWhere[Q](buildIamDomainColumns(alias))
}

func buildIamDomainWhere[Q psql.Filterable](cols iamDomainColumns) iamDomainWhere[Q] {
	return iamDomainWhere[Q]{
		ID:           psql.Where[Q, string](cols.ID.Expression),
		ProjectID:    psql.Where[Q, string](cols.ProjectID.Expression),
		ConnectionID: psql.WhereNull[Q, string](cols.ConnectionID.Expression),
		Domain:       psql.Where[Q, string](cols.Domain.Expression),
		Status:       psql.Where[Q, string](cols.Status.Expression),
		VerifiedAt:   psql.WhereNull[Q, time.Time](cols.VerifiedAt.Expression),
		CreatedAt:    psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:         psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
