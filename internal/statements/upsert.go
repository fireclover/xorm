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
func (statement *Statement) GenUpsertSQL(doUpdate bool, columns []string, args []interface{}, uniqueColValMap map[string]interface{}) (string, []interface{}, error) {
	if statement.dialect.URI().DBType == schemas.MSSQL ||
		statement.dialect.URI().DBType == schemas.DAMENG ||
		statement.dialect.URI().DBType == schemas.ORACLE {
		return statement.genMergeSQL(doUpdate, columns, args, uniqueColValMap)
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
			primaryColumnIncluded := false
			for _, primaryKeyColumn := range table.PrimaryKeys {
				if _, has := uniqueColValMap[primaryKeyColumn]; !has {
					continue
				}
				primaryColumnIncluded = true
			}
			if primaryColumnIncluded {
				write(" ON CONFLICT (", quote(table.PrimaryKeys[0]))
				for _, col := range table.PrimaryKeys[1:] {
					write(", ", quote(col))
				}
				write(") DO UPDATE SET ", updateColumns[0], " = excluded.", updateColumns[0])
				for _, column := range updateColumns[1:] {
					write(", ", column, " = excluded.", column)
				}
			}
			for _, index := range table.Indexes {
				if index.Type != schemas.UniqueType {
					continue
				}
				write(" ON CONFLICT (", quote(index.Cols[0]))
				for _, col := range index.Cols[1:] {
					write(", ", quote(col))
				}
				write(") DO UPDATE SET ", updateColumns[0], " = excluded.", updateColumns[0])
				for _, column := range updateColumns[1:] {
					write(", ", column, " = excluded.", column)
				}
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

	if len(table.AutoIncrement) > 0 &&
		(statement.dialect.URI().DBType == schemas.POSTGRES ||
			statement.dialect.URI().DBType == schemas.SQLITE) {
		write(" RETURNING ")
		if err := statement.dialect.Quoter().QuoteTo(buf.Builder, table.AutoIncrement); err != nil {
			return "", nil, err
		}
	}

	return buf.String(), buf.Args(), nil
}

func (statement *Statement) genMergeSQL(doUpdate bool, columns []string, args []interface{}, uniqueColValMap map[string]interface{}) (string, []interface{}, error) {
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

	uniqueCols := make([]string, 0, len(uniqueColValMap))
	for colName := range uniqueColValMap {
		uniqueCols = append(uniqueCols, colName)
	}
	for i, colName := range uniqueCols {
		if err := statement.WriteArg(buf, uniqueColValMap[colName]); err != nil {
			return "", nil, err
		}
		write(" AS ", quote(colName))
		if i < len(uniqueCols)-1 {
			write(", ")
		}
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

	countUniques := 0
	primaryColumnIncluded := false
	for _, primaryKeyColumn := range table.PrimaryKeys {
		if _, has := uniqueColValMap[primaryKeyColumn]; !has {
			continue
		}
		if !primaryColumnIncluded {
			write("(")
		} else {
			write(" AND ")
		}
		write("src.", quote(primaryKeyColumn), " = target.", quote(primaryKeyColumn))
		primaryColumnIncluded = true
	}
	if primaryColumnIncluded {
		write(")")
		countUniques++
	}
	for _, index := range table.Indexes {
		if index.Type != schemas.UniqueType {
			continue
		}
		if countUniques > 0 {
			write(" OR ")
		}
		countUniques++
		write("(")
		write("src.", quote(index.Cols[0]), " = target.", quote(index.Cols[0]))
		for _, col := range index.Cols[1:] {
			write(" AND src.", quote(col), " = target.", quote(col))
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
		write(" DEFAULT VALUES ")
	} else {
		// We have some values - Write the column names we need to insert:
		write(" (")
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
	if err := statement.writeInsertOutput(buf.Builder, table); err != nil {
		return "", nil, err
	}

	write(";")
	return buf.String(), buf.Args(), nil
}

// GenUpsertMapSQL generates insert map SQL
func (statement *Statement) GenUpsertMapSQL(doUpdate bool, columns []string, args []interface{}, uniqueColValMap map[string]interface{}) (string, []interface{}, error) {
	if statement.dialect.URI().DBType == schemas.MSSQL ||
		statement.dialect.URI().DBType == schemas.DAMENG ||
		statement.dialect.URI().DBType == schemas.ORACLE {
		return statement.genMergeMapSQL(doUpdate, columns, args, uniqueColValMap)
	}
	var (
		buf       = builder.NewWriter()
		exprs     = statement.ExprColumns
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
		for _, column := range append(columns, exprs.ColNames()...) {
			if _, has := uniqueColValMap[schemas.CommonQuoter.Trim(column)]; has {
				continue
			}
			updateColumns = append(updateColumns, quote(column))
		}
		doUpdate = doUpdate && (len(updateColumns) > 0)
	}

	if statement.dialect.URI().DBType == schemas.MYSQL && !doUpdate {
		write("INSERT IGNORE INTO ", quote(tableName), " (")
	} else {
		write("INSERT INTO ", quote(tableName), " (")
	}
	if err := statement.dialect.Quoter().JoinWrite(buf.Builder, append(columns, exprs.ColNames()...), ","); err != nil {
		return "", nil, err
	}
	write(")")

	if err := statement.genInsertValuesValues(buf, false, columns, args); err != nil {
		return "", nil, err
	}

	switch statement.dialect.URI().DBType {
	case schemas.SQLITE, schemas.POSTGRES:
		write(" ON CONFLICT DO ")
		if doUpdate {
			write("UPDATE SET ", updateColumns[0], " = excluded.", updateColumns[0])
			for _, column := range updateColumns[1:] {
				write(", ", column, " = excluded.", column)
			}
		} else {
			write("NOTHING")
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

	if len(table.AutoIncrement) > 0 &&
		(statement.dialect.URI().DBType == schemas.POSTGRES ||
			statement.dialect.URI().DBType == schemas.SQLITE) {
		write(" RETURNING ")
		if err := statement.dialect.Quoter().QuoteTo(buf.Builder, table.AutoIncrement); err != nil {
			return "", nil, err
		}
	}

	return buf.String(), buf.Args(), nil
}

func (statement *Statement) genMergeMapSQL(doUpdate bool, columns []string, args []interface{}, uniqueColValMap map[string]interface{}) (string, []interface{}, error) {
	var (
		buf       = builder.NewWriter()
		table     = statement.RefTable
		exprs     = statement.ExprColumns
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

	uniqueCols := make([]string, 0, len(uniqueColValMap))
	for colName := range uniqueColValMap {
		uniqueCols = append(uniqueCols, colName)
	}
	for i, colName := range uniqueCols {
		if err := statement.WriteArg(buf, uniqueColValMap[colName]); err != nil {
			return "", nil, err
		}
		write(" AS ", quote(colName))
		if i < len(uniqueCols)-1 {
			write(", ")
		}
	}
	var updateColumns []string
	var updateArgs []interface{}
	if doUpdate {
		updateColumns = make([]string, 0, len(columns))
		for _, expr := range exprs {
			if _, has := uniqueColValMap[schemas.CommonQuoter.Trim(expr.ColName)]; has {
				continue
			}
			updateColumns = append(updateColumns, quote(expr.ColName))
			updateArgs = append(updateArgs, expr.Arg)
		}
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

	countUniques := 0
	for _, index := range table.Indexes {
		if index.Type != schemas.UniqueType {
			continue
		}
		if countUniques > 0 {
			write(" OR ")
		}
		countUniques++
		write("(")
		write("src.", quote(index.Cols[0]), " = target.", quote(index.Cols[0]))
		for _, col := range index.Cols[1:] {
			write(" AND src.", quote(col), " = target.", quote(col))
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
	if err := statement.dialect.Quoter().JoinWrite(buf.Builder, append(columns, exprs.ColNames()...), ","); err != nil {
		return "", nil, err
	}
	write(")")

	if err := statement.genInsertValuesValues(buf, false, columns, args); err != nil {
		return "", nil, err
	}
	write(";")

	return buf.String(), buf.Args(), nil
}
