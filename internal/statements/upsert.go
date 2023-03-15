// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"fmt"

	"xorm.io/builder"
	"xorm.io/xorm/schemas"
)

// GenUpsertSQL generates upsert beans SQL
func (statement *Statement) GenUpsertSQL(doUpdate bool, addOuput bool, columns []string, args []interface{}, uniqueColValMap map[string]interface{}, uniqueConstraints [][]string) (string, []interface{}, error) {
	if statement.dialect.URI().DBType == schemas.MSSQL ||
		statement.dialect.URI().DBType == schemas.DAMENG ||
		statement.dialect.URI().DBType == schemas.ORACLE {
		return statement.genMergeSQL(doUpdate, addOuput, columns, args, uniqueColValMap, uniqueConstraints)
	}

	var (
		buf       = builder.NewWriter()
		table     = statement.RefTable
		tableName = statement.TableName()
	)
	quote := statement.dialect.Quoter().Quote
	write := func(args ...string) {
		for _, arg := range args {
			_, _ = buf.WriteString(arg)
		}
	}

	var updateColumns []string
	if doUpdate {
		updateColumns = make([]string, 0, len(columns))
		for _, column := range columns {
			if _, has := uniqueColValMap[schemas.CommonQuoter.Trim(column)]; has {
				continue
			}
			updateColumns = append(updateColumns, quote(column))
		}
		doUpdate = doUpdate && (len(updateColumns) > 0)
	}

	if statement.dialect.URI().DBType == schemas.MYSQL && !doUpdate {
		write("INSERT IGNORE INTO ")
	} else {
		write("INSERT INTO ")
	}

	if err := statement.dialect.Quoter().QuoteTo(buf.Builder, tableName); err != nil {
		return "", nil, err
	}

	if err := statement.genInsertValues(buf, columns, args); err != nil {
		return "", nil, err
	}

	switch statement.dialect.URI().DBType {
	case schemas.SQLITE:
		write(" ON CONFLICT DO ")
		if doUpdate {
			write("UPDATE SET ", updateColumns[0], " = excluded.", updateColumns[0])
			for _, column := range updateColumns[1:] {
				write(", ", column, " = excluded.", column)
			}
		} else {
			write("NOTHING")
		}
	case schemas.POSTGRES:
		if doUpdate {
			// In doUpdate we know that uniqueConstraints has to be length 1
			write(" ON CONFLICT (", quote(uniqueConstraints[0][0]))
			for _, uniqueColumn := range uniqueConstraints[0][1:] {
				write(", ", uniqueColumn)
			}
			write(") DO UPDATE SET ", updateColumns[0], " = excluded.", updateColumns[0])
			for _, column := range updateColumns[1:] {
				write(", ", column, " = excluded.", column)
			}
		} else {
			write(" ON CONFLICT DO NOTHING")
		}
	case schemas.MYSQL:
		if doUpdate {
			// FIXME: mysql >= 8.0.19 should use table alias
			write(" ON DUPLICATE KEY ")
			write("UPDATE ", updateColumns[0], " = VALUES(", updateColumns[0], ")")
			for _, column := range updateColumns[1:] {
				write(", ", column, " = VALUES(", column, ")")
			}
			if len(table.AutoIncrement) > 0 {
				write(", ", quote(table.AutoIncrement), " = LAST_INSERT_ID(", quote(table.AutoIncrement), ")")
			}
		}
	default:
		return "", nil, fmt.Errorf("unimplemented") // FIXME: UPSERT
	}

	if addOuput {
		if len(table.AutoIncrement) > 0 &&
			(statement.dialect.URI().DBType == schemas.POSTGRES ||
				statement.dialect.URI().DBType == schemas.SQLITE) {
			write(" RETURNING ")
			if err := statement.dialect.Quoter().QuoteTo(buf.Builder, table.AutoIncrement); err != nil {
				return "", nil, err
			}
		}
	}

	return buf.String(), buf.Args(), nil
}

func (statement *Statement) genMergeSQL(doUpdate bool, addOutput bool, columns []string, args []interface{}, uniqueColValMap map[string]interface{}, uniqueConstraints [][]string) (string, []interface{}, error) {
	var (
		buf       = builder.NewWriter()
		table     = statement.RefTable
		tableName = statement.TableName()
	)

	quote := statement.dialect.Quoter().Quote
	write := func(args ...string) {
		for _, arg := range args {
			_, _ = buf.WriteString(arg)
		}
	}

	write("MERGE INTO ", quote(tableName))
	if statement.dialect.URI().DBType == schemas.MSSQL {
		write(" WITH (HOLDLOCK)")
	}
	write(" AS target USING (SELECT ")

	uniqueColumnsCount := 0
	for uniqueColumn, uniqueValue := range uniqueColValMap {
		if uniqueColumnsCount > 0 {
			write(", ")
		}
		if err := statement.WriteArg(buf, uniqueValue); err != nil {
			return "", nil, err
		}
		write(" AS ", quote(uniqueColumn))
		uniqueColumnsCount++
	}

	var updateColumns []string
	var updateArgs []interface{}
	if doUpdate {
		updateColumns = make([]string, 0, len(columns))
		updateArgs = make([]interface{}, 0, len(columns))
		for i, column := range columns {
			if _, has := uniqueColValMap[schemas.CommonQuoter.Trim(column)]; has {
				continue
			}
			updateColumns = append(updateColumns, quote(column))
			updateArgs = append(updateArgs, args[i])
		}
		doUpdate = doUpdate && (len(updateColumns) > 0)
	}

	write(") AS src ON (")
	for i, uniqueColumns := range uniqueConstraints {
		if i > 0 { // if !doUpdate there may be more than one uniqueConstraint
			write(" OR ")
		}
		write("(src.", quote(uniqueColumns[0]), " = target.", quote(uniqueColumns[0]))
		for _, uniqueColumn := range uniqueColumns[1:] {
			write(" AND src.", quote(uniqueColumn), " = target.", quote(uniqueColumn))
		}
		write(")")
	}
	write(")")
	if doUpdate {
		write(" WHEN MATCHED THEN UPDATE SET ")
		write("target.", quote(updateColumns[0]), " = ?")
		buf.Append(updateArgs[0])
		for i, col := range updateColumns[1:] {
			write(", target.", quote(col), " = ?")
			buf.Append(updateArgs[i+1])
		}
	}
	write(" WHEN NOT MATCHED THEN INSERT ")
	includeAutoIncrement := statement.includeAutoIncrement(columns)
	if len(columns) == 0 && statement.dialect.URI().DBType == schemas.MSSQL {
		write("DEFAULT VALUES ")
	} else {
		// We have some values - Write the column names we need to insert:
		write("(")
		if includeAutoIncrement {
			columns = append(columns, table.AutoIncrement)
		}

		if err := statement.dialect.Quoter().JoinWrite(buf.Builder, append(columns, statement.ExprColumns.ColNames()...), ","); err != nil {
			return "", nil, err
		}

		write(")")
		if err := statement.genInsertValuesValues(buf, includeAutoIncrement, columns, args); err != nil {
			return "", nil, err
		}

	}
	if addOutput {
		if err := statement.writeInsertOutput(buf.Builder, table); err != nil {
			return "", nil, err
		}
	}

	write(";")
	return buf.String(), buf.Args(), nil
}
