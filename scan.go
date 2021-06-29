// Copyright 2021 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"database/sql"
	"time"

	"xorm.io/xorm/convert"
	"xorm.io/xorm/core"
)

func (session *Session) scan(rows *core.Rows, types []*sql.ColumnType, v ...interface{}) error {
	var v2 = make([]interface{}, 0, len(v))
	var turnBackIdxes = make([]int, 0, 5)
	for i, vv := range v {
		switch vv.(type) {
		case *time.Time:
			v2 = append(v2, &sql.NullString{})
			turnBackIdxes = append(turnBackIdxes, i)
		case *sql.NullTime:
			v2 = append(v2, &sql.NullString{})
			turnBackIdxes = append(turnBackIdxes, i)
		default:
			v2 = append(v2, v[i])
		}
	}
	if err := rows.Scan(v2...); err != nil {
		return err
	}
	for _, i := range turnBackIdxes {
		switch t := v[i].(type) {
		case *time.Time:
			var s = *(v2[i].(*sql.NullString))
			if !s.Valid {
				break
			}
			dt, err := convert.String2Time(s.String, session.engine.DatabaseTZ, session.engine.TZLocation)
			if err != nil {
				return err
			}
			*t = *dt
		case *sql.NullTime:
			var s = *(v2[i].(*sql.NullString))
			if !s.Valid {
				break
			}
			dt, err := convert.String2Time(s.String, session.engine.DatabaseTZ, session.engine.TZLocation)
			if err != nil {
				return err
			}
			t.Time = *dt
			t.Valid = true
		}
	}
	return nil
}

func (engine *Engine) row2mapStr(rows *core.Rows, types []*sql.ColumnType, fields []string) (map[string]string, error) {
	var scanResults = make([]interface{}, len(fields))
	for i := 0; i < len(fields); i++ {
		var s sql.NullString
		scanResults[i] = &s
	}

	if err := rows.Scan(scanResults...); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(fields))
	for ii, key := range fields {
		s := scanResults[ii].(*sql.NullString)
		result[key] = s.String
	}
	return result, nil
}
