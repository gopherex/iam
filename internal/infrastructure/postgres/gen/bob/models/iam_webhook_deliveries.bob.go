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

// IamWebhookDelivery is an object representing the database table.
type IamWebhookDelivery struct {
	ID             string              `db:"id,pk" `
	ProjectID      string              `db:"project_id" `
	Environment    string              `db:"environment" `
	WebhookID      string              `db:"webhook_id" `
	EventID        string              `db:"event_id" `
	Status         string              `db:"status" `
	AttemptCount   int32               `db:"attempt_count" `
	NextAttemptAt  null.Val[time.Time] `db:"next_attempt_at" `
	LastAttemptAt  null.Val[time.Time] `db:"last_attempt_at" `
	DeliveredAt    null.Val[time.Time] `db:"delivered_at" `
	ResponseStatus null.Val[int32]     `db:"response_status" `
	ResponseBody   null.Val[string]    `db:"response_body" `
	LastError      null.Val[string]    `db:"last_error" `
	CreatedAt      time.Time           `db:"created_at" `
	UpdatedAt      time.Time           `db:"updated_at" `
	Data           json.RawMessage     `db:"data" `
}

// IamWebhookDeliverySlice is an alias for a slice of pointers to IamWebhookDelivery.
// This should almost always be used instead of []*IamWebhookDelivery.
type IamWebhookDeliverySlice []*IamWebhookDelivery

// IamWebhookDeliveries contains methods to work with the iam_webhook_deliveries table
var IamWebhookDeliveries = psql.NewTablex[*IamWebhookDelivery, IamWebhookDeliverySlice, *IamWebhookDeliverySetter]("", "iam_webhook_deliveries", buildIamWebhookDeliveryColumns("iam_webhook_deliveries"))

// IamWebhookDeliveriesQuery is a query on the iam_webhook_deliveries table
type IamWebhookDeliveriesQuery = *psql.ViewQuery[*IamWebhookDelivery, IamWebhookDeliverySlice]

func buildIamWebhookDeliveryColumns(tableName string) iamWebhookDeliveryColumns {
	columnsExpr := expr.NewColumnsExpr(
		"id", "project_id", "environment", "webhook_id", "event_id", "status", "attempt_count", "next_attempt_at", "last_attempt_at", "delivered_at", "response_status", "response_body", "last_error", "created_at", "updated_at", "data",
	)

	if tableName != "" {
		columnsExpr = columnsExpr.WithParent(tableName)
	}

	return iamWebhookDeliveryColumns{
		ColumnsExpr:    columnsExpr,
		tableAlias:     tableName,
		ID:             buildIamWebhookDeliveryColumn(tableName, "id"),
		ProjectID:      buildIamWebhookDeliveryColumn(tableName, "project_id"),
		Environment:    buildIamWebhookDeliveryColumn(tableName, "environment"),
		WebhookID:      buildIamWebhookDeliveryColumn(tableName, "webhook_id"),
		EventID:        buildIamWebhookDeliveryColumn(tableName, "event_id"),
		Status:         buildIamWebhookDeliveryColumn(tableName, "status"),
		AttemptCount:   buildIamWebhookDeliveryColumn(tableName, "attempt_count"),
		NextAttemptAt:  buildIamWebhookDeliveryColumn(tableName, "next_attempt_at"),
		LastAttemptAt:  buildIamWebhookDeliveryColumn(tableName, "last_attempt_at"),
		DeliveredAt:    buildIamWebhookDeliveryColumn(tableName, "delivered_at"),
		ResponseStatus: buildIamWebhookDeliveryColumn(tableName, "response_status"),
		ResponseBody:   buildIamWebhookDeliveryColumn(tableName, "response_body"),
		LastError:      buildIamWebhookDeliveryColumn(tableName, "last_error"),
		CreatedAt:      buildIamWebhookDeliveryColumn(tableName, "created_at"),
		UpdatedAt:      buildIamWebhookDeliveryColumn(tableName, "updated_at"),
		Data:           buildIamWebhookDeliveryColumn(tableName, "data"),
	}
}

type iamWebhookDeliveryColumns struct {
	expr.ColumnsExpr
	tableAlias     string
	ID             iamWebhookDeliveryColumn
	ProjectID      iamWebhookDeliveryColumn
	Environment    iamWebhookDeliveryColumn
	WebhookID      iamWebhookDeliveryColumn
	EventID        iamWebhookDeliveryColumn
	Status         iamWebhookDeliveryColumn
	AttemptCount   iamWebhookDeliveryColumn
	NextAttemptAt  iamWebhookDeliveryColumn
	LastAttemptAt  iamWebhookDeliveryColumn
	DeliveredAt    iamWebhookDeliveryColumn
	ResponseStatus iamWebhookDeliveryColumn
	ResponseBody   iamWebhookDeliveryColumn
	LastError      iamWebhookDeliveryColumn
	CreatedAt      iamWebhookDeliveryColumn
	UpdatedAt      iamWebhookDeliveryColumn
	Data           iamWebhookDeliveryColumn
}

// Alias returns the current table alias for the columns set.
func (c iamWebhookDeliveryColumns) Alias() string {
	return c.tableAlias
}

// AliasedAs returns a copy of the columns set qualified by tableName.
func (iamWebhookDeliveryColumns) AliasedAs(tableName string) iamWebhookDeliveryColumns {
	return buildIamWebhookDeliveryColumns(tableName)
}

// Unqualified returns a copy of the columns set without table qualification.
func (c iamWebhookDeliveryColumns) Unqualified() iamWebhookDeliveryColumns {
	return buildIamWebhookDeliveryColumns("")
}

func buildIamWebhookDeliveryColumn(alias, name string) iamWebhookDeliveryColumn {
	return iamWebhookDeliveryColumn{
		Expression: psql.Quote(alias, name),
		alias:      alias,
		name:       name,
	}
}

type iamWebhookDeliveryColumn struct {
	psql.Expression
	alias string
	name  string
}

// Name returns the unqualified column name.
func (c iamWebhookDeliveryColumn) Name() string {
	return c.name
}

// ShouldOmitParens prevents automatic parenthesis wrapping in expression builders.
func (c iamWebhookDeliveryColumn) ShouldOmitParens() bool {
	return true
}

// IamWebhookDeliverySetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type IamWebhookDeliverySetter struct {
	ID             *string              `db:"id,pk" `
	ProjectID      *string              `db:"project_id" `
	Environment    *string              `db:"environment" `
	WebhookID      *string              `db:"webhook_id" `
	EventID        *string              `db:"event_id" `
	Status         *string              `db:"status" `
	AttemptCount   *int32               `db:"attempt_count" `
	NextAttemptAt  *null.Val[time.Time] `db:"next_attempt_at" `
	LastAttemptAt  *null.Val[time.Time] `db:"last_attempt_at" `
	DeliveredAt    *null.Val[time.Time] `db:"delivered_at" `
	ResponseStatus *null.Val[int32]     `db:"response_status" `
	ResponseBody   *null.Val[string]    `db:"response_body" `
	LastError      *null.Val[string]    `db:"last_error" `
	CreatedAt      *time.Time           `db:"created_at" `
	UpdatedAt      *time.Time           `db:"updated_at" `
	Data           *json.RawMessage     `db:"data" `
}

func (s IamWebhookDeliverySetter) SetColumns() []string {
	vals := make([]string, 0, 16)
	if s.ID != nil {
		vals = append(vals, "id")
	}
	if s.ProjectID != nil {
		vals = append(vals, "project_id")
	}
	if s.Environment != nil {
		vals = append(vals, "environment")
	}
	if s.WebhookID != nil {
		vals = append(vals, "webhook_id")
	}
	if s.EventID != nil {
		vals = append(vals, "event_id")
	}
	if s.Status != nil {
		vals = append(vals, "status")
	}
	if s.AttemptCount != nil {
		vals = append(vals, "attempt_count")
	}
	if s.NextAttemptAt != nil {
		vals = append(vals, "next_attempt_at")
	}
	if s.LastAttemptAt != nil {
		vals = append(vals, "last_attempt_at")
	}
	if s.DeliveredAt != nil {
		vals = append(vals, "delivered_at")
	}
	if s.ResponseStatus != nil {
		vals = append(vals, "response_status")
	}
	if s.ResponseBody != nil {
		vals = append(vals, "response_body")
	}
	if s.LastError != nil {
		vals = append(vals, "last_error")
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

func (s IamWebhookDeliverySetter) Overwrite(t *IamWebhookDelivery) {
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
	if s.WebhookID != nil {
		t.WebhookID = func() string {
			if s.WebhookID == nil {
				return *new(string)
			}
			return *s.WebhookID
		}()
	}
	if s.EventID != nil {
		t.EventID = func() string {
			if s.EventID == nil {
				return *new(string)
			}
			return *s.EventID
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
	if s.AttemptCount != nil {
		t.AttemptCount = func() int32 {
			if s.AttemptCount == nil {
				return *new(int32)
			}
			return *s.AttemptCount
		}()
	}
	if s.NextAttemptAt != nil {
		t.NextAttemptAt = func() null.Val[time.Time] {
			if s.NextAttemptAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.NextAttemptAt
			return *v
		}()
	}
	if s.LastAttemptAt != nil {
		t.LastAttemptAt = func() null.Val[time.Time] {
			if s.LastAttemptAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.LastAttemptAt
			return *v
		}()
	}
	if s.DeliveredAt != nil {
		t.DeliveredAt = func() null.Val[time.Time] {
			if s.DeliveredAt == nil {
				return *new(null.Val[time.Time])
			}
			v := s.DeliveredAt
			return *v
		}()
	}
	if s.ResponseStatus != nil {
		t.ResponseStatus = func() null.Val[int32] {
			if s.ResponseStatus == nil {
				return *new(null.Val[int32])
			}
			v := s.ResponseStatus
			return *v
		}()
	}
	if s.ResponseBody != nil {
		t.ResponseBody = func() null.Val[string] {
			if s.ResponseBody == nil {
				return *new(null.Val[string])
			}
			v := s.ResponseBody
			return *v
		}()
	}
	if s.LastError != nil {
		t.LastError = func() null.Val[string] {
			if s.LastError == nil {
				return *new(null.Val[string])
			}
			v := s.LastError
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

func (s *IamWebhookDeliverySetter) Apply(q *dialect.InsertQuery) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return IamWebhookDeliveries.BeforeInsertHooks.RunHooks(ctx, exec, s)
	})

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 16)
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

		if s.WebhookID != nil {
			vals[3] = psql.Arg(func() string {
				if s.WebhookID == nil {
					return *new(string)
				}
				return *s.WebhookID
			}())
		} else {
			vals[3] = psql.Raw("DEFAULT")
		}

		if s.EventID != nil {
			vals[4] = psql.Arg(func() string {
				if s.EventID == nil {
					return *new(string)
				}
				return *s.EventID
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

		if s.AttemptCount != nil {
			vals[6] = psql.Arg(func() int32 {
				if s.AttemptCount == nil {
					return *new(int32)
				}
				return *s.AttemptCount
			}())
		} else {
			vals[6] = psql.Raw("DEFAULT")
		}

		if s.NextAttemptAt != nil {
			vals[7] = psql.Arg(func() null.Val[time.Time] {
				if s.NextAttemptAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.NextAttemptAt
				return *v
			}())
		} else {
			vals[7] = psql.Raw("DEFAULT")
		}

		if s.LastAttemptAt != nil {
			vals[8] = psql.Arg(func() null.Val[time.Time] {
				if s.LastAttemptAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.LastAttemptAt
				return *v
			}())
		} else {
			vals[8] = psql.Raw("DEFAULT")
		}

		if s.DeliveredAt != nil {
			vals[9] = psql.Arg(func() null.Val[time.Time] {
				if s.DeliveredAt == nil {
					return *new(null.Val[time.Time])
				}
				v := s.DeliveredAt
				return *v
			}())
		} else {
			vals[9] = psql.Raw("DEFAULT")
		}

		if s.ResponseStatus != nil {
			vals[10] = psql.Arg(func() null.Val[int32] {
				if s.ResponseStatus == nil {
					return *new(null.Val[int32])
				}
				v := s.ResponseStatus
				return *v
			}())
		} else {
			vals[10] = psql.Raw("DEFAULT")
		}

		if s.ResponseBody != nil {
			vals[11] = psql.Arg(func() null.Val[string] {
				if s.ResponseBody == nil {
					return *new(null.Val[string])
				}
				v := s.ResponseBody
				return *v
			}())
		} else {
			vals[11] = psql.Raw("DEFAULT")
		}

		if s.LastError != nil {
			vals[12] = psql.Arg(func() null.Val[string] {
				if s.LastError == nil {
					return *new(null.Val[string])
				}
				v := s.LastError
				return *v
			}())
		} else {
			vals[12] = psql.Raw("DEFAULT")
		}

		if s.CreatedAt != nil {
			vals[13] = psql.Arg(func() time.Time {
				if s.CreatedAt == nil {
					return *new(time.Time)
				}
				return *s.CreatedAt
			}())
		} else {
			vals[13] = psql.Raw("DEFAULT")
		}

		if s.UpdatedAt != nil {
			vals[14] = psql.Arg(func() time.Time {
				if s.UpdatedAt == nil {
					return *new(time.Time)
				}
				return *s.UpdatedAt
			}())
		} else {
			vals[14] = psql.Raw("DEFAULT")
		}

		if s.Data != nil {
			vals[15] = psql.Arg(func() json.RawMessage {
				if s.Data == nil {
					return *new(json.RawMessage)
				}
				return *s.Data
			}())
		} else {
			vals[15] = psql.Raw("DEFAULT")
		}

		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
	}))
}

func (s IamWebhookDeliverySetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s IamWebhookDeliverySetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 16)

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

	if s.WebhookID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "webhook_id")...),
			psql.Arg(s.WebhookID),
		}})
	}

	if s.EventID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "event_id")...),
			psql.Arg(s.EventID),
		}})
	}

	if s.Status != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "status")...),
			psql.Arg(s.Status),
		}})
	}

	if s.AttemptCount != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "attempt_count")...),
			psql.Arg(s.AttemptCount),
		}})
	}

	if s.NextAttemptAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "next_attempt_at")...),
			psql.Arg(s.NextAttemptAt),
		}})
	}

	if s.LastAttemptAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "last_attempt_at")...),
			psql.Arg(s.LastAttemptAt),
		}})
	}

	if s.DeliveredAt != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "delivered_at")...),
			psql.Arg(s.DeliveredAt),
		}})
	}

	if s.ResponseStatus != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "response_status")...),
			psql.Arg(s.ResponseStatus),
		}})
	}

	if s.ResponseBody != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "response_body")...),
			psql.Arg(s.ResponseBody),
		}})
	}

	if s.LastError != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			psql.Quote(append(prefix, "last_error")...),
			psql.Arg(s.LastError),
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

// FindIamWebhookDelivery retrieves a single record by primary key
// If cols is empty Find will return all columns.
func FindIamWebhookDelivery(ctx context.Context, exec bob.Executor, IDPK string, cols ...string) (*IamWebhookDelivery, error) {
	if len(cols) == 0 {
		return IamWebhookDeliveries.Query(
			sm.Where(IamWebhookDeliveries.Columns.ID.EQ(psql.Arg(IDPK))),
		).One(ctx, exec)
	}

	return IamWebhookDeliveries.Query(
		sm.Where(IamWebhookDeliveries.Columns.ID.EQ(psql.Arg(IDPK))),
		sm.Columns(IamWebhookDeliveries.Columns.Only(cols...)),
	).One(ctx, exec)
}

// IamWebhookDeliveryExists checks the presence of a single record by primary key
func IamWebhookDeliveryExists(ctx context.Context, exec bob.Executor, IDPK string) (bool, error) {
	return IamWebhookDeliveries.Query(
		sm.Where(IamWebhookDeliveries.Columns.ID.EQ(psql.Arg(IDPK))),
	).Exists(ctx, exec)
}

// AfterQueryHook is called after IamWebhookDelivery is retrieved from the database
func (o *IamWebhookDelivery) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamWebhookDeliveries.AfterSelectHooks.RunHooks(ctx, exec, IamWebhookDeliverySlice{o})
	case bob.QueryTypeInsert:
		ctx, err = IamWebhookDeliveries.AfterInsertHooks.RunHooks(ctx, exec, IamWebhookDeliverySlice{o})
	case bob.QueryTypeUpdate:
		ctx, err = IamWebhookDeliveries.AfterUpdateHooks.RunHooks(ctx, exec, IamWebhookDeliverySlice{o})
	case bob.QueryTypeDelete:
		ctx, err = IamWebhookDeliveries.AfterDeleteHooks.RunHooks(ctx, exec, IamWebhookDeliverySlice{o})
	case bob.QueryTypeMerge:
		ctx, err = IamWebhookDeliveries.AfterMergeHooks.RunHooks(ctx, exec, IamWebhookDeliverySlice{o})
	}

	return err
}

// primaryKeyVals returns the primary key values of the IamWebhookDelivery
func (o *IamWebhookDelivery) primaryKeyVals() bob.Expression {
	return psql.Arg(o.ID)
}

func (o *IamWebhookDelivery) pkEQ() dialect.Expression {
	return psql.Quote("iam_webhook_deliveries", "id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
	}))
}

// Update uses an executor to update the IamWebhookDelivery
func (o *IamWebhookDelivery) Update(ctx context.Context, exec bob.Executor, s *IamWebhookDeliverySetter) error {
	v, err := IamWebhookDeliveries.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *v

	return nil
}

// Delete deletes a single IamWebhookDelivery record with an executor
func (o *IamWebhookDelivery) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := IamWebhookDeliveries.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
	return err
}

// Reload refreshes the IamWebhookDelivery using the executor
func (o *IamWebhookDelivery) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := IamWebhookDeliveries.Query(
		sm.Where(IamWebhookDeliveries.Columns.ID.EQ(psql.Arg(o.ID))),
	).One(ctx, exec)
	if err != nil {
		return err
	}

	*o = *o2

	return nil
}

// AfterQueryHook is called after IamWebhookDeliverySlice is retrieved from the database
func (o IamWebhookDeliverySlice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	var err error

	switch queryType {
	case bob.QueryTypeSelect:
		ctx, err = IamWebhookDeliveries.AfterSelectHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeInsert:
		ctx, err = IamWebhookDeliveries.AfterInsertHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeUpdate:
		ctx, err = IamWebhookDeliveries.AfterUpdateHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeDelete:
		ctx, err = IamWebhookDeliveries.AfterDeleteHooks.RunHooks(ctx, exec, o)
	case bob.QueryTypeMerge:
		ctx, err = IamWebhookDeliveries.AfterMergeHooks.RunHooks(ctx, exec, o)
	}

	return err
}

func (o IamWebhookDeliverySlice) pkIN() dialect.Expression {
	if len(o) == 0 {
		return psql.Raw("NULL")
	}

	return psql.Quote("iam_webhook_deliveries", "id").In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
func (o IamWebhookDeliverySlice) copyMatchingRows(from ...*IamWebhookDelivery) {
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
func (o IamWebhookDeliverySlice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebhookDeliveries.BeforeUpdateHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebhookDelivery:
				o.copyMatchingRows(retrieved)
			case []*IamWebhookDelivery:
				o.copyMatchingRows(retrieved...)
			case IamWebhookDeliverySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebhookDelivery or a slice of IamWebhookDelivery
				// then run the AfterUpdateHooks on the slice
				_, err = IamWebhookDeliveries.AfterUpdateHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o IamWebhookDeliverySlice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebhookDeliveries.BeforeDeleteHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebhookDelivery:
				o.copyMatchingRows(retrieved)
			case []*IamWebhookDelivery:
				o.copyMatchingRows(retrieved...)
			case IamWebhookDeliverySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebhookDelivery or a slice of IamWebhookDelivery
				// then run the AfterDeleteHooks on the slice
				_, err = IamWebhookDeliveries.AfterDeleteHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))

		q.AppendWhere(o.pkIN())
	})
}

// MergeMod modifies a merge query to run BeforeMergeHooks and AfterMergeHooks
// and updates the slice with the returned rows.
func (o IamWebhookDeliverySlice) MergeMod() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(q *dialect.MergeQuery) {
		q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
			return IamWebhookDeliveries.BeforeMergeHooks.RunHooks(ctx, exec, o)
		})

		q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
			var err error
			switch retrieved := retrieved.(type) {
			case *IamWebhookDelivery:
				o.copyMatchingRows(retrieved)
			case []*IamWebhookDelivery:
				o.copyMatchingRows(retrieved...)
			case IamWebhookDeliverySlice:
				o.copyMatchingRows(retrieved...)
			default:
				// If the retrieved value is not a IamWebhookDelivery or a slice of IamWebhookDelivery
				// then run the AfterMergeHooks on the slice
				_, err = IamWebhookDeliveries.AfterMergeHooks.RunHooks(ctx, exec, o)
			}

			return err
		}))
	})
}

func (o IamWebhookDeliverySlice) UpdateAll(ctx context.Context, exec bob.Executor, vals IamWebhookDeliverySetter) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamWebhookDeliveries.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
	return err
}

func (o IamWebhookDeliverySlice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	_, err := IamWebhookDeliveries.Delete(o.DeleteMod()).Exec(ctx, exec)
	return err
}

func (o IamWebhookDeliverySlice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	if len(o) == 0 {
		return nil
	}

	o2, err := IamWebhookDeliveries.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

	o.copyMatchingRows(o2...)

	return nil
}

type iamWebhookDeliveryWhere[Q psql.Filterable] struct {
	ID             psql.WhereMod[Q, string]
	ProjectID      psql.WhereMod[Q, string]
	Environment    psql.WhereMod[Q, string]
	WebhookID      psql.WhereMod[Q, string]
	EventID        psql.WhereMod[Q, string]
	Status         psql.WhereMod[Q, string]
	AttemptCount   psql.WhereMod[Q, int32]
	NextAttemptAt  psql.WhereNullMod[Q, time.Time]
	LastAttemptAt  psql.WhereNullMod[Q, time.Time]
	DeliveredAt    psql.WhereNullMod[Q, time.Time]
	ResponseStatus psql.WhereNullMod[Q, int32]
	ResponseBody   psql.WhereNullMod[Q, string]
	LastError      psql.WhereNullMod[Q, string]
	CreatedAt      psql.WhereMod[Q, time.Time]
	UpdatedAt      psql.WhereMod[Q, time.Time]
	Data           psql.WhereMod[Q, json.RawMessage]
}

func (iamWebhookDeliveryWhere[Q]) AliasedAs(alias string) iamWebhookDeliveryWhere[Q] {
	return buildIamWebhookDeliveryWhere[Q](buildIamWebhookDeliveryColumns(alias))
}

func buildIamWebhookDeliveryWhere[Q psql.Filterable](cols iamWebhookDeliveryColumns) iamWebhookDeliveryWhere[Q] {
	return iamWebhookDeliveryWhere[Q]{
		ID:             psql.Where[Q, string](cols.ID.Expression),
		ProjectID:      psql.Where[Q, string](cols.ProjectID.Expression),
		Environment:    psql.Where[Q, string](cols.Environment.Expression),
		WebhookID:      psql.Where[Q, string](cols.WebhookID.Expression),
		EventID:        psql.Where[Q, string](cols.EventID.Expression),
		Status:         psql.Where[Q, string](cols.Status.Expression),
		AttemptCount:   psql.Where[Q, int32](cols.AttemptCount.Expression),
		NextAttemptAt:  psql.WhereNull[Q, time.Time](cols.NextAttemptAt.Expression),
		LastAttemptAt:  psql.WhereNull[Q, time.Time](cols.LastAttemptAt.Expression),
		DeliveredAt:    psql.WhereNull[Q, time.Time](cols.DeliveredAt.Expression),
		ResponseStatus: psql.WhereNull[Q, int32](cols.ResponseStatus.Expression),
		ResponseBody:   psql.WhereNull[Q, string](cols.ResponseBody.Expression),
		LastError:      psql.WhereNull[Q, string](cols.LastError.Expression),
		CreatedAt:      psql.Where[Q, time.Time](cols.CreatedAt.Expression),
		UpdatedAt:      psql.Where[Q, time.Time](cols.UpdatedAt.Expression),
		Data:           psql.Where[Q, json.RawMessage](cols.Data.Expression),
	}
}
