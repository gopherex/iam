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

// IamSecurityAttempt is an object representing the database table.
type IamSecurityAttempt struct {
	ProjectID   string    `db:"project_id,pk" `
	Environment string    `db:"environment,pk" `
	SubjectHash string    `db:"subject_hash,pk" `
	Kind        string    `db:"kind,pk" `
	WindowStart time.Time `db:"window_start" `
	Attempts    int32     `db:"attempts" `
}

// IamSecurityAttemptSlice is an alias for a slice of pointers to IamSecurityAttempt.
// This should almost always be used instead of []*IamSecurityAttempt.
type IamSecurityAttemptSlice []*IamSecurityAttempt

// IamSecurityAttempts contains methods to work with the iam_security_attempts table
var IamSecurityAttempts = psql.NewTablex[*IamSecurityAttempt, IamSecurityAttemptSlice, *IamSecurityAttemptSetter]("", "iam_security_attempts", buildIamSecurityAttemptColumns("iam_security_attempts"))

// IamSecurityAttemptsQuery is a query on the iam_security_attempts table
type IamSecurityAttemptsQuery = *psql.ViewQuery[*IamSecurityAttempt, IamSecurityAttemptSlice]

func buildIamSecurityAttemptColumns(tableName string) iamSecurityAttemptColumns {
	columnsExpr := expr.NewColumnsExpr(
		"project_id", "environment", "subject_hash", "kind", "window_start", "attempts",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamSecurityAttemptColumns{
		ColumnsExpr: columnsExpr,
		tableAlias:  tableName,
		ProjectID:   buildIamSecurityAttemptColumn(tableName, "project_id"),
		Environment: buildIamSecurityAttemptColumn(tableName, "environment"),
		SubjectHash: buildIamSecurityAttemptColumn(tableName, "subject_hash"),
		Kind:        buildIamSecurityAttemptColumn(tableName, "kind"),
		WindowStart: buildIamSecurityAttemptColumn(tableName, "window_start"),
		Attempts:    buildIamSecurityAttemptColumn(tableName, "attempts"),
	}
}

type iamSecurityAttemptColumns struct {
	expr.ColumnsExpr
	tableAlias  string
	ProjectID   iamSecurityAttemptColumn
	Environment iamSecurityAttemptColumn
	SubjectHash iamSecurityAttemptColumn
	Kind        iamSecurityAttemptColumn
	WindowStart iamSecurityAttemptColumn
	Attempts    iamSecurityAttemptColumn
}

// Alias returns the current table alias for the columns set.
func (c iamSecurityAttemptColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamSecurityAttemptColumns) AliasedAs(tableName string) iamSecurityAttemptColumns {
	return buildIamSecurityAttemptColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamSecurityAttemptColumns) Unqualified() iamSecurityAttemptColumns {
	return buildIamSecurityAttemptColumns("")
}

func buildIamSecurityAttemptColumn(alias, name string) iamSecurityAttemptColumn {
	return iamSecurityAttemptColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamSecurityAttemptColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamSecurityAttemptColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamSecurityAttemptColumn) ShouldOmitParens() bool {
	return true
}

// IamSecurityAttemptSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamSecurityAttemptSetter struct {
	ProjectID   *string    `db:"project_id,pk" `
	Environment *string    `db:"environment,pk" `
	SubjectHash *string    `db:"subject_hash,pk" `
	Kind        *string    `db:"kind,pk" `
	WindowStart *time.Time `db:"window_start" `
	Attempts    *int32     `db:"attempts" `
}

func (s IamSecurityAttemptSetter) SetColumns() []string {
	vals := make([]string, 0, 6)
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.SubjectHash != nil {
		vals = append(vals, "subject_hash")
	}
	if s.Kind != nil {
		vals = append(vals, "kind")
	}
	if s.WindowStart != nil {
		vals = append(vals, "window_start")
	}
	if s.Attempts != nil {
		vals = append(vals, "attempts")
	}
	return vals
}

func (s IamSecurityAttemptSetter) Overwrite(t *IamSecurityAttempt) {
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
	if s.SubjectHash != nil {
		t.SubjectHash = func() string {
			if s.SubjectHash == nil {
				return *new(string)
			}
			return *s.SubjectHash
		}()
	}
	if s.Kind != nil {
		t.Kind = func() string {
			if s.Kind == nil {
				return *new(string)
			}
			return *s.Kind
		}()
	}
	if s.WindowStart != nil {
		t.WindowStart = func() time.Time {
			if s.WindowStart == nil {
				return *new(time.Time)
			}
			return *s.WindowStart
		}()
	}
	if s.Attempts != nil {
		t.Attempts = func() int32 {
			if s.Attempts == nil {
				return *new(int32)
			}
			return *s.Attempts
		}()
	}
}

func (s *IamSecurityAttemptSetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamSecurityAttempts.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 6)
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

		if s.SubjectHash != nil {
			vals[2] = psql.Arg(func() string {
				if s.SubjectHash == nil {
					return *new(string)
				}
				return *s.SubjectHash
			}())
		} else {
			vals[2] = psql.Raw("DEFAULT")
		}

		if s.Kind != nil {
			vals[3] = psql.Arg(func() string {
				if s.Kind == nil {
					return *new(string)
				}
				return *s.Kind
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.WindowStart != nil {
			vals[4] = psql.Arg(func() time.Time {
				if s.WindowStart == nil {
					return *new(time.Time)
				}
				return *s.WindowStart
			}())
		} else {
			vals[4] = psql.Raw("DEFAULT")
		}

		if s.Attempts != nil {
			vals[5] = psql.Arg(func() int32 {
				if s.Attempts == nil {
					return *new(int32)
				}
				return *s.Attempts
			}())
		} else {
			vals[5] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamSecurityAttemptSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamSecurityAttemptSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 6)

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

	if s.SubjectHash != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "subject_hash")...),
			psql.Arg(s.SubjectHash),
		}})
	}

	if s.Kind != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "kind")...),
			psql.Arg(s.Kind),
		}})
	}

	if s.WindowStart != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "window_start")...),
			psql.Arg(s.WindowStart),
		}})
	}

	if s.Attempts != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "attempts")...),
			psql.Arg(s.Attempts),
		}})
	}

	return exprs
}

// FindIamSecurityAttempt retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamSecurityAttempt(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, SubjectHashPK string, KindPK string, cols ...string) (*IamSecurityAttempt, error) {
	if len(cols) == 0 {
		return IamSecurityAttempts.Query(
			sm.Where(IamSecurityAttempts.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
			sm.Where(IamSecurityAttempts.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
			sm.Where(IamSecurityAttempts.Columns.SubjectHash.EQ(psql.Arg(SubjectHashPK))),
			sm.Where(IamSecurityAttempts.Columns.Kind.EQ(psql.Arg(KindPK))),
		).One(ctx, exec)
	}

	return IamSecurityAttempts.Query(
		sm.Where(IamSecurityAttempts.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamSecurityAttempts.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamSecurityAttempts.Columns.SubjectHash.EQ(psql.Arg(SubjectHashPK))),
		sm.Where(IamSecurityAttempts.Columns.Kind.EQ(psql.Arg(KindPK))),
		sm.Columns(IamSecurityAttempts.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamSecurityAttemptExists checks the presence of a single record by primary key
func IamSecurityAttemptExists(ctx context.Context, exec bob.Executor, ProjectIDPK string, EnvironmentPK string, SubjectHashPK string, KindPK string) (bool, error) {
	return IamSecurityAttempts.Query(
		sm.Where(IamSecurityAttempts.Columns.ProjectID.EQ(psql.Arg(ProjectIDPK))),
		sm.Where(IamSecurityAttempts.Columns.Environment.EQ(psql.Arg(EnvironmentPK))),
		sm.Where(IamSecurityAttempts.Columns.SubjectHash.EQ(psql.Arg(SubjectHashPK))),
		sm.Where(IamSecurityAttempts.Columns.Kind.EQ(psql.Arg(KindPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamSecurityAttempt is retrieved from the database
func (o *IamSecurityAttempt) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityAttempts.AfterSelectHooks.RunHooks(ctx, exec, IamSecurityAttemptSlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityAttempts.AfterInsertHooks.RunHooks(ctx, exec, IamSecurityAttemptSlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityAttempts.AfterUpdateHooks.RunHooks(ctx, exec, IamSecurityAttemptSlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityAttempts.AfterDeleteHooks.RunHooks(ctx, exec, IamSecurityAttemptSlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityAttempts.AfterMergeHooks.RunHooks(ctx, exec, IamSecurityAttemptSlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamSecurityAttempt
func (o *IamSecurityAttempt) primaryKeyVals() bob.Expression {
	return psql.ArgGroup(
		o.ProjectID,
		o.Environment,
		o.SubjectHash,
		o.Kind,
	)
}

func (o *IamSecurityAttempt) pkEQ() dialect.Expression {
	return psql.Group(psql.Quote("iam_security_attempts", "project_id"), psql.Quote("iam_security_attempts", "environment"), psql.Quote("iam_security_attempts", "subject_hash"), psql.Quote("iam_security_attempts", "kind")).EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamSecurityAttempt
func (o *IamSecurityAttempt) Update(ctx context.Context, exec bob.Executor, s *IamSecurityAttemptSetter) error {
	v, err := IamSecurityAttempts.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamSecurityAttempt record with an executor
func (o *IamSecurityAttempt) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamSecurityAttempts.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamSecurityAttempt using the executor
func (o *IamSecurityAttempt) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamSecurityAttempts.Query(
		sm.Where(IamSecurityAttempts.Columns.ProjectID.EQ(psql.Arg(o.ProjectID))),
		sm.Where(IamSecurityAttempts.Columns.Environment.EQ(psql.Arg(o.Environment))),
		sm.Where(IamSecurityAttempts.Columns.SubjectHash.EQ(psql.Arg(o.SubjectHash))),
		sm.Where(IamSecurityAttempts.Columns.Kind.EQ(psql.Arg(o.Kind))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamSecurityAttemptSlice is retrieved from the database
func (o IamSecurityAttemptSlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamSecurityAttempts.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamSecurityAttempts.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamSecurityAttempts.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamSecurityAttempts.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamSecurityAttempts.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamSecurityAttemptSlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Group(psql.Quote("iam_security_attempts", "project_id"), psql.Quote("iam_security_attempts", "environment"), psql.Quote("iam_security_attempts", "subject_hash"), psql.Quote("iam_security_attempts", "kind")).In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamSecurityAttemptSlice) copyMatchingRows(from ...*IamSecurityAttempt) {
	for i, old := range o {
		for _, new := range from {
			if new.ProjectID != old.ProjectID {
				continue
			}
			if new.Environment != old.Environment {
				continue
			}
			if new.SubjectHash != old.SubjectHash {
				continue
			}
			if new.Kind != old.Kind {
				continue
			}

			o[i] = new
			break
		}
	}
}

// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o IamSecurityAttemptSlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityAttempts.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityAttempt:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityAttempt:
				o.copyMatchingRows(retrieved...)
			case IamSecurityAttemptSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityAttempt or a slice of IamSecurityAttempt
				// then run the AfterUpdateHooks on the slice
				_, err = IamSecurityAttempts.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamSecurityAttemptSlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityAttempts.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityAttempt:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityAttempt:
				o.copyMatchingRows(retrieved...)
			case IamSecurityAttemptSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityAttempt or a slice of IamSecurityAttempt
				// then run the AfterDeleteHooks on the slice
				_, err = IamSecurityAttempts.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamSecurityAttemptSlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamSecurityAttempts.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamSecurityAttempt:
				o.copyMatchingRows(retrieved)
			case []*IamSecurityAttempt:
				o.copyMatchingRows(retrieved...)
			case IamSecurityAttemptSlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamSecurityAttempt or a slice of IamSecurityAttempt
				// then run the AfterMergeHooks on the slice
				_, err = IamSecurityAttempts.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamSecurityAttemptSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamSecurityAttemptSetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityAttempts.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamSecurityAttemptSlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamSecurityAttempts.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamSecurityAttemptSlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamSecurityAttempts.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamSecurityAttemptWhere[Q psql.Filterable] struct {
	ProjectID   psql.WhereMod[Q, string]
	Environment psql.WhereMod[Q, string]
	SubjectHash psql.WhereMod[Q, string]
	Kind        psql.WhereMod[Q, string]
	WindowStart psql.WhereMod[Q, time.Time]
	Attempts    psql.WhereMod[Q, int32]
}

func (iamSecurityAttemptWhere[Q]) AliasedAs(alias string) iamSecurityAttemptWhere[Q] {
	return buildIamSecurityAttemptWhere[Q](buildIamSecurityAttemptColumns(alias))
}

func buildIamSecurityAttemptWhere[Q psql.Filterable](cols iamSecurityAttemptColumns) iamSecurityAttemptWhere[Q] {
	return iamSecurityAttemptWhere[Q]{
		ProjectID:   psql.Where[Q, string](cols.ProjectID.Expression),
		Environment: psql.Where[Q, string](cols.Environment.Expression),
		SubjectHash: psql.Where[Q, string](cols.SubjectHash.Expression),
		Kind:        psql.Where[Q, string](cols.Kind.Expression),
		WindowStart: psql.Where[Q, time.Time](cols.WindowStart.Expression),
		Attempts:    psql.Where[Q, int32](cols.Attempts.Expression),
	}
}
