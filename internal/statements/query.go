// Copyright 2019 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"xorm.io/builder"
	"xorm.io/xorm/internal/utils"
	"xorm.io/xorm/schemas"
)

// GenQuerySQL generate query SQL
func (statement *Statement) GenQuerySQL(sqlOrArgs ...interface{}) (string, []interface{}, error) {
	if len(sqlOrArgs) > 0 {
		return statement.ConvertSQLOrArgs(sqlOrArgs...)
	}

	if statement.RawSQL != "" {
		return statement.GenRawSQL(), statement.RawParams, nil
	}

	if len(statement.TableName()) <= 0 {
		return "", nil, ErrTableNotFound
	}

	if err := statement.ProcessIDParam(); err != nil {
		return "", nil, err
	}

	buf := builder.NewWriter()
	if err := statement.writeSelect(buf, statement.genSelectColumnStr(), true); err != nil {
		return "", nil, err
	}
	return buf.String(), buf.Args(), nil
}

// GenSumSQL generates sum SQL
func (statement *Statement) GenSumSQL(bean interface{}, columns ...string) (string, []interface{}, error) {
	if statement.RawSQL != "" {
		return statement.GenRawSQL(), statement.RawParams, nil
	}

	if err := statement.SetRefBean(bean); err != nil {
		return "", nil, err
	}

	sumStrs := make([]string, 0, len(columns))
	for _, colName := range columns {
		if !strings.Contains(colName, " ") && !strings.Contains(colName, "(") {
			colName = statement.quote(colName)
		} else {
			colName = statement.ReplaceQuote(colName)
		}
		sumStrs = append(sumStrs, fmt.Sprintf("COALESCE(sum(%s),0)", colName))
	}

	if err := statement.MergeConds(bean); err != nil {
		return "", nil, err
	}

	buf := builder.NewWriter()
	if err := statement.writeSelect(buf, strings.Join(sumStrs, ", "), true); err != nil {
		return "", nil, err
	}
	return buf.String(), buf.Args(), nil
}

// GenGetSQL generates Get SQL
func (statement *Statement) GenGetSQL(bean interface{}) (string, []interface{}, error) {
	var isStruct bool
	if bean != nil {
		v := rValue(bean)
		isStruct = v.Kind() == reflect.Struct
		if isStruct {
			if err := statement.SetRefBean(bean); err != nil {
				return "", nil, err
			}
		}
	}

	columnStr := statement.ColumnStr()
	if len(statement.SelectStr) > 0 {
		columnStr = statement.SelectStr
	} else {
		// TODO: always generate column names, not use * even if join
		if len(statement.joins) == 0 {
			if len(columnStr) == 0 {
				if len(statement.GroupByStr) > 0 {
					columnStr = statement.quoteColumnStr(statement.GroupByStr)
				} else {
					columnStr = statement.genColumnStr()
				}
			}
		} else {
			if len(columnStr) == 0 {
				if len(statement.GroupByStr) > 0 {
					columnStr = statement.quoteColumnStr(statement.GroupByStr)
				}
			}
		}
	}

	if len(columnStr) == 0 {
		columnStr = "*"
	}

	if isStruct {
		if err := statement.MergeConds(bean); err != nil {
			return "", nil, err
		}
	} else {
		if err := statement.ProcessIDParam(); err != nil {
			return "", nil, err
		}
	}

	buf := builder.NewWriter()
	if err := statement.writeSelect(buf, columnStr, true); err != nil {
		return "", nil, err
	}
	return buf.String(), buf.Args(), nil
}

// GenCountSQL generates the SQL for counting
func (statement *Statement) GenCountSQL(beans ...interface{}) (string, []interface{}, error) {
	if statement.RawSQL != "" {
		return statement.GenRawSQL(), statement.RawParams, nil
	}

	if len(beans) > 0 {
		if err := statement.SetRefBean(beans[0]); err != nil {
			return "", nil, err
		}
		if err := statement.MergeConds(beans[0]); err != nil {
			return "", nil, err
		}
	}

	selectSQL := statement.SelectStr
	if len(selectSQL) <= 0 {
		if statement.IsDistinct {
			selectSQL = fmt.Sprintf("count(DISTINCT %s)", statement.ColumnStr())
		} else if statement.ColumnStr() != "" {
			selectSQL = fmt.Sprintf("count(%s)", statement.ColumnStr())
		} else {
			selectSQL = "count(*)"
		}
	}
	var subQuerySelect string
	if statement.GroupByStr != "" {
		subQuerySelect = statement.GroupByStr
	} else {
		subQuerySelect = selectSQL
	}

	buf := builder.NewWriter()
	if statement.GroupByStr != "" {
		if _, err := fmt.Fprintf(buf, "SELECT %s FROM (", selectSQL); err != nil {
			return "", nil, err
		}
	}

	if err := statement.writeSelect(buf, subQuerySelect, false); err != nil {
		return "", nil, err
	}

	if statement.GroupByStr != "" {
		if _, err := fmt.Fprintf(buf, ") sub"); err != nil {
			return "", nil, err
		}
	}

	return buf.String(), buf.Args(), nil
}

func (statement *Statement) writeFrom(w *builder.BytesWriter) error {
	if _, err := fmt.Fprint(w, " FROM "); err != nil {
		return err
	}
	if err := statement.writeTableName(w); err != nil {
		return err
	}
	if err := statement.writeAlias(w); err != nil {
		return err
	}
	return statement.writeJoins(w)
}

func (statement *Statement) writeLimitOffset(w builder.Writer) error {
	if statement.Start > 0 {
		if statement.LimitN != nil {
			_, err := fmt.Fprintf(w, " LIMIT %v OFFSET %v", *statement.LimitN, statement.Start)
			return err
		}
		_, err := fmt.Fprintf(w, " LIMIT 0 OFFSET %v", statement.Start)
		return err
	}
	if statement.LimitN != nil {
		_, err := fmt.Fprint(w, " LIMIT ", *statement.LimitN)
		return err
	}
	// no limit statement
	return nil
}

func (statement *Statement) writeTop(w builder.Writer) error {
	if statement.dialect.URI().DBType != schemas.MSSQL {
		return nil
	}
	if statement.LimitN == nil {
		return nil
	}
	_, err := fmt.Fprintf(w, " TOP %d", *statement.LimitN)
	return err
}

func (statement *Statement) writeDistinct(w builder.Writer) error {
	if statement.IsDistinct && !strings.HasPrefix(statement.SelectStr, "count(") {
		_, err := fmt.Fprint(w, " DISTINCT")
		return err
	}
	return nil
}

func (statement *Statement) writeSelectColumns(w *builder.BytesWriter, columnStr string) error {
	if _, err := fmt.Fprintf(w, "SELECT "); err != nil {
		return err
	}
	if err := statement.writeDistinct(w); err != nil {
		return err
	}
	if err := statement.writeTop(w); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, " ", columnStr)
	return err
}

func (statement *Statement) writeWhere(w *builder.BytesWriter) error {
	if !statement.cond.IsValid() {
		return statement.writeMssqlPaginationCond(w)
	}
	if _, err := fmt.Fprint(w, " WHERE "); err != nil {
		return err
	}
	if err := statement.cond.WriteTo(statement.QuoteReplacer(w)); err != nil {
		return err
	}
	return statement.writeMssqlPaginationCond(w)
}

func (statement *Statement) writeForUpdate(w io.Writer) error {
	if !statement.IsForUpdate {
		return nil
	}

	if statement.dialect.URI().DBType != schemas.MYSQL {
		return errors.New("only support mysql for update")
	}
	_, err := fmt.Fprint(w, " FOR UPDATE")
	return err
}

func (statement *Statement) writeMssqlPaginationCond(w *builder.BytesWriter) error {
	if statement.RefTable == nil {
		return errors.New("unsupported query limit without reference table")
	}

	if statement.dialect.URI().DBType != schemas.MSSQL || statement.Start <= 0 {
		return nil
	}

	mssqlCondi := builder.NewWriter()
	if err := statement.writeTop(mssqlCondi); err != nil {
		return err
	}

	var column string
	if len(statement.RefTable.PKColumns()) == 0 {
		for _, index := range statement.RefTable.Indexes {
			if len(index.Cols) == 1 {
				column = index.Cols[0]
				break
			}
		}
		if len(column) == 0 {
			column = statement.RefTable.ColumnsSeq()[0]
		}
	} else {
		column = statement.RefTable.PKColumns()[0].Name
	}
	if statement.NeedTableName() {
		if len(statement.TableAlias) > 0 {
			column = fmt.Sprintf("%s.%s", statement.TableAlias, column)
		} else {
			column = fmt.Sprintf("%s.%s", statement.TableName(), column)
		}
	}
	if _, err := fmt.Fprintf(mssqlCondi, "(%s NOT IN (SELECT TOP %d %s",
		column, statement.Start, column); err != nil {
		return err
	}
	if err := statement.writeFrom(mssqlCondi); err != nil {
		return err
	}
	if err := statement.writeWhere(mssqlCondi); err != nil {
		return err
	}
	if err := statement.WriteOrderBy(mssqlCondi); err != nil {
		return err
	}
	if err := statement.writeGroupBy(mssqlCondi); err != nil {
		return err
	}
	if _, err := fmt.Fprint(mssqlCondi, "))"); err != nil {
		return err
	}

	if statement.cond.IsValid() {
		if _, err := fmt.Fprint(w, " AND "); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprint(w, " WHERE "); err != nil {
			return err
		}
	}

	return utils.WriteBuilder(w, mssqlCondi)
}

func (statement *Statement) writeOracleLimit(w *builder.BytesWriter, columnStr string) error {
	if statement.LimitN == nil {
		return nil
	}

	oldString := w.String()
	w.Reset()
	rawColStr := columnStr
	if rawColStr == "*" {
		rawColStr = "at.*"
	}
	_, err := fmt.Fprintf(w, "SELECT %v FROM (SELECT %v,ROWNUM RN FROM (%v) at WHERE ROWNUM <= %d) aat WHERE RN > %d",
		columnStr, rawColStr, oldString, statement.Start+*statement.LimitN, statement.Start)
	return err
}

func (statement *Statement) writeSelect(buf *builder.BytesWriter, columnStr string, needLimit bool) error {
	if err := statement.writeSelectColumns(buf, columnStr); err != nil {
		return err
	}
	if err := statement.writeFrom(buf); err != nil {
		return err
	}
	if err := statement.writeWhere(buf); err != nil {
		return err
	}
	if err := statement.writeGroupBy(buf); err != nil {
		return err
	}
	if err := statement.writeHaving(buf); err != nil {
		return err
	}
	if err := statement.WriteOrderBy(buf); err != nil {
		return err
	}

	dialect := statement.dialect
	if needLimit {
		if dialect.URI().DBType == schemas.ORACLE {
			if err := statement.writeOracleLimit(buf, columnStr); err != nil {
				return err
			}
		} else if dialect.URI().DBType != schemas.MSSQL {
			if err := statement.writeLimitOffset(buf); err != nil {
				return err
			}
		}
	}
	return statement.writeForUpdate(buf)
}

// GenExistSQL generates Exist SQL
func (statement *Statement) GenExistSQL(bean ...interface{}) (string, []interface{}, error) {
	if statement.RawSQL != "" {
		return statement.GenRawSQL(), statement.RawParams, nil
	}

	var b interface{}
	if len(bean) > 0 {
		b = bean[0]
		beanValue := reflect.ValueOf(bean[0])
		if beanValue.Kind() != reflect.Ptr {
			return "", nil, errors.New("needs a pointer")
		}

		if beanValue.Elem().Kind() == reflect.Struct {
			if err := statement.SetRefBean(bean[0]); err != nil {
				return "", nil, err
			}
		}
	}
	tableName := statement.TableName()
	if len(tableName) <= 0 {
		return "", nil, ErrTableNotFound
	}
	if statement.RefTable != nil {
		return statement.Limit(1).GenGetSQL(b)
	}

	tableName = statement.quote(tableName)

	buf := builder.NewWriter()
	if statement.dialect.URI().DBType == schemas.MSSQL {
		if _, err := fmt.Fprintf(buf, "SELECT TOP 1 * FROM %s", tableName); err != nil {
			return "", nil, err
		}
		if err := statement.writeJoins(buf); err != nil {
			return "", nil, err
		}
		if statement.Conds().IsValid() {
			if _, err := fmt.Fprintf(buf, " WHERE "); err != nil {
				return "", nil, err
			}
			if err := statement.Conds().WriteTo(statement.QuoteReplacer(buf)); err != nil {
				return "", nil, err
			}
		}
	} else if statement.dialect.URI().DBType == schemas.ORACLE {
		if _, err := fmt.Fprintf(buf, "SELECT * FROM %s", tableName); err != nil {
			return "", nil, err
		}
		if err := statement.writeJoins(buf); err != nil {
			return "", nil, err
		}
		if _, err := fmt.Fprintf(buf, " WHERE "); err != nil {
			return "", nil, err
		}
		if statement.Conds().IsValid() {
			if err := statement.Conds().WriteTo(statement.QuoteReplacer(buf)); err != nil {
				return "", nil, err
			}
			if _, err := fmt.Fprintf(buf, " AND "); err != nil {
				return "", nil, err
			}
		}
		if _, err := fmt.Fprintf(buf, "ROWNUM=1"); err != nil {
			return "", nil, err
		}
	} else {
		if _, err := fmt.Fprintf(buf, "SELECT 1 FROM %s", tableName); err != nil {
			return "", nil, err
		}
		if err := statement.writeJoins(buf); err != nil {
			return "", nil, err
		}
		if statement.Conds().IsValid() {
			if _, err := fmt.Fprintf(buf, " WHERE "); err != nil {
				return "", nil, err
			}
			if err := statement.Conds().WriteTo(statement.QuoteReplacer(buf)); err != nil {
				return "", nil, err
			}
		}
		if _, err := fmt.Fprintf(buf, " LIMIT 1"); err != nil {
			return "", nil, err
		}
	}

	return buf.String(), buf.Args(), nil
}

func (statement *Statement) genSelectColumnStr() string {
	if len(statement.SelectStr) > 0 {
		return statement.SelectStr
	}

	columnStr := statement.ColumnStr()
	if len(statement.joins) == 0 {
		if columnStr == "" {
			if statement.GroupByStr != "" {
				columnStr = statement.quoteColumnStr(statement.GroupByStr)
			} else {
				columnStr = statement.genColumnStr()
			}
		}
	} else {
		if columnStr == "" {
			if statement.GroupByStr != "" {
				columnStr = statement.quoteColumnStr(statement.GroupByStr)
			} else {
				columnStr = "*"
			}
		}
	}
	if columnStr == "" {
		columnStr = "*"
	}
	return columnStr
}

// GenFindSQL generates Find SQL
func (statement *Statement) GenFindSQL(autoCond builder.Cond) (string, []interface{}, error) {
	if statement.RawSQL != "" {
		return statement.GenRawSQL(), statement.RawParams, nil
	}

	if len(statement.TableName()) <= 0 {
		return "", nil, ErrTableNotFound
	}

	statement.cond = statement.cond.And(autoCond)

	buf := builder.NewWriter()
	if err := statement.writeSelect(buf, statement.genSelectColumnStr(), true); err != nil {
		return "", nil, err
	}
	return buf.String(), buf.Args(), nil
}
