// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"errors"
	"fmt"
	"time"

	"xorm.io/xorm/dialects"
	"xorm.io/xorm/schemas"
)

var (
	// ErrNeedDeletedCond delete needs less one condition error
	ErrNeedDeletedCond = errors.New("Delete action needs at least one condition")

	// ErrNotImplemented not implemented
	ErrNotImplemented = errors.New("Not implemented")
)

// GenDeleteSQL generated delete SQL according conditions
func (statement *Statement) GenDeleteSQL(bean interface{}) (string, string, []interface{}, *time.Time, error) {
	condSQL, condArgs, err := statement.GenConds(bean)
	if err != nil {
		return "", "", nil, nil, err
	}
	pLimitN := statement.LimitN
	if len(condSQL) == 0 && (pLimitN == nil || *pLimitN == 0) {
		return "", "", nil, nil, ErrNeedDeletedCond
	}

	var tableNameNoQuote = statement.TableName()
	var tableName = statement.quote(tableNameNoQuote)
	var table = statement.RefTable
	var deleteSQL string
	if len(condSQL) > 0 {
		deleteSQL = fmt.Sprintf("DELETE FROM %v WHERE %v", tableName, condSQL)
	} else {
		deleteSQL = fmt.Sprintf("DELETE FROM %v", tableName)
	}

	var orderSQL string
	if len(statement.OrderStr) > 0 {
		orderSQL += fmt.Sprintf(" ORDER BY %s", statement.OrderStr)
	}
	if pLimitN != nil && *pLimitN > 0 {
		limitNValue := *pLimitN
		orderSQL += fmt.Sprintf(" LIMIT %d", limitNValue)
	}

	if len(orderSQL) > 0 {
		switch statement.dialect.DBType() {
		case schemas.POSTGRES:
			inSQL := fmt.Sprintf("ctid IN (SELECT ctid FROM %s%s)", tableName, orderSQL)
			if len(condSQL) > 0 {
				deleteSQL += " AND " + inSQL
			} else {
				deleteSQL += " WHERE " + inSQL
			}
		case schemas.SQLITE:
			inSQL := fmt.Sprintf("rowid IN (SELECT rowid FROM %s%s)", tableName, orderSQL)
			if len(condSQL) > 0 {
				deleteSQL += " AND " + inSQL
			} else {
				deleteSQL += " WHERE " + inSQL
			}
			// TODO: how to handle delete limit on mssql?
		case schemas.MSSQL:
			return "", "", nil, nil, ErrNotImplemented
		default:
			deleteSQL += orderSQL
		}
	}

	var realSQL string
	if statement.GetUnscoped() || table.DeletedColumn() == nil { // tag "deleted" is disabled
		return deleteSQL, deleteSQL, condArgs, nil, nil
	}

	deletedColumn := table.DeletedColumn()
	realSQL = fmt.Sprintf("UPDATE %v SET %v = ? WHERE %v",
		statement.quote(statement.TableName()),
		statement.quote(deletedColumn.Name),
		condSQL)

	if len(orderSQL) > 0 {
		switch statement.dialect.DBType() {
		case schemas.POSTGRES:
			inSQL := fmt.Sprintf("ctid IN (SELECT ctid FROM %s%s)", tableName, orderSQL)
			if len(condSQL) > 0 {
				realSQL += " AND " + inSQL
			} else {
				realSQL += " WHERE " + inSQL
			}
		case schemas.SQLITE:
			inSQL := fmt.Sprintf("rowid IN (SELECT rowid FROM %s%s)", tableName, orderSQL)
			if len(condSQL) > 0 {
				realSQL += " AND " + inSQL
			} else {
				realSQL += " WHERE " + inSQL
			}
			// TODO: how to handle delete limit on mssql?
		case schemas.MSSQL:
			return "", "", nil, nil, ErrNotImplemented
		default:
			realSQL += orderSQL
		}
	}

	// !oinume! Insert nowTime to the head of statement.Params
	condArgs = append(condArgs, "")
	paramsLen := len(condArgs)
	copy(condArgs[1:paramsLen], condArgs[0:paramsLen-1])

	now := ColumnNow(deletedColumn, statement.defaultTimeZone)
	val := dialects.FormatTime(statement.dialect, deletedColumn.SQLType.Name, now)
	condArgs[0] = val

	return realSQL, deleteSQL, condArgs, &now, nil
}

// ColumnNow returns the current time for a column
func ColumnNow(col *schemas.Column, defaultTimeZone *time.Location) time.Time {
	t := time.Now()
	tz := defaultTimeZone
	if !col.DisableTimeZone && col.TimeZone != nil {
		tz = col.TimeZone
	}
	return t.In(tz)
}
