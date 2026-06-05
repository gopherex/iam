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

// IamBlock is an object representing the database table.
type IamBlock struct {
	ID        string              `db:"id,pk" `
	ProjectID string              `db:"project_id" `
	Subject   string              `db:"subject" `
	CreatedAt time.Time           `db:"created_at" `
	ExpiresAt null.Val[time.Time] `db:"expires_at" `
	Data      json.RawMessage     `db:"data" `
}

// IamBlockSlice is an alias for a slice of pointers to IamBlock.
// This should almost always be used instead of []*IamBlock.
type IamBlockSlice []*IamBlock

// IamBlocks contains methods to work with the iam_blocks table
var IamBlocks = psql.NewTablex[*IamBlock, IamBlockSlice, *IamBlockSetter]("", "iam_blocks", buildIamBlockColumns("iam_blocks"))

// IamBlocksQuery is a query on the iam_blocks table
type IamBlocksQuery = *psql.ViewQuery[*IamBlock, IamBlockSlice]

func buildIamBlockColumns(tableName string) iamBlockColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "subject", "created_at", "expires_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamBlockColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ID:          buildIamBlockColumn(tableName, "id"),
		ProjectID:   buildIamBlockColumn(tableName, "project_id"),
		Subject:     buildIamBlockColumn(tableName, "subject"),
		CreatedAt:   buildIamBlockColumn(tableName, "created_at"),
		ExpiresAt:   buildIamBlockColumn(tableName, "expires_at"),
		Data:        buildIamBlockColumn(tableName, "data"),
	}
}

type iamBlockColumns struct {
	expr.ColumnsExpr
	tableAlias string
	ID         iamBlockColumn
	ProjectID  iamBlockColumn
	Subject    iamBlockColumn
	CreatedAt  iamBlockColumn
	ExpiresAt  iamBlockColumn
	Data       iamBlockColumn
}

// Alias returns the current table alias for the columns set.
func (c iamBlockColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamBlockColumns) AliasedAs(tableName string) iamBlockColumns {
	return buildIamBlockColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamBlockColumns) Unqualified() iamBlockColumns {
	return buildIamBlockColumns("")
}

func buildIamBlockColumn(alias, name string) iamBlockColumn {
	return iamBlockColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamBlockColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamBlockColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamBlockColumn) ShouldOmitParens() bool {
	return true
}

// IamBlockSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamBlockSetter struct {
	ID        *string              `db:"id,pk" `
	ProjectID *string              `db:"project_id" `
	Subject   *string              `db:"subject" `
	CreatedAt *time.Time           `db:"created_at" `
	ExpiresAt *null.Val[time.Time] `db:"expires_at" `
	Data      *json.RawMessage     `db:"data" `
}

func (s IamBlockSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Subject != nil {
		vals = append(vals, "subject")
	}
	if s.CreatedAt != nil {
		vals = append(vals, "created_at")
	}
	if s.ExpiresAt != nil {
		vals = append(vals, "expires_at")
	}
	if s.Data != nil {
		vals = append(vals, "data")
	}
	return vals
}

func (s IamBlockSetter) Overwrite(t *IamBlock) {
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
	if s.Subject != nil {
		t.Subject = func() string {
			if s.Subject == nil {
				return *new(string)
			}
			return *s.Subject
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
	if s.ExpiresAt != nil {
		t.ExpiresAt = func() null.Val[time.Time] {
			if s.ExpiresAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.ExpiresAt
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

func (s *IamBlockSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamBlocks.BeforeInsertHooks.RunHooks(ctx, exec, s)
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

		if s.Subject != nil {
			vals[2] = psql.Arg(func() string {
				if s.Subject == nil {
					return *new(string)
				}
				return *s.Subject
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

		if s.ExpiresAt != nil {
			vals[4] = psql.Arg(func() null.Val[time.Time] {
				if s.ExpiresAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.ExpiresAt
				return *v
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

func (s IamBlockSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamBlockSetter) Expressions(prefix ...string) []bob.Expression {
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

	if s.Subject != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "subject")...),
			psql.Arg(s.Subject),
		}})
	}

	if s.CreatedAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "created_at")...),
			psql.Arg(s.CreatedAt),
		}})
	}

	if s.ExpiresAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "expires_at")...),
			psql.Arg(s.ExpiresAt),
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

// FindIamBlock retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamBlock(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamBlock, error) {
	if len(cols) == 0 {
		return IamBlocks.Query(
			sm.Where(IamBlocks.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamBlocks.Query(
		sm.Where(IamBlocks.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamBlocks.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamBlockExists checks the presence of a single record by primary key
func IamBlockExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamBlocks.Query(
		sm.Where(IamBlocks.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamBlock is retrieved from the database
func (o *IamBlock) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamBlocks.AfterSelectHooks.RunHooks(ctx, exec, IamBlockSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamBlocks.AfterInsertHooks.RunHooks(ctx, exec, IamBlockSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamBlocks.AfterUpdateHooks.RunHooks(ctx, exec, IamBlockSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamBlocks.AfterDeleteHooks.RunHooks(ctx, exec, IamBlockSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamBlocks.AfterMergeHooks.RunHooks(ctx, exec, IamBlockSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamBlock
func (o *IamBlock) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamBlock) pkEQ() dialect.Expression {
	return psql.Quote("iam_blocks", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamBlock
func (o *IamBlock) Update(ctx context.Context, exec bob.Executor, s *IamBlockSetter) error {
	v, err := IamBlocks.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamBlock record with an executor
func (o *IamBlock) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamBlocks.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamBlock using the executor
func (o *IamBlock) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamBlocks.Query(
		sm.Where(IamBlocks.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamBlockSlice is retrieved from the database
func (o IamBlockSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamBlocks.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamBlocks.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamBlocks.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamBlocks.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamBlocks.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamBlockSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_blocks", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamBlockSlice) copyMatchingRows(from ...*IamBlock) {
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
func (o IamBlockSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamBlocks.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamBlock:
				o.copyMatchingRows(retrieved)
			case []*IamBlock:
				o.copyMatchingRows(retrieved...)
			case IamBlockSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamBlock or a slice of IamBlock
				// then run the AfterUpdateHooks on the slice
				_, err = IamBlocks.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamBlockSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamBlocks.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamBlock:
				o.copyMatchingRows(retrieved)
			case []*IamBlock:
				o.copyMatchingRows(retrieved...)
			case IamBlockSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamBlock or a slice of IamBlock
				// then run the AfterDeleteHooks on the slice
				_, err = IamBlocks.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamBlockSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamBlocks.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamBlock:
				o.copyMatchingRows(retrieved)
			case []*IamBlock:
				o.copyMatchingRows(retrieved...)
			case IamBlockSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamBlock or a slice of IamBlock
				// then run the AfterMergeHooks on the slice
				_, err = IamBlocks.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamBlockSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamBlockSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamBlocks.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamBlockSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamBlocks.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamBlockSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamBlocks.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamBlockWhere[Q psql.Filterable] struct {
	ID        psql.WhereMod[Q, string]
	ProjectID psql.WhereMod[Q, string]
	Subject   psql.WhereMod[Q, string]
	CreatedAt psql.WhereMod[Q, time.Time]
	ExpiresAt psql.WhereNullMod[Q, time.Time]
	Data      psql.WhereMod[Q, json.RawMessage]
}

func (iamBlockWhere[Q]) AliasedAs(alias string) iamBlockWhere[Q] {
	return buildIamBlockWhere[Q](buildIamBlockColumns(alias))
}

func buildIamBlockWhere[Q psql.Filterable](cols iamBlockColumns) iamBlockWhere[Q] {
	return iamBlockWhere[Q]{
		ID:        psql.Where[Q, string](cols.ID.Expression),
		ProjectID: psql.Where[Q, string](cols.ProjectID.Expression),
		Subject:   psql.Where[Q, string](cols.Subject.Expression),
		CreatedAt: psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		ExpiresAt: psql.WhereNull[Q, time.Time](cols.ExpiresAt.Expression),
		Data:      psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
