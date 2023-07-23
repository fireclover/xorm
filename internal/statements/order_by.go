// Copyright 2022 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"fmt"

	"xorm.io/builder"
)

type orderBy struct {
	orderStr  interface{}
	orderArgs []interface{}
	direction string // ASC, DESC or "", "" means raw orderStr
}

func (statement *Statement) HasOrderBy() bool {
	return len(statement.orderBy) > 0
}

// ResetOrderBy reset ordery conditions
func (statement *Statement) ResetOrderBy() {
	statement.orderBy = []orderBy{}
}

func (statement *Statement) writeOrderBy(w *builder.BytesWriter, orderBy orderBy) error {
	switch t := orderBy.orderStr.(type) {
	case (*builder.Expression):
		if _, err := fmt.Fprint(w.Builder, statement.dialect.Quoter().Replace(t.Content())); err != nil {
			return err
		}
		w.Append(t.Args()...)
		return nil
	case string:
		if orderBy.direction == "" {
			if _, err := fmt.Fprint(w.Builder, statement.dialect.Quoter().Replace(t)); err != nil {
				return err
			}
			w.Append(orderBy.orderArgs...)
			return nil
		}
		if err := statement.dialect.Quoter().QuoteTo(w.Builder, t); err != nil {
			return err
		}
		_, err := fmt.Fprint(w, " ", orderBy.direction)
		return err
	default:
		return ErrUnSupportedSQLType
	}
}

// WriteOrderBy write order by to writer
func (statement *Statement) writeOrderBys(w *builder.BytesWriter) error {
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
	statement.orderBy = append(statement.orderBy, orderBy{order, args, ""})
	return statement
}

// Desc generate `ORDER BY xx DESC`
func (statement *Statement) Desc(colNames ...string) *Statement {
	for _, colName := range colNames {
		statement.orderBy = append(statement.orderBy, orderBy{colName, nil, "DESC"})
	}
	return statement
}

// Asc provide asc order by query condition, the input parameters are columns.
func (statement *Statement) Asc(colNames ...string) *Statement {
	for _, colName := range colNames {
		statement.orderBy = append(statement.orderBy, orderBy{colName, nil, "ASC"})
	}
	return statement
}
