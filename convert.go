// Copyright 2017 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"time"

	"xorm.io/xorm/internal/convert"
)

var errNilPtr = errors.New("destination pointer is nil") // embedded in descriptive error

func strconvErr(err error) error {
	if ne, ok := err.(*strconv.NumError); ok {
		return ne.Err
	}
	return err
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// convertAssign copies to dest the value in src, converting it if possible.
// An error is returned if the copy would result in loss of information.
// dest should be a pointer type.
func convertAssign(dest, src interface{}, originalLocation *time.Location, convertedLocation *time.Location) error {
	// Common cases, without reflect.
	switch s := src.(type) {
	case *interface{}:
		return convertAssign(dest, *s, originalLocation, convertedLocation)
	case string:
		switch d := dest.(type) {
		case *string:
			if d == nil {
				return errNilPtr
			}
			*d = s
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = []byte(s)
			return nil
		}
	case []byte:
		switch d := dest.(type) {
		case *string:
			if d == nil {
				return errNilPtr
			}
			*d = string(s)
			return nil
		case *interface{}:
			if d == nil {
				return errNilPtr
			}
			*d = cloneBytes(s)
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = cloneBytes(s)
			return nil
		}
	case time.Time:
		switch d := dest.(type) {
		case *string:
			*d = s.Format(time.RFC3339Nano)
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = []byte(s.Format(time.RFC3339Nano))
			return nil
		}
	case nil:
		switch d := dest.(type) {
		case *interface{}:
			if d == nil {
				return errNilPtr
			}
			*d = nil
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}
			*d = nil
			return nil
		}
	case *sql.NullString:
		switch d := dest.(type) {
		case *int:
			if s.Valid {
				*d, _ = strconv.Atoi(s.String)
			}
			return nil
		case *int64:
			if s.Valid {
				*d, _ = strconv.ParseInt(s.String, 10, 64)
			}
			return nil
		case *string:
			if s.Valid {
				*d = s.String
			}
			return nil
		case *time.Time:
			if s.Valid {
				var err error
				dt, err := convert.String2Time(s.String, originalLocation, convertedLocation)
				if err != nil {
					return err
				}
				*d = *dt
			}
			return nil
		case *sql.NullTime:
			if s.Valid {
				var err error
				dt, err := convert.String2Time(s.String, originalLocation, convertedLocation)
				if err != nil {
					return err
				}
				d.Valid = true
				d.Time = *dt
			}
			return nil
		case *big.Float:
			if s.Valid {
				if d == nil {
					d = big.NewFloat(0)
				}
				d.SetString(s.String)
			}
			return nil
		}
	case *sql.NullInt32:
		switch d := dest.(type) {
		case *int:
			if s.Valid {
				*d = int(s.Int32)
			}
			return nil
		case *int8:
			if s.Valid {
				*d = int8(s.Int32)
			}
			return nil
		case *int16:
			if s.Valid {
				*d = int16(s.Int32)
			}
			return nil
		case *int32:
			if s.Valid {
				*d = s.Int32
			}
			return nil
		case *int64:
			if s.Valid {
				*d = int64(s.Int32)
			}
			return nil
		}
	case *sql.NullInt64:
		switch d := dest.(type) {
		case *int:
			if s.Valid {
				*d = int(s.Int64)
			}
			return nil
		case *int8:
			if s.Valid {
				*d = int8(s.Int64)
			}
			return nil
		case *int16:
			if s.Valid {
				*d = int16(s.Int64)
			}
			return nil
		case *int32:
			if s.Valid {
				*d = int32(s.Int64)
			}
			return nil
		case *int64:
			if s.Valid {
				*d = s.Int64
			}
			return nil
		}
	case *sql.NullFloat64:
		switch d := dest.(type) {
		case *int:
			if s.Valid {
				*d = int(s.Float64)
			}
			return nil
		case *float64:
			if s.Valid {
				*d = s.Float64
			}
			return nil
		}
	case *sql.NullBool:
		switch d := dest.(type) {
		case *bool:
			if s.Valid {
				*d = s.Bool
			}
			return nil
		}
	case *sql.NullTime:
		switch d := dest.(type) {
		case *time.Time:
			if s.Valid {
				*d = s.Time
			}
			return nil
		case *string:
			if s.Valid {
				*d = s.Time.In(convertedLocation).Format("2006-01-02 15:04:05")
			}
			return nil
		}
	case *convert.NullUint32:
		switch d := dest.(type) {
		case *uint8:
			if s.Valid {
				*d = uint8(s.Uint32)
			}
			return nil
		case *uint16:
			if s.Valid {
				*d = uint16(s.Uint32)
			}
			return nil
		case *uint:
			if s.Valid {
				*d = uint(s.Uint32)
			}
			return nil
		}
	case *convert.NullUint64:
		switch d := dest.(type) {
		case *uint64:
			if s.Valid {
				*d = s.Uint64
			}
			return nil
		}
	case *sql.RawBytes:
		switch d := dest.(type) {
		case convert.Conversion:
			return d.FromDB(*s)
		}
	}

	var sv reflect.Value

	switch d := dest.(type) {
	case *string:
		sv = reflect.ValueOf(src)
		switch sv.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			*d = convert.AsString(src)
			return nil
		}
	case *[]byte:
		if b, ok := convert.AsBytes(src); ok {
			*d = b
			return nil
		}
	case *bool:
		bv, err := driver.Bool.ConvertValue(src)
		if err == nil {
			*d = bv.(bool)
		}
		return err
	case *interface{}:
		*d = src
		return nil
	}

	return convertAssignV(reflect.ValueOf(dest), src)
}

func convertAssignV(dv reflect.Value, src interface{}) error {
	if src == nil {
		return nil
	}

	if dv.Type().Implements(scannerType) {
		return dv.Interface().(sql.Scanner).Scan(src)
	}

	switch dv.Kind() {
	case reflect.Ptr:
		if dv.IsNil() {
			dv.Set(reflect.New(dv.Type().Elem()))
		}
		return convertAssignV(dv.Elem(), src)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i64, err := convert.AsInt64(src)
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T to a %s: %v", src, dv.Kind(), err)
		}
		dv.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u64, err := convert.AsUint64(src)
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T to a %s: %v", src, dv.Kind(), err)
		}
		dv.SetUint(u64)
		return nil
	case reflect.Float32, reflect.Float64:
		f64, err := convert.AsFloat64(src)
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T to a %s: %v", src, dv.Kind(), err)
		}
		dv.SetFloat(f64)
		return nil
	case reflect.String:
		dv.SetString(convert.AsString(src))
		return nil
	case reflect.Bool:
		b, err := convert.AsBool(src)
		if err != nil {
			return err
		}
		dv.SetBool(b)
		return nil
	case reflect.Slice, reflect.Map, reflect.Struct, reflect.Array:
		data, ok := convert.AsBytes(src)
		if !ok {
			return fmt.Errorf("onvertAssignV: src cannot be as bytes %#v", src)
		}
		if data == nil {
			return nil
		}
		if dv.Kind() != reflect.Ptr {
			dv = dv.Addr()
		}
		return json.Unmarshal(data, dv.Interface())
	default:
		return fmt.Errorf("convertAssignV: unsupported Scan, storing driver.Value type %T into type %T", src, dv.Interface())
	}
}

func asKind(vv reflect.Value, tp reflect.Type) (interface{}, error) {
	switch tp.Kind() {
	case reflect.Ptr:
		return asKind(vv.Elem(), tp.Elem())
	case reflect.Int64:
		return vv.Int(), nil
	case reflect.Int:
		return int(vv.Int()), nil
	case reflect.Int32:
		return int32(vv.Int()), nil
	case reflect.Int16:
		return int16(vv.Int()), nil
	case reflect.Int8:
		return int8(vv.Int()), nil
	case reflect.Uint64:
		return vv.Uint(), nil
	case reflect.Uint:
		return uint(vv.Uint()), nil
	case reflect.Uint32:
		return uint32(vv.Uint()), nil
	case reflect.Uint16:
		return uint16(vv.Uint()), nil
	case reflect.Uint8:
		return uint8(vv.Uint()), nil
	case reflect.String:
		return vv.String(), nil
	case reflect.Slice:
		if tp.Elem().Kind() == reflect.Uint8 {
			v, err := strconv.ParseInt(string(vv.Interface().([]byte)), 10, 64)
			if err != nil {
				return nil, err
			}
			return v, nil
		}
	}
	return nil, fmt.Errorf("unsupported primary key type: %v, %v", tp, vv)
}
