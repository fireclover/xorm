// Copyright 2022 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"fmt"

	"xorm.io/builder"
)

func (statement *Statement) HasOrderBy() bool {
	return len(statement.orderBy) > 0
}

// ResetOrderBy reset ordery conditions
func (statement *Statement) ResetOrderBy() {
	statement.orderBy = []orderBy{}
}

func (statement *Statement) writeOrderBy(w builder.Writer, orderBy orderBy) error {
	switch t := orderBy.orderStr.(type) {
	case (*builder.Expression):
		if _, err := fmt.Fprint(w, statement.ReplaceQuote(t.Content())); err != nil {
			return err
		}
		w.Append(t.Args()...)
		return nil
	case string:
		if _, err := fmt.Fprint(w, statement.dialect.Quoter().Quote(t)); err != nil {
			return err
		}
		w.Append(orderBy.orderArgs...)
		return nil
	default:
		return ErrUnSupportedSQLType
	}
}

// WriteOrderBy write order by to writer
func (statement *Statement) writeOrderBys(w builder.Writer) error {
	if len(statement.orderBy) == 0 {
		return nil
	}

	if _, err := fmt.Fprint(w, " ORDER BY "); err != nil {
		return err
	}
	for i, ob := range statement.orderBy {
		if err := statement.writeOrderBy(w, ob); err != nil {
			return err
		}
		if i < len(statement.orderBy)-1 {
			if _, err := fmt.Fprint(w, ", "); err != nil {
				return err
			}
		}
	}
	return nil
}

// OrderBy generate "Order By order" statement
func (statement *Statement) OrderBy(order interface{}, args ...interface{}) *Statement {
	statement.orderBy = append(statement.orderBy, orderBy{order, args})
	return statement
}

// Desc generate `ORDER BY xx DESC`
func (statement *Statement) Desc(colNames ...string) *Statement {
	for _, colName := range colNames {
		statement.orderBy = append(statement.orderBy, orderBy{colName + " DESC", nil})
	}
	return statement
}

// Asc provide asc order by query condition, the input parameters are columns.
func (statement *Statement) Asc(colNames ...string) *Statement {
	for _, colName := range colNames {
		statement.orderBy = append(statement.orderBy, orderBy{colName + " ASC", nil})
	}
	return statement
}
