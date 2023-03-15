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

	fmt.Println(session.statement.TableName())
	for _, bean := range beans {
		var cnt int64
		var err error
		switch v := bean.(type) {
		case map[string]interface{}:
			cnt, err = session.upsertMapInterface(doUpdate, v)
		case []map[string]interface{}: // FIXME: handle multiple?
			for _, m := range v {
				cnt, err := session.upsertMapInterface(doUpdate, m)
				if err != nil {
					return affected, err
				}
				affected += cnt
			}
		case map[string]string:
			cnt, err = session.upsertMapString(doUpdate, v)
		case []map[string]string: // FIXME: handle multiple?
			for _, m := range v {
				cnt, err := session.upsertMapString(doUpdate, m)
				if err != nil {
					return affected, err
				}
				affected += cnt
			}
		default:
			sliceValue := reflect.Indirect(reflect.ValueOf(bean))
			if sliceValue.Kind() == reflect.Slice { // FIXME: handle multiple?
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
	if session.statement.RefTable == nil {
		return 0, ErrTableNotFound
	}

	uniqueColValMap, uniqueConstraints, err := session.getUniqueColumns(doUpdate, columns, args)
	if err != nil {
		return 0, err
	}

	sql, args, err := session.statement.GenUpsertSQL(doUpdate, false, columns, args, uniqueColValMap, uniqueConstraints)
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
	if doUpdate && session.engine.dialect.URI().DBType == schemas.MYSQL && affected == 2 {
		// for MYSQL if INSERT ... ON CONFLICT RowsAffected == 2 means UPDATE
		affected = 1
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
	// For the moment we're going to disable Conds for upsert as I'm not certain how best to implement those
	if doUpdate && (len(session.statement.ExprColumns) > 0 || session.statement.Conds().IsValid()) {
		return 0, ErrConditionType
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

	uniqueColValMap, uniqueConstraints, err := session.getUniqueColumns(doUpdate, colNames, args)
	if err != nil {
		return 0, err
	}

	sqlStr, args, err := session.statement.GenUpsertSQL(doUpdate, true, colNames, args, uniqueColValMap, uniqueConstraints)
	if err != nil {
		return 0, err
	}
	sqlStr = session.engine.dialect.Quoter().Replace(sqlStr)

	// if there is auto increment column and driver doesn't support return it
	if len(table.AutoIncrement) > 0 && (!session.engine.driver.Features().SupportReturnInsertedID || session.engine.dialect.URI().DBType == schemas.SQLITE) {
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
	n, err := res.RowsAffected()
	if err != nil || n == 0 {
		return 0, err
	}

	if session.engine.dialect.URI().DBType == schemas.MYSQL && n == 2 {
		// for MYSQL if INSERT ... ON CONFLICT RowsAffected == 2 means UPDATE
		n = 1
	}

	if table.AutoIncrement == "" {
		return n, nil
	}

	id, err := res.LastInsertId()
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

var (
	ErrNoUniqueConstraints       = fmt.Errorf("provided bean has no unique constraints")
	ErrMultipleUniqueConstraints = fmt.Errorf("cannot upsert if there is more than one unique constraint tested")
)

func (session *Session) getUniqueColumns(doUpdate bool, argColumns []string, args []interface{}) (uniqueColValMap map[string]interface{}, constraints [][]string, err error) {
	// We need to collect the constraints that are being "tested" by argColumns as compared to the table
	//
	// There are two cases:
	//
	// 1. Insert on conflict do nothing
	// 2. Upsert
	//
	// If we are an "Insert on conflict do nothing" then more than one "constraint" can be tested.
	// If we are an "Upsert" only one "constraint" can be tested.
	//
	// In Xorm the only constraints we know of are "Unique Indices" and "Primary Keys".
	//
	// For unique indices - every column in the unique index is being tested.
	//
	// If the primary key is a single column and it is autoincrement then an empty PK column is not testing an unique constraint
	// otherwise it does count.

	uniqueColValMap = make(map[string]interface{})
	table := session.statement.RefTable
	// Shortcut when there are no indices and no primary keys
	if len(table.Indexes) == 0 && len(table.PrimaryKeys) == 0 {
		return nil, nil, ErrNoUniqueConstraints
	}

	numberOfUniqueConstraints := 0

	// Check the primary key:
	switch len(table.PrimaryKeys) {
	case 0:
		// No primary keys - nothing to do
	case 1:
		// check if the pkColumn is included
		value := session.getUniqueColumnValue(table.PrimaryKeys[0], argColumns, args)
		if value != nil {
			numberOfUniqueConstraints++
			uniqueColValMap[table.PrimaryKeys[0]] = value
			constraints = append(constraints, table.PrimaryKeys)
		}
	default:
		numberOfUniqueConstraints++
		constraints = append(constraints, table.PrimaryKeys)
		for _, column := range table.PrimaryKeys {
			value := session.getUniqueColumnValue(column, argColumns, args)
			if value == nil {
				value = "" // default to empty
			}
			uniqueColValMap[column] = value
		}
	}

	// Iterate across the indexes in the provided table
	for _, index := range table.Indexes {
		if index.Type != schemas.UniqueType {
			continue
		}
		numberOfUniqueConstraints++
		constraints = append(constraints, index.Cols)

		// index is a Unique constraint
		for _, column := range index.Cols {
			if _, has := uniqueColValMap[column]; has {
				continue
			}

			value := session.getUniqueColumnValue(column, argColumns, args)
			if value == nil {
				value = "" // default to empty
			}
			uniqueColValMap[column] = value
		}
	}
	if doUpdate && numberOfUniqueConstraints > 1 {
		return nil, nil, ErrMultipleUniqueConstraints
	}
	if len(constraints) == 0 {
		return nil, nil, ErrNoUniqueConstraints
	}

	return uniqueColValMap, constraints, nil
}

func (session *Session) getUniqueColumnValue(indexColumnName string, argColumns []string, args []interface{}) (value interface{}) {
	table := session.statement.RefTable

	// Now iterate across colNames and add to the uniqueCols
	for i, col := range argColumns {
		if col == indexColumnName {
			return args[i]
		}
	}

	indexColumn := table.GetColumn(indexColumnName)
	if indexColumn.IsAutoIncrement {
		return nil
	}

	if !indexColumn.DefaultIsEmpty {
		value = indexColumn.Default
	}

	if indexColumn.MapType == schemas.ONLYFROMDB {
		return value
	}
	// FIXME: what do we do here?!
	if session.statement.OmitColumnMap.Contain(indexColumn.Name) {
		return value
	}
	// FIXME: what do we do here?!
	if len(session.statement.ColumnMap) > 0 && !session.statement.ColumnMap.Contain(indexColumn.Name) {
		return value
	}
	// FIXME: what do we do here?!
	if session.statement.IncrColumns.IsColExist(indexColumn.Name) {
		for _, exprCol := range session.statement.IncrColumns {
			if exprCol.ColName == indexColumn.Name {
				return exprCol.Arg
			}
		}
		return value
	} else if session.statement.DecrColumns.IsColExist(indexColumn.Name) {
		for _, exprCol := range session.statement.DecrColumns {
			if exprCol.ColName == indexColumn.Name {
				return exprCol.Arg
			}
		}
		return value
	} else if session.statement.ExprColumns.IsColExist(indexColumn.Name) {
		for _, exprCol := range session.statement.ExprColumns {
			if exprCol.ColName == indexColumn.Name {
				return exprCol.Arg
			}
		}
	}

	// FIXME: not sure if there's anything else we can do
	return value
}
