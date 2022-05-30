// Copyright 2019 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"errors"
	"fmt"
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

	columnStr := statement.ColumnStr()
	if len(statement.SelectStr) > 0 {
		columnStr = statement.SelectStr
	} else {
		if statement.JoinStr == "" {
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
	}

	if err := statement.ProcessIDParam(); err != nil {
		return "", nil, err
	}

	sqlStr, condArgs, err := statement.genSelectSQL(columnStr, true, true)
	if err != nil {
		return "", nil, err
	}
	args := append(statement.joinArgs, condArgs...)

	// for mssql and use limit
	qs := strings.Count(sqlStr, "?")
	if len(args)*2 == qs {
		args = append(args, args...)
	}

	return sqlStr, args, nil
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
	sumSelect := strings.Join(sumStrs, ", ")

	if err := statement.MergeConds(bean); err != nil {
		return "", nil, err
	}

	sqlStr, condArgs, err := statement.genSelectSQL(sumSelect, true, true)
	if err != nil {
		return "", nil, err
	}

	return sqlStr, append(statement.joinArgs, condArgs...), nil
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
		if len(statement.JoinStr) == 0 {
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

	sqlStr, condArgs, err := statement.genSelectSQL(columnStr, true, true)
	if err != nil {
		return "", nil, err
	}

	return sqlStr, append(statement.joinArgs, condArgs...), nil
}

// GenCountSQL generates the SQL for counting
func (statement *Statement) GenCountSQL(beans ...interface{}) (string, []interface{}, error) {
	if statement.RawSQL != "" {
		return statement.GenRawSQL(), statement.RawParams, nil
	}

	var condArgs []interface{}
	var err error
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

	sqlStr, condArgs, err := statement.genSelectSQL(subQuerySelect, false, false)
	if err != nil {
		return "", nil, err
	}

	if statement.GroupByStr != "" {
		sqlStr = fmt.Sprintf("SELECT %s FROM (%s) sub", selectSQL, sqlStr)
	}

	return sqlStr, append(statement.joinArgs, condArgs...), nil
}

func (statement *Statement) fromBuilder() *strings.Builder {
	var builder strings.Builder
	quote := statement.quote
	dialect := statement.dialect

	builder.WriteString(" FROM ")

	if dialect.URI().DBType == schemas.MSSQL && strings.Contains(statement.TableName(), "..") {
		builder.WriteString(statement.TableName())
	} else {
		builder.WriteString(quote(statement.TableName()))
	}

	if statement.TableAlias != "" {
		if dialect.URI().DBType == schemas.ORACLE {
			builder.WriteString(" ")
		} else {
			builder.WriteString(" AS ")
		}
		builder.WriteString(quote(statement.TableAlias))
	}
	if statement.JoinStr != "" {
		builder.WriteString(" ")
		builder.WriteString(statement.JoinStr)
	}
	return &builder
}

func (statement *Statement) genSelectSQL(columnStr string, needLimit, needOrderBy bool) (string, []interface{}, error) {
	var (
		distinct      string
		dialect       = statement.dialect
		fromStr       = statement.fromBuilder().String()
		top, whereStr string
		mssqlCondi    = builder.NewWriter()
	)

	if statement.IsDistinct && !strings.HasPrefix(columnStr, "count") {
		distinct = "DISTINCT "
	}

	condWriter := builder.NewWriter()
	if err := statement.cond.WriteTo(condWriter); err != nil {
		return "", nil, err
	}

	if condWriter.Len() > 0 {
		whereStr = " WHERE "
	}

	pLimitN := statement.LimitN
	if dialect.URI().DBType == schemas.MSSQL {
		if pLimitN != nil {
			LimitNValue := *pLimitN
			top = fmt.Sprintf("TOP %d ", LimitNValue)
		}
		if statement.Start > 0 {
			if statement.RefTable == nil {
				return "", nil, errors.New("Unsupported query limit without reference table")
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
			if statement.needTableName() {
				if len(statement.TableAlias) > 0 {
					column = fmt.Sprintf("%s.%s", statement.TableAlias, column)
				} else {
					column = fmt.Sprintf("%s.%s", statement.TableName(), column)
				}
			}

			orderByWriter := builder.NewWriter()
			if needOrderBy {
				if err := statement.WriteOrderBy(orderByWriter); err != nil {
					return "", nil, err
				}
			}

			if _, err := fmt.Fprintf(mssqlCondi, "(%s NOT IN (SELECT TOP %d %s%s%s",
				column, statement.Start, column, fromStr, whereStr); err != nil {
				return "", nil, err
			}
			if err := utils.WriteBuilder(mssqlCondi, condWriter, orderByWriter); err != nil {
				return "", nil, err
			}

			if err := statement.WriteGroupBy(mssqlCondi); err != nil {
				return "", nil, err
			}

			if _, err := fmt.Fprint(mssqlCondi, "))"); err != nil {
				return "", nil, err
			}
		}
	}

	buf := builder.NewWriter()
	fmt.Fprintf(buf, "SELECT %v%v%v%v%v", distinct, top, columnStr, fromStr, whereStr)
	if err := utils.WriteBuilder(buf, condWriter); err != nil {
		return "", nil, err
	}
	if mssqlCondi.Len() > 0 {
		if len(whereStr) > 0 {
			fmt.Fprint(buf, " AND ")
		} else {
			fmt.Fprint(buf, " WHERE ")
		}

		if err := utils.WriteBuilder(buf, mssqlCondi); err != nil {
			return "", nil, err
		}
	}

	if err := statement.WriteGroupBy(buf); err != nil {
		return "", nil, err
	}
	if statement.HavingStr != "" {
		fmt.Fprint(buf, " ", statement.HavingStr)
	}
	if needOrderBy {
		if err := statement.WriteOrderBy(buf); err != nil {
			return "", nil, err
		}
	}
	if needLimit {
		if dialect.URI().DBType != schemas.MSSQL && dialect.URI().DBType != schemas.ORACLE {
			if statement.Start > 0 {
				if pLimitN != nil {
					fmt.Fprintf(buf, " LIMIT %v OFFSET %v", *pLimitN, statement.Start)
				} else {
					fmt.Fprintf(buf, " LIMIT 0 OFFSET %v", statement.Start)
				}
			} else if pLimitN != nil {
				fmt.Fprint(buf, " LIMIT ", *pLimitN)
			}
		} else if dialect.URI().DBType == schemas.ORACLE {
			if pLimitN != nil {
				oldString := buf.String()
				buf.Reset()
				rawColStr := columnStr
				if rawColStr == "*" {
					rawColStr = "at.*"
				}
				fmt.Fprintf(buf, "SELECT %v FROM (SELECT %v,ROWNUM RN FROM (%v) at WHERE ROWNUM <= %d) aat WHERE RN > %d",
					columnStr, rawColStr, oldString, statement.Start+*pLimitN, statement.Start)
			}
		}
	}
	if statement.IsForUpdate {
		return dialect.ForUpdateSQL(buf.String()), buf.Args(), nil
	}

	return buf.String(), buf.Args(), nil
}

// GenExistSQL generates Exist SQL
func (statement *Statement) GenExistSQL(bean ...interface{}) (string, []interface{}, error) {
	if statement.RawSQL != "" {
		return statement.GenRawSQL(), statement.RawParams, nil
	}

	var joinStr string
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
	if len(statement.JoinStr) > 0 {
		joinStr = " " + statement.JoinStr + " "
	}

	buf := builder.NewWriter()
	if statement.dialect.URI().DBType == schemas.MSSQL {
		if _, err := fmt.Fprintf(buf, "SELECT TOP 1 * FROM %s%s", tableName, joinStr); err != nil {
			return "", nil, err
		}
		if statement.Conds().IsValid() {
			if _, err := fmt.Fprintf(buf, " WHERE "); err != nil {
				return "", nil, err
			}
			if err := statement.Conds().WriteTo(buf); err != nil {
				return "", nil, err
			}
		}
	} else if statement.dialect.URI().DBType == schemas.ORACLE {
		if _, err := fmt.Fprintf(buf, "SELECT * FROM %s%s WHERE ", tableName, joinStr); err != nil {
			return "", nil, err
		}
		if statement.Conds().IsValid() {
			if err := statement.Conds().WriteTo(buf); err != nil {
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
		if _, err := fmt.Fprintf(buf, "SELECT 1 FROM %s%s", tableName, joinStr); err != nil {
			return "", nil, err
		}
		if statement.Conds().IsValid() {
			if _, err := fmt.Fprintf(buf, " WHERE "); err != nil {
				return "", nil, err
			}
			if err := statement.Conds().WriteTo(buf); err != nil {
				return "", nil, err
			}
		}
		if _, err := fmt.Fprintf(buf, " LIMIT 1"); err != nil {
			return "", nil, err
		}
	}

	return buf.String(), buf.Args(), nil
}

// GenFindSQL generates Find SQL
func (statement *Statement) GenFindSQL(autoCond builder.Cond) (string, []interface{}, error) {
	if statement.RawSQL != "" {
		return statement.GenRawSQL(), statement.RawParams, nil
	}

	var sqlStr string
	var args []interface{}
	var err error

	if len(statement.TableName()) <= 0 {
		return "", nil, ErrTableNotFound
	}

	columnStr := statement.ColumnStr()
	if len(statement.SelectStr) > 0 {
		columnStr = statement.SelectStr
	} else {
		if statement.JoinStr == "" {
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
	}

	statement.cond = statement.cond.And(autoCond)

	sqlStr, condArgs, err := statement.genSelectSQL(columnStr, true, true)
	if err != nil {
		return "", nil, err
	}
	args = append(statement.joinArgs, condArgs...)
	// for mssql and use limit
	qs := strings.Count(sqlStr, "?")
	if len(args)*2 == qs {
		args = append(args, args...)
	}

	return sqlStr, args, nil
}
