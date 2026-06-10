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

// IamWebauthnCredential is an object representing the database table.
type IamWebauthnCredential struct {
	ID           string              `db:"id,pk" `
	ProjectID    string              `db:"project_id" `
	Environment  string              `db:"environment" `
	UserID       string              `db:"user_id" `
	CredentialID string              `db:"credential_id" `
	PublicKey    null.Val[[]byte]    `db:"public_key" `
	SignCount    int64               `db:"sign_count" `
	CreatedAt    time.Time           `db:"created_at" `
	LastUsedAt   null.Val[time.Time] `db:"last_used_at" `
	Data         json.RawMessage     `db:"data" `
}

// IamWebauthnCredentialSlice is an alias for a slice of pointers to IamWebauthnCredential.
// This should almost always be used instead of []*IamWebauthnCredential.
type IamWebauthnCredentialSlice []*IamWebauthnCredential

// IamWebauthnCredentials contains methods to work with the iam_webauthn_credentials table
var IamWebauthnCredentials = psql.NewTablex[*IamWebauthnCredential, IamWebauthnCredentialSlice, *IamWebauthnCredentialSetter]("", "iam_webauthn_credentials", buildIamWebauthnCredentialColumns("iam_webauthn_credentials"))

// IamWebauthnCredentialsQuery is a query on the iam_webauthn_credentials table
type IamWebauthnCredentialsQuery = *psql.ViewQuery[*IamWebauthnCredential, IamWebauthnCredentialSlice]

func buildIamWebauthnCredentialColumns(tableName string) iamWebauthnCredentialColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "user_id", "credential_id", "public_key", "sign_count", "created_at", "last_used_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamWebauthnCredentialColumns{
		ColumnsExpr:  columnsExpr,
		tableAlias:   tableName,
		ID:           buildIamWebauthnCredentialColumn(tableName, "id"),
		ProjectID:    buildIamWebauthnCredentialColumn(tableName, "project_id"),
		Environment:  buildIamWebauthnCredentialColumn(tableName, "environment"),
		UserID:       buildIamWebauthnCredentialColumn(tableName, "user_id"),
		CredentialID: buildIamWebauthnCredentialColumn(tableName, "credential_id"),
		PublicKey:    buildIamWebauthnCredentialColumn(tableName, "public_key"),
		SignCount:    buildIamWebauthnCredentialColumn(tableName, "sign_count"),
		CreatedAt:    buildIamWebauthnCredentialColumn(tableName, "created_at"),
		LastUsedAt:   buildIamWebauthnCredentialColumn(tableName, "last_used_at"),
		Data:         buildIamWebauthnCredentialColumn(tableName, "data"),
	}
}

type iamWebauthnCredentialColumns struct {
	expr.ColumnsExpr
	tableAlias   string
	ID           iamWebauthnCredentialColumn
	ProjectID    iamWebauthnCredentialColumn
	Environment  iamWebauthnCredentialColumn
	UserID       iamWebauthnCredentialColumn
	CredentialID iamWebauthnCredentialColumn
	PublicKey    iamWebauthnCredentialColumn
	SignCount    iamWebauthnCredentialColumn
	CreatedAt    iamWebauthnCredentialColumn
	LastUsedAt   iamWebauthnCredentialColumn
	Data         iamWebauthnCredentialColumn
}

// Alias returns the current table alias for the columns set.
func (c iamWebauthnCredentialColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamWebauthnCredentialColumns) AliasedAs(tableName string) iamWebauthnCredentialColumns {
	return buildIamWebauthnCredentialColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamWebauthnCredentialColumns) Unqualified() iamWebauthnCredentialColumns {
	return buildIamWebauthnCredentialColumns("")
}

func buildIamWebauthnCredentialColumn(alias, name string) iamWebauthnCredentialColumn {
	return iamWebauthnCredentialColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamWebauthnCredentialColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamWebauthnCredentialColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamWebauthnCredentialColumn) ShouldOmitParens() bool {
	return true
}

// IamWebauthnCredentialSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamWebauthnCredentialSetter struct {
	ID           *string              `db:"id,pk" `
	ProjectID    *string              `db:"project_id" `
	Environment  *string              `db:"environment" `
	UserID       *string              `db:"user_id" `
	CredentialID *string              `db:"credential_id" `
	PublicKey    *null.Val[[]byte]    `db:"public_key" `
	SignCount    *int64               `db:"sign_count" `
	CreatedAt    *time.Time           `db:"created_at" `
	LastUsedAt   *null.Val[time.Time] `db:"last_used_at" `
	Data         *json.RawMessage     `db:"data" `
}

func (s IamWebauthnCredentialSetter) SetColumns() []string {
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
	if s.CredentialID != nil {
		vals = append(vals, "credential_id")
	}
	if s.PublicKey != nil {
		vals = append(vals, "public_key")
	}
	if s.SignCount != nil {
		vals = append(vals, "sign_count")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.LastUsedAt != nil {
		vals = append(vals, "last_used_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamWebauthnCredentialSetter) Overwrite(t *IamWebauthnCredential) {
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
	if s.CredentialID != nil {
		t.CredentialID = func() string {
			if s.CredentialID == nil {
				return *new(string)
			}
			return *s.CredentialID
		}()
	}
	if s.PublicKey != nil {
		t.PublicKey = func() null.Val[[]byte] {
			if s.PublicKey == nil {
				return *new(null.Val[[]byte])
			}
			v := s.PublicKey
			return *v
		}()
	}
	if s.SignCount != nil {
		t.SignCount = func() int64 {
			if s.SignCount == nil {
				return *new(int64)
			}
			return *s.SignCount
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
	if s.LastUsedAt != nil {
		t.LastUsedAt = func() null.Val[time.Time] {
			if s.LastUsedAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.LastUsedAt
			return *v
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

func (s *IamWebauthnCredentialSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamWebauthnCredentials.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.CredentialID != nil {
			vals[4] = psql.Arg(func() string {
				if s.CredentialID == nil {
					return *new(string)
				}
				return *s.CredentialID
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.PublicKey != nil {
			vals[5] = psql.Arg(func() null.Val[[]byte] {
				if s.PublicKey == nil {
					return *new(null.Val[[]byte])
				}
				v := s.PublicKey
				return *v
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.SignCount != nil {
			vals[6] = psql.Arg(func() int64 {
				if s.SignCount == nil {
					return *new(int64)
				}
				return *s.SignCount
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[7] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		if s.LastUsedAt != nil {
			vals[8] = psql.Arg(func() null.Val[time.Time] {
				if s.LastUsedAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.LastUsedAt
				return *v
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

func (s IamWebauthnCredentialSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamWebauthnCredentialSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.CredentialID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "credential_id")...),
			psql.Arg(s.CredentialID),
		}})
	}

	if s.PublicKey != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "public_key")...),
			psql.Arg(s.PublicKey),
		}})
	}

	if s.SignCount != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "sign_count")...),
			psql.Arg(s.SignCount),
		}})
	}

	if s.CreatedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "created_at")...),
			psql.Arg(s.CreatedAt),
		}})
	}

	if s.LastUsedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "last_used_at")...),
			psql.Arg(s.LastUsedAt),
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

// FindIamWebauthnCredential retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamWebauthnCredential(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamWebauthnCredential, error) {
	if len(cols) == 0 {
		return IamWebauthnCredentials.Query(
			sm.Where(IamWebauthnCredentials.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamWebauthnCredentials.Query(
		sm.Where(IamWebauthnCredentials.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamWebauthnCredentials.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamWebauthnCredentialExists checks the presence of a single record by primary key
func IamWebauthnCredentialExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamWebauthnCredentials.Query(
		sm.Where(IamWebauthnCredentials.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamWebauthnCredential is retrieved from the database
func (o *IamWebauthnCredential) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamWebauthnCredentials.AfterSelectHooks.RunHooks(ctx, exec, IamWebauthnCredentialSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamWebauthnCredentials.AfterInsertHooks.RunHooks(ctx, exec, IamWebauthnCredentialSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamWebauthnCredentials.AfterUpdateHooks.RunHooks(ctx, exec, IamWebauthnCredentialSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamWebauthnCredentials.AfterDeleteHooks.RunHooks(ctx, exec, IamWebauthnCredentialSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamWebauthnCredentials.AfterMergeHooks.RunHooks(ctx, exec, IamWebauthnCredentialSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamWebauthnCredential
func (o *IamWebauthnCredential) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamWebauthnCredential) pkEQ() dialect.Expression {
	return psql.Quote("iam_webauthn_credentials", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamWebauthnCredential
func (o *IamWebauthnCredential) Update(ctx context.Context, exec bob.Executor, s *IamWebauthnCredentialSetter) error {
	v, err := IamWebauthnCredentials.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamWebauthnCredential record with an executor
func (o *IamWebauthnCredential) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamWebauthnCredentials.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamWebauthnCredential using the executor
func (o *IamWebauthnCredential) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamWebauthnCredentials.Query(
		sm.Where(IamWebauthnCredentials.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamWebauthnCredentialSlice is retrieved from the database
func (o IamWebauthnCredentialSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamWebauthnCredentials.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamWebauthnCredentials.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamWebauthnCredentials.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamWebauthnCredentials.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamWebauthnCredentials.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamWebauthnCredentialSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_webauthn_credentials", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamWebauthnCredentialSlice) copyMatchingRows(from ...*IamWebauthnCredential) {
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
func (o IamWebauthnCredentialSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebauthnCredentials.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebauthnCredential:
				o.copyMatchingRows(retrieved)
			case []*IamWebauthnCredential:
				o.copyMatchingRows(retrieved...)
			case IamWebauthnCredentialSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebauthnCredential or a slice of IamWebauthnCredential
				// then run the AfterUpdateHooks on the slice
				_, err = IamWebauthnCredentials.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamWebauthnCredentialSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebauthnCredentials.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebauthnCredential:
				o.copyMatchingRows(retrieved)
			case []*IamWebauthnCredential:
				o.copyMatchingRows(retrieved...)
			case IamWebauthnCredentialSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebauthnCredential or a slice of IamWebauthnCredential
				// then run the AfterDeleteHooks on the slice
				_, err = IamWebauthnCredentials.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamWebauthnCredentialSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebauthnCredentials.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebauthnCredential:
				o.copyMatchingRows(retrieved)
			case []*IamWebauthnCredential:
				o.copyMatchingRows(retrieved...)
			case IamWebauthnCredentialSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebauthnCredential or a slice of IamWebauthnCredential
				// then run the AfterMergeHooks on the slice
				_, err = IamWebauthnCredentials.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamWebauthnCredentialSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamWebauthnCredentialSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamWebauthnCredentials.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamWebauthnCredentialSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamWebauthnCredentials.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamWebauthnCredentialSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamWebauthnCredentials.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamWebauthnCredentialWhere[Q psql.Filterable] struct {
	ID           psql.WhereMod[Q, string]
	ProjectID    psql.WhereMod[Q, string]
	Environment  psql.WhereMod[Q, string]
	UserID       psql.WhereMod[Q, string]
	CredentialID psql.WhereMod[Q, string]
	PublicKey    psql.WhereNullMod[Q, []byte]
	SignCount    psql.WhereMod[Q, int64]
	CreatedAt    psql.WhereMod[Q, time.Time]
	LastUsedAt   psql.WhereNullMod[Q, time.Time]
	Data         psql.WhereMod[Q, json.RawMessage]
}

func (iamWebauthnCredentialWhere[Q]) AliasedAs(alias string) iamWebauthnCredentialWhere[Q] {
	return buildIamWebauthnCredentialWhere[Q](buildIamWebauthnCredentialColumns(alias))
}

func buildIamWebauthnCredentialWhere[Q psql.Filterable](cols iamWebauthnCredentialColumns) iamWebauthnCredentialWhere[Q] {
	return iamWebauthnCredentialWhere[Q]{
		ID:           psql.Where[Q, string](cols.ID.Expression),
		ProjectID:    psql.Where[Q, string](cols.ProjectID.Expression),
		Environment:  psql.Where[Q, string](cols.Environment.Expression),
		UserID:       psql.Where[Q, string](cols.UserID.Expression),
		CredentialID: psql.Where[Q, string](cols.CredentialID.Expression),
		PublicKey:    psql.WhereNull[Q, []byte](cols.PublicKey.Expression),
		SignCount:    psql.Where[Q, int64](cols.SignCount.Expression),
		CreatedAt:    psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		LastUsedAt:   psql.WhereNull[Q, time.Time](cols.LastUsedAt.Expression),
		Data:         psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
