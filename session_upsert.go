package xorm

import (
	"database/sql"
	"fmt"
	"reflect"

	"xorm.io/xorm/convert"
	"xorm.io/xorm/internal/utils"
	"xorm.io/xorm/schemas"
)

func (session *Session) InsertOnConflictDoNothing(beans ...interface{}) (int64, error) {
	return session.upsert(false, beans...)
}

func (session *Session) Upsert(beans ...interface{}) (int64, error) {
	return session.upsert(true, beans...)
}

func (session *Session) upsert(doUpdate bool, beans ...interface{}) (int64, error) {
	var affected int64
	var err error

	if session.isAutoClose {
		defer session.Close()
	}

	session.autoResetStatement = false
	defer func() {
		session.autoResetStatement = true
		session.resetStatement()
	}()
	for _, bean := range beans {
		var cnt int64
		var err error
		switch v := bean.(type) {
		case map[string]interface{}:
			cnt, err = session.upsertMapInterface(doUpdate, v)
		case []map[string]interface{}: // FIXME: handle multiple
			for _, m := range v {
				cnt, err := session.upsertMapInterface(doUpdate, m)
				if err != nil {
					return affected, err
				}
				affected += cnt
			}
		case map[string]string:
			cnt, err = session.upsertMapString(doUpdate, v)
		case []map[string]string: // FIXME: handle multiple
			for _, m := range v {
				cnt, err := session.upsertMapString(doUpdate, m)
				if err != nil {
					return affected, err
				}
				affected += cnt
			}
		default:
			sliceValue := reflect.Indirect(reflect.ValueOf(bean))
			if sliceValue.Kind() == reflect.Slice { // FIXME: handle multiple
				if sliceValue.Len() <= 0 {
					return 0, ErrNoElementsOnSlice
				}
				for i := 0; i < sliceValue.Len(); i++ {
					v := sliceValue.Index(i)
					bean := v.Interface()
					cnt, err := session.upsertStruct(doUpdate, bean)
					if err != nil {
						return affected, err
					}
					affected += cnt
				}
			} else {
				cnt, err = session.upsertStruct(doUpdate, bean)
			}
		}
		if err != nil {
			return affected, err
		}
		affected += cnt
	}

	return affected, err
}

func (session *Session) upsertMapInterface(doUpdate bool, m map[string]interface{}) (int64, error) {
	if len(m) == 0 {
		return 0, ErrParamsType
	}

	tableName := session.statement.TableName()
	if len(tableName) == 0 {
		return 0, ErrTableNotFound
	}

	columns, args := utils.MapToSlices(m, session.statement.ExprColumns.ColNamesTrim(), schemas.CommonQuoter.Trim)
	return session.upsertMap(doUpdate, columns, args)
}

func (session *Session) upsertMapString(doUpdate bool, m map[string]string) (int64, error) {
	if len(m) == 0 {
		return 0, ErrParamsType
	}

	tableName := session.statement.TableName()
	if len(tableName) == 0 {
		return 0, ErrTableNotFound
	}

	columns, args := utils.MapStringToSlices(m, session.statement.ExprColumns.ColNamesTrim(), schemas.CommonQuoter.Trim)
	return session.upsertMap(doUpdate, columns, args)
}

func (session *Session) upsertMap(doUpdate bool, columns []string, args []interface{}) (int64, error) {
	tableName := session.statement.TableName()
	if len(tableName) == 0 {
		return 0, ErrTableNotFound
	}

	uniqueColValMap, err := session.getUniqueColumns(columns, args)
	if err != nil {
		return 0, err
	}

	sql, args, err := session.statement.GenUpsertMapSQL(doUpdate, columns, args, uniqueColValMap)
	if err != nil {
		return 0, err
	}
	sql = session.engine.dialect.Quoter().Replace(sql)

	if err := session.cacheInsert(tableName); err != nil {
		return 0, err
	}

	res, err := session.exec(sql, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (session *Session) upsertStruct(doUpdate bool, bean interface{}) (int64, error) {
	if err := session.statement.SetRefBean(bean); err != nil {
		return 0, err
	}
	if len(session.statement.TableName()) == 0 {
		return 0, ErrTableNotFound
	}

	// handle BeforeInsertProcessor
	for _, closure := range session.beforeClosures {
		closure(bean)
	}
	cleanupProcessorsClosures(&session.beforeClosures) // cleanup after used

	if processor, ok := interface{}(bean).(BeforeInsertProcessor); ok {
		processor.BeforeInsert()
	}

	tableName := session.statement.TableName()
	table := session.statement.RefTable

	colNames, args, err := session.genInsertColumns(bean)
	if err != nil {
		return 0, err
	}

	uniqueColValMap, err := session.getUniqueColumns(colNames, args)
	if err != nil {
		return 0, err
	}

	sqlStr, args, err := session.statement.GenUpsertSQL(doUpdate, colNames, args, uniqueColValMap)
	if err != nil {
		return 0, err
	}
	sqlStr = session.engine.dialect.Quoter().Replace(sqlStr)

	// if there is auto increment column and driver doesn't support return it
	if len(table.AutoIncrement) > 0 && !session.engine.driver.Features().SupportReturnInsertedID {
		n, err := session.execInsertSqlNoAutoReturn(sqlStr, bean, colNames, args)
		if err == sql.ErrNoRows {
			return n, nil
		}
		return n, err
	}

	res, err := session.exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}

	defer session.handleAfterInsertProcessorFunc(bean)

	_ = session.cacheInsert(tableName)

	if table.Version != "" && session.statement.CheckVersion {
		verValue, err := table.VersionColumn().ValueOf(bean)
		if err != nil {
			session.engine.logger.Errorf("%v", err)
		} else if verValue.IsValid() && verValue.CanSet() {
			session.incrVersionFieldValue(verValue)
		}
	}

	if table.AutoIncrement == "" {
		return res.RowsAffected()
	}

	n, err := res.RowsAffected()
	if err != nil || n == 0 {
		return 0, err
	}

	var id int64
	id, err = res.LastInsertId()
	if err != nil || id <= 0 {
		return n, err
	}

	aiValue, err := table.AutoIncrColumn().ValueOf(bean)
	if err != nil {
		session.engine.logger.Errorf("%v", err)
	}

	if aiValue == nil || !aiValue.IsValid() || !aiValue.CanSet() {
		return n, err
	}

	if err := convert.AssignValue(*aiValue, id); err != nil {
		return 0, err
	}

	return n, err
}

func (session *Session) getUniqueColumns(colNames []string, args []interface{}) (uniqueColValMap map[string]interface{}, err error) {
	uniqueColValMap = make(map[string]interface{})
	table := session.statement.RefTable
	if len(table.Indexes) == 0 {
		return nil, fmt.Errorf("provided bean has no unique constraints")
	}

	// Iterate across the indexes in the provided table
	for _, index := range table.Indexes {
		if index.Type != schemas.UniqueType {
			continue
		}

		// index is a Unique constraint
	indexCol:
		for _, indexColumnName := range index.Cols {
			if _, has := uniqueColValMap[indexColumnName]; has {
				// column is already included in uniqueCols so we don't need to add it again
				continue indexCol
			}

			// Now iterate across colNames and add to the uniqueCols
			for i, col := range colNames {
				if col == indexColumnName {
					uniqueColValMap[col] = args[i]
					continue indexCol
				}
			}

			indexColumn := table.GetColumn(indexColumnName)
			if !indexColumn.DefaultIsEmpty {
				uniqueColValMap[indexColumnName] = indexColumn.Default
			}

			if indexColumn.MapType == schemas.ONLYFROMDB || indexColumn.IsAutoIncrement {
				continue indexCol
			}
			// FIXME: what do we do here?!
			if session.statement.OmitColumnMap.Contain(indexColumn.Name) {
				continue indexCol
			}
			// FIXME: what do we do here?!
			if len(session.statement.ColumnMap) > 0 && !session.statement.ColumnMap.Contain(indexColumn.Name) {
				continue indexCol
			}
			// FIXME: what do we do here?!
			if session.statement.IncrColumns.IsColExist(indexColumn.Name) {
				for _, exprCol := range session.statement.IncrColumns {
					if exprCol.ColName == indexColumn.Name {
						uniqueColValMap[indexColumnName] = exprCol.Arg
					}
				}
				continue indexCol
			} else if session.statement.DecrColumns.IsColExist(indexColumn.Name) {
				for _, exprCol := range session.statement.DecrColumns {
					if exprCol.ColName == indexColumn.Name {
						uniqueColValMap[indexColumnName] = exprCol.Arg
					}
				}
				continue indexCol
			} else if session.statement.ExprColumns.IsColExist(indexColumn.Name) {
				for _, exprCol := range session.statement.ExprColumns {
					if exprCol.ColName == indexColumn.Name {
						uniqueColValMap[indexColumnName] = exprCol.Arg
					}
				}
			}

			// FIXME: not sure if there's anything else we can do
			return nil, fmt.Errorf("provided bean does not provide a value for unique constraint %s field %s which has no default", index.Name, indexColumnName)
		}
	}
	return uniqueColValMap, nil
}
