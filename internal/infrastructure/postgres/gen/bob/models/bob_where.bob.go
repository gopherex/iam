// Code generated . DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

var (
	SelectWhere     = Where[*dialect.SelectQuery]()
	UpdateWhere     = Where[*dialect.UpdateQuery]()
	DeleteWhere     = Where[*dialect.DeleteQuery]()
	OnConflictWhere = Where[*clause.ConflictClause]() // Used in ON CONFLICT DO UPDATE
)

func Where[Q psql.Filterable]() struct {
	IamUsers    iamUserWhere[Q]
	IamSessions iamSessionWhere[Q]
} {
	return struct {
		IamUsers    iamUserWhere[Q]
		IamSessions iamSessionWhere[Q]
	}{
		IamUsers:    buildIamUserWhere[Q](IamUsers.Columns),
		IamSessions: buildIamSessionWhere[Q](IamSessions.Columns),
	}
}
