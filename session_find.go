// Copyright 2016 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"

	"xorm.io/builder"

	"xorm.io/xorm/v2/convert"
	"xorm.io/xorm/v2/internal/utils"
	"xorm.io/xorm/v2/schemas"
)

const (
	tpStruct = iota
	tpNonStruct
)

// Find retrieve records from table, condiBeans's non-empty fields
// are conditions. beans could be []Struct, []*Struct, map[int64]Struct
// map[int64]*Struct
func (session *Session) Find(rowsSlicePtr interface{}, condiBean ...interface{}) error {
	if session.isAutoClose {
		defer session.Close()
	}
	return session.find(rowsSlicePtr, condiBean...)
}

// FindAndCount find the results and also return the counts
func (session *Session) FindAndCount(rowsSlicePtr interface{}, condiBean ...interface{}) (int64, error) {
	if session.isAutoClose {
		defer session.Close()
	}

	session.autoResetStatement = false
	err := session.find(rowsSlicePtr, condiBean...)
	if err != nil {
		return 0, err
	}

	sliceValue := reflect.Indirect(reflect.ValueOf(rowsSlicePtr))
	if sliceValue.Kind() != reflect.Slice && sliceValue.Kind() != reflect.Map {
		return 0, errors.New("needs a pointer to a slice or a map")
	}

	sliceElementType := sliceValue.Type().Elem()
	if sliceElementType.Kind() == reflect.Ptr {
		sliceElementType = sliceElementType.Elem()
	}
	session.autoResetStatement = true

	if session.statement.SelectStr != "" {
		session.statement.SelectStr = ""
	}
	if len(session.statement.ColumnMap) > 0 && !session.statement.IsDistinct {
		session.statement.ColumnMap = []string{}
	}
	session.statement.ResetOrderBy()
	if session.statement.LimitN != nil {
		session.statement.LimitN = nil
	}
	if session.statement.Start > 0 {
		session.statement.Start = 0
	}

	// session has stored the conditions so we use `unscoped` to avoid duplicated condition.
	if sliceElementType.Kind() == reflect.Struct {
		return session.Unscoped().Count(reflect.New(sliceElementType).Interface())
	}

	return session.Unscoped().Count()
}

func (session *Session) find(rowsSlicePtr interface{}, condiBean ...interface{}) error {
	defer session.resetStatement()
	if session.statement.LastError != nil {
		return session.statement.LastError
	}

	sliceValue := reflect.Indirect(reflect.ValueOf(rowsSlicePtr))
	isSlice := sliceValue.Kind() == reflect.Slice
	isMap := sliceValue.Kind() == reflect.Map
	if !isSlice && !isMap {
		return errors.New("needs a pointer to a slice or a map")
	}

	sliceElementType := sliceValue.Type().Elem()

	tp := tpStruct
	if session.statement.RefTable == nil {
		if sliceElementType.Kind() == reflect.Ptr {
			if sliceElementType.Elem().Kind() == reflect.Struct {
				pv := reflect.New(sliceElementType.Elem())
				if err := session.statement.SetRefValue(pv); err != nil {
					return err
				}
			} else {
				tp = tpNonStruct
			}
		} else if sliceElementType.Kind() == reflect.Struct {
			pv := reflect.New(sliceElementType)
			if err := session.statement.SetRefValue(pv); err != nil {
				return err
			}
		} else {
			tp = tpNonStruct
		}
	}

	var (
		table          = session.statement.RefTable
		addedTableName = session.statement.NeedTableName()
		autoCond       builder.Cond
	)
	if tp == tpStruct {
		if !session.statement.NoAutoCondition && len(condiBean) > 0 {
			condTable, err := session.engine.tagParser.Parse(reflect.ValueOf(condiBean[0]))
			if err != nil {
				return err
			}
			autoCond, err = session.statement.BuildConds(condTable, condiBean[0], true, true, false, true, addedTableName)
			if err != nil {
				return err
			}
		} else {
			if col := table.DeletedColumn(); col != nil && !session.statement.GetUnscoped() { // tag "deleted" is enabled
				autoCond = session.statement.CondDeleted(col)
			}
		}
	}

	// if it's a map with Cols but primary key not in column list, we still need the primary key
	if isMap && !session.statement.ColumnMap.IsEmpty() {
		for _, k := range session.statement.RefTable.PrimaryKeys {
			session.statement.ColumnMap.Add(k)
		}
	}

	sqlStr, args, err := session.statement.GenFindSQL(autoCond)
	if err != nil {
		return err
	}

	return session.noCacheFind(table, sliceValue, sqlStr, args...)
}

type QueryedField struct {
	FieldName      string
	LowerFieldName string
	ColumnType     *sql.ColumnType
	TempIndex      int
	ColumnSchema   *schemas.Column
}

type ColumnsSchema struct {
	Fields     []*QueryedField
	FieldNames []string
	Types      []*sql.ColumnType
}

func (columnsSchema *ColumnsSchema) ParseTableSchema(table *schemas.Table) {
	for _, field := range columnsSchema.Fields {
		field.ColumnSchema = table.GetColumnIdx(field.FieldName, field.TempIndex)
	}
}

func ParseColumnsSchema(fieldNames []string, types []*sql.ColumnType, table *schemas.Table) *ColumnsSchema {
	var columnsSchema ColumnsSchema

	fields := make([]*QueryedField, 0, len(fieldNames))

	for i, fieldName := range fieldNames {
		field := &QueryedField{
			FieldName:      fieldName,
			LowerFieldName: strings.ToLower(fieldName),
			ColumnType:     types[i],
		}
		fields = append(fields, field)
	}

	columnsSchema.Fields = fields

	tempMap := make(map[string]int)
	for _, field := range fields {
		var idx int
		var ok bool

		if idx, ok = tempMap[field.LowerFieldName]; !ok {
			idx = 0
		} else {
			idx++
		}

		tempMap[field.LowerFieldName] = idx
		field.TempIndex = idx
	}

	if table != nil {
		columnsSchema.ParseTableSchema(table)
	}

	return &columnsSchema
}

func (session *Session) noCacheFind(table *schemas.Table, containerValue reflect.Value, sqlStr string, args ...interface{}) error {
	elemType := containerValue.Type().Elem()
	var isPointer bool
	if elemType.Kind() == reflect.Ptr {
		isPointer = true
		elemType = elemType.Elem()
	}
	if elemType.Kind() == reflect.Ptr {
		return errors.New("pointer to pointer is not supported")
	}

	rows, err := session.queryRows(sqlStr, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	fields, err := rows.Columns()
	if err != nil {
		return err
	}

	types, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	newElemFunc := func(fields []string) reflect.Value {
		return utils.New(elemType, len(fields), len(fields))
	}

	var containerValueSetFunc func(*reflect.Value, schemas.PK) error

	if containerValue.Kind() == reflect.Slice {
		containerValueSetFunc = func(newValue *reflect.Value, pk schemas.PK) error {
			if isPointer {
				containerValue.Set(reflect.Append(containerValue, newValue.Elem().Addr()))
			} else {
				containerValue.Set(reflect.Append(containerValue, newValue.Elem()))
			}
			return nil
		}
	} else {
		keyType := containerValue.Type().Key()
		if len(table.PrimaryKeys) == 0 {
			return errors.New("don't support multiple primary key's map has non-slice key type")
		}
		if len(table.PrimaryKeys) > 1 && keyType.Kind() != reflect.Slice {
			return errors.New("don't support multiple primary key's map has non-slice key type")
		}

		containerValueSetFunc = func(newValue *reflect.Value, pk schemas.PK) error {
			keyValue := reflect.New(keyType)
			cols := table.PKColumns()
			if len(cols) == 1 {
				if err := convert.AssignValue(keyValue, pk[0]); err != nil {
					return err
				}
			} else {
				keyValue.Set(reflect.ValueOf(&pk))
			}

			if isPointer {
				containerValue.SetMapIndex(keyValue.Elem(), newValue.Elem().Addr())
			} else {
				containerValue.SetMapIndex(keyValue.Elem(), newValue.Elem())
			}
			return nil
		}
	}

	if elemType.Kind() == reflect.Struct {
		newValue := newElemFunc(fields)
		tb, err := session.engine.tagParser.ParseWithCache(newValue)
		if err != nil {
			return err
		}

		columnsSchema := ParseColumnsSchema(fields, types, tb)

		err = session.rows2Beans(rows, columnsSchema, fields, types, tb, newElemFunc, containerValueSetFunc)
		rows.Close()
		if err != nil {
			return err
		}
		return session.executeProcessors()
	}

	for rows.Next() {
		newValue := newElemFunc(fields)
		bean := newValue.Interface()

		switch elemType.Kind() {
		case reflect.Slice:
			err = session.getSlice(rows, types, fields, bean)
		case reflect.Map:
			err = session.getMap(rows, types, fields, bean)
		default:
			err = rows.Scan(bean)
		}
		if err != nil {
			return err
		}

		if err := containerValueSetFunc(&newValue, nil); err != nil {
			return err
		}
	}
	return rows.Err()
}
