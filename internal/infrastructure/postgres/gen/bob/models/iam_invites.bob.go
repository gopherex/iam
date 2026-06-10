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

// IamInvite is an object representing the database table.
type IamInvite struct {
	ID          string              `db:"id,pk" `
	ProjectID   string              `db:"project_id" `
	Environment string              `db:"environment" `
	Email       null.Val[string]    `db:"email" `
	TokenHash   string              `db:"token_hash" `
	Status      string              `db:"status" `
	ExpiresAt   null.Val[time.Time] `db:"expires_at" `
	AcceptedAt  null.Val[time.Time] `db:"accepted_at" `
	CreatedAt   time.Time           `db:"created_at" `
	UpdatedAt   time.Time           `db:"updated_at" `
	Data        json.RawMessage     `db:"data" `
}

// IamInviteSlice is an alias for a slice of pointers to IamInvite.
// This should almost always be used instead of []*IamInvite.
type IamInviteSlice []*IamInvite

// IamInvites contains methods to work with the iam_invites table
var IamInvites = psql.NewTablex[*IamInvite, IamInviteSlice, *IamInviteSetter]("", "iam_invites", buildIamInviteColumns("iam_invites"))

// IamInvitesQuery is a query on the iam_invites table
type IamInvitesQuery = *psql.ViewQuery[*IamInvite, IamInviteSlice]

func buildIamInviteColumns(tableName string) iamInviteColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "email", "token_hash", "status", "expires_at", "accepted_at", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamInviteColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamInviteColumn(tableName, "id"),
		ProjectID:   buildIamInviteColumn(tableName, "project_id"),
		Environment: buildIamInviteColumn(tableName, "environment"),
		Email:       buildIamInviteColumn(tableName, "email"),
		TokenHash:   buildIamInviteColumn(tableName, "token_hash"),
		Status:      buildIamInviteColumn(tableName, "status"),
		ExpiresAt:   buildIamInviteColumn(tableName, "expires_at"),
		AcceptedAt:  buildIamInviteColumn(tableName, "accepted_at"),
		CreatedAt:   buildIamInviteColumn(tableName, "created_at"),
		UpdatedAt:   buildIamInviteColumn(tableName, "updated_at"),
		Data:        buildIamInviteColumn(tableName, "data"),
	}
}

type iamInviteColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ID          iamInviteColumn
	ProjectID   iamInviteColumn
	Environment iamInviteColumn
	Email       iamInviteColumn
	TokenHash   iamInviteColumn
	Status      iamInviteColumn
	ExpiresAt   iamInviteColumn
	AcceptedAt  iamInviteColumn
	CreatedAt   iamInviteColumn
	UpdatedAt   iamInviteColumn
	Data        iamInviteColumn
}

// Alias returns the current table alias for the columns set.
func (c iamInviteColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamInviteColumns) AliasedAs(tableName string) iamInviteColumns {
	return buildIamInviteColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamInviteColumns) Unqualified() iamInviteColumns {
	return buildIamInviteColumns("")
}

func buildIamInviteColumn(alias, name string) iamInviteColumn {
	return iamInviteColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamInviteColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamInviteColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamInviteColumn) ShouldOmitParens() bool {
	return true
}

// IamInviteSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamInviteSetter struct {
	ID          *string              `db:"id,pk" `
	ProjectID   *string              `db:"project_id" `
	Environment *string              `db:"environment" `
	Email       *null.Val[string]    `db:"email" `
	TokenHash   *string              `db:"token_hash" `
	Status      *string              `db:"status" `
	ExpiresAt   *null.Val[time.Time] `db:"expires_at" `
	AcceptedAt  *null.Val[time.Time] `db:"accepted_at" `
	CreatedAt   *time.Time           `db:"created_at" `
	UpdatedAt   *time.Time           `db:"updated_at" `
	Data        *json.RawMessage     `db:"data" `
}

func (s IamInviteSetter) SetColumns() []string {
	vals := make([]string, 0, 11)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.Email != nil {
		vals = append(vals, "email")
	}
	if s.TokenHash != nil {
		vals = append(vals, "token_hash")
	}
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
	}
	if s.AcceptedAt != nil {
		vals = append(vals, "accepted_at")
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

func (s IamInviteSetter) Overwrite(t *IamInvite) {
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
	if s.Email != nil {
		t.Email = func() null.Val[string] {
			if s.Email == nil {
				return *new(null.Val[string])
			}
			v := s.Email
			return *v
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
	if s.Status != nil {
		t.Status = func() string {
			if s.Status == nil {
				return *new(string)
			}
			return *s.Status
		}()
	}
	if s.ExpiresAt != nil {
		t.ExpiresAt = func() null.Val[time.Time] {
			if s.ExpiresAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.ExpiresAt
			return *v
		}()
	}
	if s.AcceptedAt != nil {
		t.AcceptedAt = func() null.Val[time.Time] {
			if s.AcceptedAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.AcceptedAt
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

func (s *IamInviteSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamInvites.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Email != nil {
			vals[3] = psql.Arg(func() null.Val[string] {
				if s.Email == nil {
					return *new(null.Val[string])
				}
				v := s.Email
				return *v
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.TokenHash != nil {
			vals[4] = psql.Arg(func() string {
				if s.TokenHash == nil {
					return *new(string)
				}
				return *s.TokenHash
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.Status != nil {
			vals[5] = psql.Arg(func() string {
				if s.Status == nil {
					return *new(string)
				}
				return *s.Status
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[6] = psql.Arg(func() null.Val[time.Time] {
				if s.ExpiresAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.ExpiresAt
				return *v
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.AcceptedAt != nil {
			vals[7] = psql.Arg(func() null.Val[time.Time] {
				if s.AcceptedAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.AcceptedAt
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

func (s IamInviteSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamInviteSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Environment != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "environment")...),
			psql.Arg(s.Environment),
		}})
	}

	if s.Email != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "email")...),
			psql.Arg(s.Email),
		}})
	}

	if s.TokenHash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "token_hash")...),
			psql.Arg(s.TokenHash),
		}})
	}

	if s.Status != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "status")...),
			psql.Arg(s.Status),
		}})
	}

	if s.ExpiresAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "expires_at")...),
			psql.Arg(s.ExpiresAt),
		}})
	}

	if s.AcceptedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "accepted_at")...),
			psql.Arg(s.AcceptedAt),
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

// FindIamInvite retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamInvite(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamInvite, error) {
	if len(cols) == 0 {
		return IamInvites.Query(
			sm.Where(IamInvites.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamInvites.Query(
		sm.Where(IamInvites.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamInvites.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamInviteExists checks the presence of a single record by primary key
func IamInviteExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamInvites.Query(
		sm.Where(IamInvites.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamInvite is retrieved from the database
func (o *IamInvite) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamInvites.AfterSelectHooks.RunHooks(ctx, exec, IamInviteSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamInvites.AfterInsertHooks.RunHooks(ctx, exec, IamInviteSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamInvites.AfterUpdateHooks.RunHooks(ctx, exec, IamInviteSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamInvites.AfterDeleteHooks.RunHooks(ctx, exec, IamInviteSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamInvites.AfterMergeHooks.RunHooks(ctx, exec, IamInviteSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamInvite
func (o *IamInvite) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamInvite) pkEQ() dialect.Expression {
	return psql.Quote("iam_invites", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamInvite
func (o *IamInvite) Update(ctx context.Context, exec bob.Executor, s *IamInviteSetter) error {
	v, err := IamInvites.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamInvite record with an executor
func (o *IamInvite) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamInvites.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamInvite using the executor
func (o *IamInvite) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamInvites.Query(
		sm.Where(IamInvites.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamInviteSlice is retrieved from the database
func (o IamInviteSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamInvites.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamInvites.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamInvites.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamInvites.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamInvites.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamInviteSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_invites", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamInviteSlice) copyMatchingRows(from ...*IamInvite) {
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
func (o IamInviteSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamInvites.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamInvite:
				o.copyMatchingRows(retrieved)
			case []*IamInvite:
				o.copyMatchingRows(retrieved...)
			case IamInviteSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamInvite or a slice of IamInvite
				// then run the AfterUpdateHooks on the slice
				_, err = IamInvites.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamInviteSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamInvites.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamInvite:
				o.copyMatchingRows(retrieved)
			case []*IamInvite:
				o.copyMatchingRows(retrieved...)
			case IamInviteSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamInvite or a slice of IamInvite
				// then run the AfterDeleteHooks on the slice
				_, err = IamInvites.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamInviteSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamInvites.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamInvite:
				o.copyMatchingRows(retrieved)
			case []*IamInvite:
				o.copyMatchingRows(retrieved...)
			case IamInviteSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamInvite or a slice of IamInvite
				// then run the AfterMergeHooks on the slice
				_, err = IamInvites.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamInviteSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamInviteSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamInvites.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamInviteSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamInvites.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamInviteSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamInvites.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamInviteWhere[Q psql.Filterable] struct {
	ID          psql.WhereMod[Q, string]
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	Email       psql.WhereNullMod[Q, string]
	TokenHash   psql.WhereMod[Q, string]
	Status      psql.WhereMod[Q, string]
	ExpiresAt   psql.WhereNullMod[Q, time.Time]
	AcceptedAt  psql.WhereNullMod[Q, time.Time]
	CreatedAt   psql.WhereMod[Q, time.Time]
	UpdatedAt   psql.WhereMod[Q, time.Time]
	Data        psql.WhereMod[Q, json.RawMessage]
}

func (iamInviteWhere[Q]) AliasedAs(alias string) iamInviteWhere[Q] {
	return buildIamInviteWhere[Q](buildIamInviteColumns(alias))
}

func buildIamInviteWhere[Q psql.Filterable](cols iamInviteColumns) iamInviteWhere[Q] {
	return iamInviteWhere[Q]{
		ID:          psql.Where[Q, string](cols.ID.Expression),
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		Email:       psql.WhereNull[Q, string](cols.Email.Expression),
		TokenHash:   psql.Where[Q, string](cols.TokenHash.Expression),
		Status:      psql.Where[Q, string](cols.Status.Expression),
		ExpiresAt:   psql.WhereNull[Q, time.Time](cols.ExpiresAt.Expression),
		AcceptedAt:  psql.WhereNull[Q, time.Time](cols.AcceptedAt.Expression),
		CreatedAt:   psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt:   psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:        psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
