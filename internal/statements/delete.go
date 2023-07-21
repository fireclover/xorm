// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"errors"
	"fmt"
	"time"

	"xorm.io/builder"
	"xorm.io/xorm/internal/utils"
	"xorm.io/xorm/schemas"
)

func (statement *Statement) writeDeleteOrder(w builder.Writer) error {
	if err := statement.WriteOrderBy(w); err != nil {
		return err
	}

	if statement.LimitN != nil && *statement.LimitN > 0 {
		limitNValue := *statement.LimitN
		if _, err := fmt.Fprintf(w, " LIMIT %d", limitNValue); err != nil {
			return err
		}
	}

	return nil
}

// ErrNotImplemented not implemented
var ErrNotImplemented = errors.New("Not implemented")

func (statement *Statement) writeOrderCond(orderCondWriter builder.Writer, condWriter, orderSQLWriter *builder.BytesWriter, tableName string) error {
	if orderSQLWriter.Len() > 0 {
		switch statement.dialect.URI().DBType {
		case schemas.POSTGRES:
			if condWriter.Len() > 0 {
				fmt.Fprintf(orderCondWriter, " AND ")
			} else {
				fmt.Fprintf(orderCondWriter, " WHERE ")
			}
			fmt.Fprintf(orderCondWriter, "ctid IN (SELECT ctid FROM %s%s)", tableName, orderSQLWriter.String())
			orderCondWriter.Append(orderSQLWriter.Args()...)
		case schemas.SQLITE:
			if condWriter.Len() > 0 {
				fmt.Fprintf(orderCondWriter, " AND ")
			} else {
				fmt.Fprintf(orderCondWriter, " WHERE ")
			}
			fmt.Fprintf(orderCondWriter, "rowid IN (SELECT rowid FROM %s%s)", tableName, orderSQLWriter.String())
			// TODO: how to handle delete limit on mssql?
		case schemas.MSSQL:
			return ErrNotImplemented
		default:
			fmt.Fprint(orderCondWriter, orderSQLWriter.String())
			orderCondWriter.Append(orderSQLWriter.Args()...)
		}
	}
	return nil
}

func (statement *Statement) WriteDelete(realSQLWriter, deleteSQLWriter *builder.BytesWriter, nowTime func(*schemas.Column) (interface{}, time.Time, error)) error {
	var (
		condWriter = builder.NewWriter()
		err        error
	)

	if err = statement.Conds().WriteTo(statement.QuoteReplacer(condWriter)); err != nil {
		return err
	}

	tableNameNoQuote := statement.TableName()
	tableName := statement.dialect.Quoter().Quote(tableNameNoQuote)
	table := statement.RefTable
	fmt.Fprintf(deleteSQLWriter, "DELETE FROM %v", tableName)
	if condWriter.Len() > 0 {
		fmt.Fprintf(deleteSQLWriter, " WHERE %v", condWriter.String())
		deleteSQLWriter.Append(condWriter.Args()...)
	}

	orderSQLWriter := builder.NewWriter()
	if err := statement.writeDeleteOrder(orderSQLWriter); err != nil {
		return err
	}

	orderCondWriter := builder.NewWriter()
	if err := statement.writeOrderCond(orderCondWriter, condWriter, orderSQLWriter, tableName); err != nil {
		return err
	}

	argsForCache := make([]interface{}, 0, len(deleteSQLWriter.Args())*2)
	copy(argsForCache, deleteSQLWriter.Args())
	argsForCache = append(deleteSQLWriter.Args(), argsForCache...)
	if statement.GetUnscoped() || table == nil || table.DeletedColumn() == nil { // tag "deleted" is disabled
		return utils.WriteBuilder(realSQLWriter, deleteSQLWriter, orderCondWriter)
	}

	deletedColumn := table.DeletedColumn()
	if _, err := fmt.Fprintf(realSQLWriter, "UPDATE %v SET %v = ? WHERE %v",
		statement.dialect.Quoter().Quote(statement.TableName()),
		statement.dialect.Quoter().Quote(deletedColumn.Name),
		condWriter.String()); err != nil {
		return err
	}

	val, _, err := nowTime(deletedColumn)
	if err != nil {
		return err
	}
	realSQLWriter.Append(val)
	realSQLWriter.Append(condWriter.Args()...)

	return utils.WriteBuilder(realSQLWriter, orderCondWriter)
}
