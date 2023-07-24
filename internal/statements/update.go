// Copyright 2017 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"time"

	"xorm.io/builder"
	"xorm.io/xorm/convert"
	"xorm.io/xorm/dialects"
	"xorm.io/xorm/internal/json"
	"xorm.io/xorm/internal/utils"
	"xorm.io/xorm/schemas"
)

func (statement *Statement) ifAddColUpdate(col *schemas.Column, includeVersion, includeUpdated, includeNil,
	includeAutoIncr, update bool,
) (bool, error) {
	columnMap := statement.ColumnMap
	omitColumnMap := statement.OmitColumnMap
	unscoped := statement.unscoped

	if !includeVersion && col.IsVersion {
		return false, nil
	}
	if col.IsCreated && !columnMap.Contain(col.Name) {
		return false, nil
	}
	if !includeUpdated && col.IsUpdated {
		return false, nil
	}
	if !includeAutoIncr && col.IsAutoIncrement {
		return false, nil
	}
	if col.IsDeleted && !unscoped {
		return false, nil
	}
	if omitColumnMap.Contain(col.Name) {
		return false, nil
	}
	if len(columnMap) > 0 && !columnMap.Contain(col.Name) {
		return false, nil
	}

	if col.MapType == schemas.ONLYFROMDB {
		return false, nil
	}

	if statement.IncrColumns.IsColExist(col.Name) {
		return false, nil
	} else if statement.DecrColumns.IsColExist(col.Name) {
		return false, nil
	} else if statement.ExprColumns.IsColExist(col.Name) {
		return false, nil
	}

	return true, nil
}

// BuildUpdates auto generating update columnes and values according a struct
func (statement *Statement) BuildUpdates(tableValue reflect.Value,
	includeVersion, includeUpdated, includeNil,
	includeAutoIncr, update bool,
) ([]string, []interface{}, error) {
	table := statement.RefTable
	allUseBool := statement.allUseBool
	useAllCols := statement.useAllCols
	mustColumnMap := statement.MustColumnMap
	nullableMap := statement.NullableMap

	colNames := make([]string, 0)
	args := make([]interface{}, 0)

	for _, col := range table.Columns() {
		ok, err := statement.ifAddColUpdate(col, includeVersion, includeUpdated, includeNil,
			includeAutoIncr, update)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			continue
		}

		fieldValuePtr, err := col.ValueOfV(&tableValue)
		if err != nil {
			return nil, nil, err
		}
		if fieldValuePtr == nil {
			continue
		}

		fieldValue := *fieldValuePtr
		fieldType := reflect.TypeOf(fieldValue.Interface())
		if fieldType == nil {
			continue
		}

		requiredField := useAllCols
		includeNil := useAllCols

		if b, ok := getFlagForColumn(mustColumnMap, col); ok {
			if b {
				requiredField = true
			} else {
				continue
			}
		}

		// !evalphobia! set fieldValue as nil when column is nullable and zero-value
		if b, ok := getFlagForColumn(nullableMap, col); ok {
			if b && col.Nullable && utils.IsZero(fieldValue.Interface()) {
				var nilValue *int
				fieldValue = reflect.ValueOf(nilValue)
				fieldType = reflect.TypeOf(fieldValue.Interface())
				includeNil = true
			}
		}

		var val interface{}

		if fieldValue.CanAddr() {
			if structConvert, ok := fieldValue.Addr().Interface().(convert.Conversion); ok {
				data, err := structConvert.ToDB()
				if err != nil {
					return nil, nil, err
				}
				if data != nil {
					val = data
					if !col.SQLType.IsBlob() {
						val = string(data)
					}
				}
				goto APPEND
			}
		}

		if structConvert, ok := fieldValue.Interface().(convert.Conversion); ok && !fieldValue.IsNil() {
			data, err := structConvert.ToDB()
			if err != nil {
				return nil, nil, err
			}
			if data != nil {
				val = data
				if !col.SQLType.IsBlob() {
					val = string(data)
				}
			}
			goto APPEND
		}

		if fieldType.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				if includeNil {
					args = append(args, nil)
					colNames = append(colNames, fmt.Sprintf("%v=?", statement.quote(col.Name)))
				}
				continue
			} else if !fieldValue.IsValid() {
				continue
			} else {
				// dereference ptr type to instance type
				fieldValue = fieldValue.Elem()
				fieldType = reflect.TypeOf(fieldValue.Interface())
				requiredField = true
			}
		}

		switch fieldType.Kind() {
		case reflect.Bool:
			if allUseBool || requiredField {
				val = fieldValue.Interface()
			} else {
				// if a bool in a struct, it will not be as a condition because it default is false,
				// please use Where() instead
				continue
			}
		case reflect.String:
			if !requiredField && fieldValue.String() == "" {
				continue
			}
			// for MyString, should convert to string or panic
			if fieldType.String() != reflect.String.String() {
				val = fieldValue.String()
			} else {
				val = fieldValue.Interface()
			}
		case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
			if !requiredField && fieldValue.Int() == 0 {
				continue
			}
			val = fieldValue.Interface()
		case reflect.Float32, reflect.Float64:
			if !requiredField && fieldValue.Float() == 0.0 {
				continue
			}
			val = fieldValue.Interface()
		case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
			if !requiredField && fieldValue.Uint() == 0 {
				continue
			}
			val = fieldValue.Interface()
		case reflect.Struct:
			if fieldType.ConvertibleTo(schemas.TimeType) {
				t := fieldValue.Convert(schemas.TimeType).Interface().(time.Time)
				if !requiredField && (t.IsZero() || !fieldValue.IsValid()) {
					continue
				}
				val, err = dialects.FormatColumnTime(statement.dialect, statement.defaultTimeZone, col, t)
				if err != nil {
					return nil, nil, err
				}
			} else if nulType, ok := fieldValue.Interface().(driver.Valuer); ok {
				val, _ = nulType.Value()
				if val == nil && !requiredField {
					continue
				}
			} else {
				if !col.IsJSON {
					table, err := statement.tagParser.ParseWithCache(fieldValue)
					if err != nil {
						val = fieldValue.Interface()
					} else {
						if len(table.PrimaryKeys) == 1 {
							pkField := reflect.Indirect(fieldValue).FieldByName(table.PKColumns()[0].FieldName)
							// fix non-int pk issues
							if pkField.IsValid() && (!requiredField && !utils.IsZero(pkField.Interface())) {
								val = pkField.Interface()
							} else {
								continue
							}
						} else {
							return nil, nil, errors.New("Not supported multiple primary keys")
						}
					}
				} else {
					// Blank struct could not be as update data
					if requiredField || !utils.IsStructZero(fieldValue) {
						bytes, err := json.DefaultJSONHandler.Marshal(fieldValue.Interface())
						if err != nil {
							return nil, nil, fmt.Errorf("mashal %v failed", fieldValue.Interface())
						}
						if col.SQLType.IsText() {
							val = string(bytes)
						} else if col.SQLType.IsBlob() {
							val = bytes
						}
					} else {
						continue
					}
				}
			}
		case reflect.Array, reflect.Slice, reflect.Map:
			if !requiredField {
				if fieldValue == reflect.Zero(fieldType) {
					continue
				}
				if fieldType.Kind() == reflect.Array {
					if utils.IsArrayZero(fieldValue) {
						continue
					}
				} else if fieldValue.IsNil() || !fieldValue.IsValid() || fieldValue.Len() == 0 {
					continue
				}
			}

			if col.SQLType.IsText() {
				bytes, err := json.DefaultJSONHandler.Marshal(fieldValue.Interface())
				if err != nil {
					return nil, nil, err
				}
				val = string(bytes)
			} else if col.SQLType.IsBlob() {
				var bytes []byte
				var err error
				if fieldType.Kind() == reflect.Slice &&
					fieldType.Elem().Kind() == reflect.Uint8 {
					if fieldValue.Len() > 0 {
						val = fieldValue.Bytes()
					} else {
						continue
					}
				} else if fieldType.Kind() == reflect.Array &&
					fieldType.Elem().Kind() == reflect.Uint8 {
					val = fieldValue.Slice(0, 0).Interface()
				} else {
					bytes, err = json.DefaultJSONHandler.Marshal(fieldValue.Interface())
					if err != nil {
						return nil, nil, err
					}
					val = bytes
				}
			} else {
				continue
			}
		default:
			val = fieldValue.Interface()
		}

	APPEND:
		args = append(args, val)
		colNames = append(colNames, fmt.Sprintf("%v = ?", statement.quote(col.Name)))
	}

	return colNames, args, nil
}

func (statement *Statement) writeUpdateTop(updateWriter *builder.BytesWriter) error {
	if statement.dialect.URI().DBType != schemas.MSSQL || statement.LimitN == nil {
		return nil
	}

	table := statement.RefTable
	if statement.HasOrderBy() && table != nil && len(table.PrimaryKeys) == 1 {
		return nil
	}

	_, err := fmt.Fprintf(updateWriter, " TOP (%d)", *statement.LimitN)
	return err
}

func (statement *Statement) writeUpdateTableName(updateWriter *builder.BytesWriter) error {
	tableName := statement.quote(statement.TableName())
	if statement.TableAlias == "" {
		_, err := fmt.Fprint(updateWriter, " ", tableName)
		return err
	}

	switch statement.dialect.URI().DBType {
	case schemas.MSSQL:
		_, err := fmt.Fprint(updateWriter, " ", statement.TableAlias)
		return err
	default:
		_, err := fmt.Fprint(updateWriter, " ", tableName, " AS ", statement.TableAlias)
		return err
	}
}

func (statement *Statement) writeUpdateFrom(updateWriter *builder.BytesWriter) error {
	if statement.dialect.URI().DBType != schemas.MSSQL || statement.TableAlias == "" {
		return nil
	}

	_, err := fmt.Fprint(updateWriter, " FROM ", statement.quote(statement.TableName()), " ", statement.TableAlias)
	return err
}

func (statement *Statement) writeUpdateLimit(updateWriter *builder.BytesWriter, cond builder.Cond) error {
	if statement.LimitN == nil {
		return nil
	}

	table := statement.RefTable
	tableName := statement.TableName()

	limitValue := *statement.LimitN
	switch statement.dialect.URI().DBType {
	case schemas.MYSQL:
		_, err := fmt.Fprintf(updateWriter, " LIMIT %d", limitValue)
		return err
	case schemas.SQLITE:
		if cond.IsValid() {
			if _, err := fmt.Fprint(updateWriter, " AND "); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprint(updateWriter, " WHERE "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(updateWriter, "rowid IN (SELECT rowid FROM ", statement.quote(tableName)); err != nil {
			return err
		}
		if err := statement.writeWhereCond(updateWriter, cond); err != nil {
			return err
		}
		if err := statement.writeOrderBys(updateWriter); err != nil {
			return err
		}
		_, err := fmt.Fprintf(updateWriter, " LIMIT %d)", limitValue)
		return err
	case schemas.POSTGRES:
		if cond.IsValid() {
			if _, err := fmt.Fprint(updateWriter, " AND "); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprint(updateWriter, " WHERE "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(updateWriter, "CTID IN (SELECT CTID FROM ", statement.quote(tableName)); err != nil {
			return err
		}
		if err := statement.writeWhereCond(updateWriter, cond); err != nil {
			return err
		}
		if err := statement.writeOrderBys(updateWriter); err != nil {
			return err
		}
		_, err := fmt.Fprintf(updateWriter, " LIMIT %d)", limitValue)
		return err
	case schemas.MSSQL:
		if statement.HasOrderBy() && table != nil && len(table.PrimaryKeys) == 1 {
			if _, err := fmt.Fprintf(updateWriter, " WHERE %s IN (SELECT TOP (%d) %s FROM %v",
				table.PrimaryKeys[0], limitValue, table.PrimaryKeys[0],
				statement.quote(tableName)); err != nil {
				return err
			}
			if err := statement.writeWhereCond(updateWriter, cond); err != nil {
				return err
			}
			if err := statement.writeOrderBys(updateWriter); err != nil {
				return err
			}
			_, err := fmt.Fprint(updateWriter, ")")
			return err
		}
		return nil
	default: // TODO: Oracle support needed
		return fmt.Errorf("not implemented")
	}
}

func (statement *Statement) WriteUpdate(updateWriter *builder.BytesWriter, cond builder.Cond, colNames []string, args []interface{}) error {
	if _, err := fmt.Fprintf(updateWriter, "UPDATE"); err != nil {
		return err
	}

	if err := statement.writeUpdateTop(updateWriter); err != nil {
		return err
	}

	if err := statement.writeUpdateTableName(updateWriter); err != nil {
		return err
	}

	// write set
	if _, err := fmt.Fprint(updateWriter, " SET "); err != nil {
		return err
	}
	for i, colName := range colNames {
		if i > 0 {
			if _, err := fmt.Fprint(updateWriter, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(updateWriter, colName); err != nil {
			return err
		}
	}
	updateWriter.Append(args...)

	// write from
	if err := statement.writeUpdateFrom(updateWriter); err != nil {
		return err
	}

	if statement.dialect.URI().DBType == schemas.MSSQL {
		table := statement.RefTable
		if statement.HasOrderBy() && table != nil && len(table.PrimaryKeys) == 1 {
		} else {
			// write where
			if err := statement.writeWhereCond(updateWriter, cond); err != nil {
				return err
			}
		}
	} else {
		// write where
		if err := statement.writeWhereCond(updateWriter, cond); err != nil {
			return err
		}
	}

	if statement.dialect.URI().DBType == schemas.MYSQL {
		if err := statement.writeOrderBys(updateWriter); err != nil {
			return err
		}
	}

	return statement.writeUpdateLimit(updateWriter, cond)
}
