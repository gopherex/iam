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

// IamChallenge is an object representing the database table.
type IamChallenge struct {
	ID        string           `db:"id,pk" `
	ProjectID string           `db:"project_id" `
	Type      string           `db:"type" `
	Subject   null.Val[string] `db:"subject" `
	CodeHash  null.Val[string] `db:"code_hash" `
	ExpiresAt time.Time        `db:"expires_at" `
	Consumed  bool             `db:"consumed" `
	CreatedAt time.Time        `db:"created_at" `
	Data      json.RawMessage  `db:"data" `
}

// IamChallengeSlice is an alias for a slice of pointers to IamChallenge.
// This should almost always be used instead of []*IamChallenge.
type IamChallengeSlice []*IamChallenge

// IamChallenges contains methods to work with the iam_challenges table
var IamChallenges = psql.NewTablex[*IamChallenge, IamChallengeSlice, *IamChallengeSetter]("", "iam_challenges", buildIamChallengeColumns("iam_challenges"))

// IamChallengesQuery is a query on the iam_challenges table
type IamChallengesQuery = *psql.ViewQuery[*IamChallenge, IamChallengeSlice]

func buildIamChallengeColumns(tableName string) iamChallengeColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "type", "subject", "code_hash", "expires_at", "consumed", "created_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamChallengeColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamChallengeColumn(tableName, "id"),
		ProjectID:   buildIamChallengeColumn(tableName, "project_id"),
		Type:        buildIamChallengeColumn(tableName, "type"),
		Subject:     buildIamChallengeColumn(tableName, "subject"),
		CodeHash:    buildIamChallengeColumn(tableName, "code_hash"),
		ExpiresAt:   buildIamChallengeColumn(tableName, "expires_at"),
		Consumed:    buildIamChallengeColumn(tableName, "consumed"),
		CreatedAt:   buildIamChallengeColumn(tableName, "created_at"),
		Data:        buildIamChallengeColumn(tableName, "data"),
	}
}

type iamChallengeColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamChallengeColumn
	ProjectID  iamChallengeColumn
	Type       iamChallengeColumn
	Subject    iamChallengeColumn
	CodeHash   iamChallengeColumn
	ExpiresAt  iamChallengeColumn
	Consumed   iamChallengeColumn
	CreatedAt  iamChallengeColumn
	Data       iamChallengeColumn
}

// Alias returns the current table alias for the columns set.
func (c iamChallengeColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamChallengeColumns) AliasedAs(tableName string) iamChallengeColumns {
	return buildIamChallengeColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamChallengeColumns) Unqualified() iamChallengeColumns {
	return buildIamChallengeColumns("")
}

func buildIamChallengeColumn(alias, name string) iamChallengeColumn {
	return iamChallengeColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamChallengeColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamChallengeColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamChallengeColumn) ShouldOmitParens() bool {
	return true
}

// IamChallengeSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamChallengeSetter struct {
	ID        *string           `db:"id,pk" `
	ProjectID *string           `db:"project_id" `
	Type      *string           `db:"type" `
	Subject   *null.Val[string] `db:"subject" `
	CodeHash  *null.Val[string] `db:"code_hash" `
	ExpiresAt *time.Time        `db:"expires_at" `
	Consumed  *bool             `db:"consumed" `
	CreatedAt *time.Time        `db:"created_at" `
	Data      *json.RawMessage  `db:"data" `
}

func (s IamChallengeSetter) SetColumns() []string {
	vals := make([]string, 0, 9)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Type != nil {
		vals = append(vals, "type")
	}
	if s.Subject != nil {
		vals = append(vals, "subject")
	}
	if s.CodeHash != nil {
		vals = append(vals, "code_hash")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
	}
	if s.Consumed != nil {
		vals = append(vals, "consumed")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamChallengeSetter) Overwrite(t *IamChallenge) {
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
	if s.Type != nil {
		t.Type = func() string {
			if s.Type == nil {
				return *new(string)
			}
			return *s.Type
		}()
	}
	if s.Subject != nil {
		t.Subject = func() null.Val[string] {
			if s.Subject == nil {
				return *new(null.Val[string])
			}
			v := s.Subject
			return *v
		}()
	}
	if s.CodeHash != nil {
		t.CodeHash = func() null.Val[string] {
			if s.CodeHash == nil {
				return *new(null.Val[string])
			}
			v := s.CodeHash
			return *v
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

func (s *IamChallengeSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamChallenges.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 9)
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

		if s.Type != nil {
			vals[2] = psql.Arg(func() string {
				if s.Type == nil {
					return *new(string)
				}
				return *s.Type
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Subject != nil {
			vals[3] = psql.Arg(func() null.Val[string] {
				if s.Subject == nil {
					return *new(null.Val[string])
				}
				v := s.Subject
				return *v
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.CodeHash != nil {
			vals[4] = psql.Arg(func() null.Val[string] {
				if s.CodeHash == nil {
					return *new(null.Val[string])
				}
				v := s.CodeHash
				return *v
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.ExpiresAt != nil {
			vals[5] = psql.Arg(func() time.Time {
				if s.ExpiresAt == nil {
					return *new(time.Time)
				}
				return *s.ExpiresAt
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		if s.Consumed != nil {
			vals[6] = psql.Arg(func() bool {
				if s.Consumed == nil {
					return *new(bool)
				}
				return *s.Consumed
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

		if s.Data != nil {
			vals[8] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[8] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamChallengeSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamChallengeSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 9)

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

	if s.Type != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "type")...),
			psql.Arg(s.Type),
		}})
	}

	if s.Subject != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "subject")...),
			psql.Arg(s.Subject),
		}})
	}

	if s.CodeHash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "code_hash")...),
			psql.Arg(s.CodeHash),
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

// FindIamChallenge retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamChallenge(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamChallenge, error) {
	if len(cols) == 0 {
		return IamChallenges.Query(
			sm.Where(IamChallenges.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamChallenges.Query(
		sm.Where(IamChallenges.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamChallenges.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamChallengeExists checks the presence of a single record by primary key
func IamChallengeExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamChallenges.Query(
		sm.Where(IamChallenges.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamChallenge is retrieved from the database
func (o *IamChallenge) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamChallenges.AfterSelectHooks.RunHooks(ctx, exec, IamChallengeSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamChallenges.AfterInsertHooks.RunHooks(ctx, exec, IamChallengeSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamChallenges.AfterUpdateHooks.RunHooks(ctx, exec, IamChallengeSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamChallenges.AfterDeleteHooks.RunHooks(ctx, exec, IamChallengeSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamChallenges.AfterMergeHooks.RunHooks(ctx, exec, IamChallengeSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamChallenge
func (o *IamChallenge) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamChallenge) pkEQ() dialect.Expression {
	return psql.Quote("iam_challenges", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamChallenge
func (o *IamChallenge) Update(ctx context.Context, exec bob.Executor, s *IamChallengeSetter) error {
	v, err := IamChallenges.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamChallenge record with an executor
func (o *IamChallenge) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamChallenges.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamChallenge using the executor
func (o *IamChallenge) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamChallenges.Query(
		sm.Where(IamChallenges.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamChallengeSlice is retrieved from the database
func (o IamChallengeSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamChallenges.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamChallenges.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamChallenges.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamChallenges.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamChallenges.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamChallengeSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_challenges", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamChallengeSlice) copyMatchingRows(from ...*IamChallenge) {
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
func (o IamChallengeSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamChallenges.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamChallenge:
				o.copyMatchingRows(retrieved)
			case []*IamChallenge:
				o.copyMatchingRows(retrieved...)
			case IamChallengeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamChallenge or a slice of IamChallenge
				// then run the AfterUpdateHooks on the slice
				_, err = IamChallenges.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamChallengeSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamChallenges.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamChallenge:
				o.copyMatchingRows(retrieved)
			case []*IamChallenge:
				o.copyMatchingRows(retrieved...)
			case IamChallengeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamChallenge or a slice of IamChallenge
				// then run the AfterDeleteHooks on the slice
				_, err = IamChallenges.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamChallengeSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamChallenges.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamChallenge:
				o.copyMatchingRows(retrieved)
			case []*IamChallenge:
				o.copyMatchingRows(retrieved...)
			case IamChallengeSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamChallenge or a slice of IamChallenge
				// then run the AfterMergeHooks on the slice
				_, err = IamChallenges.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamChallengeSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamChallengeSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamChallenges.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamChallengeSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamChallenges.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamChallengeSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamChallenges.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamChallengeWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Type      psql.WhereMod[Q, string]
	Subject   psql.WhereNullMod[Q, string]
	CodeHash  psql.WhereNullMod[Q, string]
	ExpiresAt psql.WhereMod[Q, time.Time]
	Consumed  psql.WhereMod[Q, bool]
	CreatedAt psql.WhereMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamChallengeWhere[Q]) AliasedAs(alias string) iamChallengeWhere[Q] {
	return buildIamChallengeWhere[Q](buildIamChallengeColumns(alias))
}

func buildIamChallengeWhere[Q psql.Filterable](cols iamChallengeColumns) iamChallengeWhere[Q] {
	return iamChallengeWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Type:      psql.Where[Q, string](cols.Type.Expression),
		Subject:   psql.WhereNull[Q, string](cols.Subject.Expression),
		CodeHash:  psql.WhereNull[Q, string](cols.CodeHash.Expression),
		ExpiresAt: psql.Where[Q, time.Time](cols.ExpiresAt.Expression),
		Consumed:  psql.Where[Q, bool](cols.Consumed.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
