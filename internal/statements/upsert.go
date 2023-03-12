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

	if statement.dialect.URI().DBType == schemas.MYSQL && !doUpdate {
		if _, err := buf.WriteString("INSERT IGNORE INTO "); err != nil {
			return "", nil, err
		}
	} else {
		if _, err := buf.WriteString("INSERT INTO "); err != nil {
			return "", nil, err
		}
	}

	if err := statement.dialect.Quoter().QuoteTo(buf.Builder, tableName); err != nil {
		return "", nil, err
	}

	if err := statement.genInsertValues(buf, columns, args); err != nil {
		return "", nil, err
	}

	switch statement.dialect.URI().DBType {
	case schemas.SQLITE, schemas.POSTGRES:
		if _, err := buf.WriteString(" ON CONFLICT DO "); err != nil {
			return "", nil, err
		}
		if doUpdate {
			return "", nil, fmt.Errorf("unimplemented") // FIXME: UPSERT
		} else {
			if _, err := buf.WriteString("NOTHING"); err != nil {
				return "", nil, err
			}
		}
	case schemas.MYSQL:
		if doUpdate {
			return "", nil, fmt.Errorf("unimplemented") // FIXME: UPSERT
		}
	default:
		return "", nil, fmt.Errorf("unimplemented") // FIXME: UPSERT
	}

	if len(table.AutoIncrement) > 0 && statement.dialect.URI().DBType == schemas.POSTGRES {
		if _, err := buf.WriteString(" RETURNING "); err != nil {
			return "", nil, err
		}
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
		write("WITH (HOLDLOCK) AS target ")
	}
	write("USING (SELECT ")

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
		write("src.", quote(index.Cols[0]), "= target.", quote(index.Cols[0]))
		for _, col := range index.Cols[1:] {
			write(" AND src.", quote(col), "= target.", quote(col))
		}
		write(")")
	}
	if doUpdate {
		return "", nil, fmt.Errorf("unimplemented")
	}
	write(") WHEN NOT MATCHED THEN INSERT")
	if err := statement.genInsertValues(buf, columns, args); err != nil {
		return "", nil, err
	}
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
		if _, err := buf.WriteString(" ON CONFLICT DO "); err != nil {
			return "", nil, err
		}
		if doUpdate {
			return "", nil, fmt.Errorf("unimplemented") // FIXME: UPSERT
		} else {
			if _, err := buf.WriteString("NOTHING"); err != nil {
				return "", nil, err
			}
		}
	case schemas.MYSQL:
		if doUpdate {
			return "", nil, fmt.Errorf("unimplemented") // FIXME: UPSERT
		}
	default:
		return "", nil, fmt.Errorf("unimplemented") // FIXME: UPSERT
	}

	if len(table.AutoIncrement) > 0 && statement.dialect.URI().DBType == schemas.POSTGRES {
		if _, err := buf.WriteString(" RETURNING "); err != nil {
			return "", nil, err
		}
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
		write("WITH (HOLDLOCK) AS target ")
	}
	write("USING (SELECT ")

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
		write("src.", quote(index.Cols[0]), "= target.", quote(index.Cols[0]))
		for _, col := range index.Cols[1:] {
			write(" AND src.", quote(col), "= target.", quote(col))
		}
		write(")")
	}
	if doUpdate {
		return "", nil, fmt.Errorf("unimplemented")
	}
	write(") WHEN NOT MATCHED THEN INSERT")
	if err := statement.dialect.Quoter().JoinWrite(buf.Builder, append(columns, exprs.ColNames()...), ","); err != nil {
		return "", nil, err
	}
	write(")")

	if err := statement.genInsertValuesValues(buf, false, columns, args); err != nil {
		return "", nil, err
	}

	return buf.String(), buf.Args(), nil
}
