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

	buf := builder.NewWriter()
	if statement.GroupByStr != "" {
		if _, err := fmt.Fprintf(buf, "SELECT %s FROM (", selectSQL); err != nil {
			return "", nil, err
		}
	}

	var subQuerySelect string
	if statement.GroupByStr != "" {
		subQuerySelect = statement.GroupByStr
	} else {
		subQuerySelect = selectSQL
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
		_, err := fmt.Fprintf(w, " OFFSET %v", statement.Start)
		return err
	}
	if statement.LimitN != nil {
		_, err := fmt.Fprint(w, " LIMIT ", *statement.LimitN)
		return err
	}
	// no limit statement
	return nil
}

func (statement *Statement) writeOffsetFetch(w builder.Writer) error {
	if statement.LimitN != nil {
		_, err := fmt.Fprintf(w, " OFFSET %v ROWS FETCH NEXT %v ROWS ONLY", statement.Start, *statement.LimitN)
		return err
	}
	if statement.Start > 0 {
		_, err := fmt.Fprintf(w, " OFFSET %v ROWS", statement.Start)
		return err
	}
	return nil
}

// write "TOP <n>" (mssql only)
func (statement *Statement) writeTop(w builder.Writer) error {
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
	if _, err := fmt.Fprintf(w, "SELECT"); err != nil {
		return err
	}
	if err := statement.writeDistinct(w); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, " ", columnStr)
	return err
}

func (statement *Statement) writeWhereCond(w *builder.BytesWriter, cond builder.Cond) error {
	if !cond.IsValid() {
		return nil
	}

	if _, err := fmt.Fprint(w, " WHERE "); err != nil {
		return err
	}
	return cond.WriteTo(statement.QuoteReplacer(w))
}

func (statement *Statement) writeWhere(w *builder.BytesWriter) error {
	return statement.writeWhereCond(w, statement.cond)
}

func (statement *Statement) writeWhereWithMssqlPagination(w *builder.BytesWriter) error {
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

// write subquery to implement limit offset
// (mssql legacy only)
func (statement *Statement) writeMssqlPaginationCond(w *builder.BytesWriter) error {
	if statement.Start <= 0 {
		return nil
	}

	if statement.RefTable == nil {
		return errors.New("unsupported query limit without reference table")
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

	subWriter := builder.NewWriter()
	if _, err := fmt.Fprintf(subWriter, "(%s NOT IN (SELECT TOP %d %s",
		column, statement.Start, column); err != nil {
		return err
	}
	if err := statement.writeFrom(subWriter); err != nil {
		return err
	}
	if err := statement.writeWhere(subWriter); err != nil {
		return err
	}
	if err := statement.writeOrderBys(subWriter); err != nil {
		return err
	}
	if err := statement.writeGroupBy(subWriter); err != nil {
		return err
	}
	if _, err := fmt.Fprint(subWriter, "))"); err != nil {
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

	return utils.WriteBuilder(w, subWriter)
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
	dbType := statement.dialect.URI().DBType
	if statement.isUsingLegacyLimitOffset() {
		if dbType == "mssql" {
			return statement.writeMssqlLegacySelect(buf, columnStr)
		}
		if dbType == "oracle" {
			return statement.writeOracleLegacySelect(buf, columnStr)
		}
	}
	// TODO: modify all functions to func(w builder.Writer) error
	writeFns := []func(*builder.BytesWriter) error{
		func(bw *builder.BytesWriter) error { return statement.writeSelectColumns(bw, columnStr) },
		statement.writeFrom,
		statement.writeWhere,
		func(bw *builder.BytesWriter) error { return statement.writeGroupBy(bw) },
		func(bw *builder.BytesWriter) error { return statement.writeHaving(bw) },
		func(bw *builder.BytesWriter) (err error) {
			if dbType == "mssql" && len(statement.orderBy) == 0 && needLimit {
				// ORDER BY is mandatory to use OFFSET and FETCH clause (only in sqlserver)
				if statement.LimitN == nil && statement.Start == 0 {
					// no need to add
					return
				}
				if statement.IsDistinct || len(statement.GroupByStr) > 0 {
					// the order-by column should be one of distincts or group-bys
					// order by the first column
					_, err = bw.WriteString(" ORDER BY 1 ASC")
					return
				}
				if statement.RefTable == nil || len(statement.RefTable.PrimaryKeys) != 1 {
					// no primary key, order by the first column
					_, err = bw.WriteString(" ORDER BY 1 ASC")
					return
				}
				// order by primary key
				statement.orderBy = []orderBy{{orderStr: statement.colName(statement.RefTable.GetColumn(statement.RefTable.PrimaryKeys[0]), statement.TableName()), direction: "ASC"}}
			}
			return statement.writeOrderBys(bw)
		},
		func(bw *builder.BytesWriter) error {
			if dbType == "mssql" || dbType == "oracle" {
				return statement.writeOffsetFetch(bw)
			}
			return statement.writeLimitOffset(bw)
		},
		func(bw *builder.BytesWriter) error { return statement.writeForUpdate(bw) },
	}
	return statement.writeSelectWithFns(buf, writeFns...)
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
		if err := statement.writeWhere(buf); err != nil {
			return "", nil, err
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
		if err := statement.writeWhere(buf); err != nil {
			return "", nil, err
		}
		if _, err := fmt.Fprintf(buf, " LIMIT 1"); err != nil {
			return "", nil, err
		}
	}

	return buf.String(), buf.Args(), nil
}

func (statement *Statement) genSelectColumnStr() string {
	// manually select columns
	if len(statement.SelectStr) > 0 {
		return statement.SelectStr
	}

	columnStr := statement.ColumnStr()
	if columnStr != "" {
		return columnStr
	}

	// autodetect columns
	if statement.GroupByStr != "" {
		return statement.quoteColumnStr(statement.GroupByStr)
	}

	if len(statement.joins) != 0 {
		return "*"
	}

	columnStr = statement.genColumnStr()
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
