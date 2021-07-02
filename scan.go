// Copyright 2021 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"database/sql"
	"fmt"

	"xorm.io/xorm/core"
	"xorm.io/xorm/dialects"
)

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

func (engine *Engine) row2mapBytes(rows *core.Rows, types []*sql.ColumnType, fields []string) (map[string][]byte, error) {
	var scanResults = make([]interface{}, len(fields))
	for i := 0; i < len(fields); i++ {
		var s sql.NullString
		scanResults[i] = &s
	}

	if err := rows.Scan(scanResults...); err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(fields))
	for ii, key := range fields {
		s := scanResults[ii].(*sql.NullString)
		result[key] = []byte(s.String)
	}
	return result, nil
}

func (engine *Engine) row2sliceStr(rows *core.Rows, types []*sql.ColumnType, fields []string) ([]string, error) {
	results := make([]string, 0, len(fields))
	var scanResults = make([]interface{}, len(fields))
	for i := 0; i < len(fields); i++ {
		var s sql.NullString
		scanResults[i] = &s
	}

	if err := rows.Scan(scanResults...); err != nil {
		return nil, err
	}

	for i := 0; i < len(fields); i++ {
		results = append(results, scanResults[i].(*sql.NullString).String)
	}
	return results, nil
}

func (engine *Engine) row2mapInterface(rows *core.Rows, types []*sql.ColumnType, fields []string) (map[string]interface{}, error) {
	var resultsMap = make(map[string]interface{}, len(fields))
	var scanResultContainers = make([]interface{}, len(fields))
	for i := 0; i < len(fields); i++ {
		scanResult, err := engine.driver.GenScanResult(types[i].DatabaseTypeName())
		if err != nil {
			return nil, err
		}
		scanResultContainers[i] = scanResult
	}
	if err := engine.driver.Scan(&dialects.ScanContext{
		DBLocation:   engine.DatabaseTZ,
		UserLocation: engine.TZLocation,
	}, rows, types, scanResultContainers...); err != nil {
		return nil, err
	}

	for ii, key := range fields {
		switch t := scanResultContainers[ii].(type) {
		case *sql.NullInt32:
			resultsMap[key] = t.Int32
		case *sql.NullInt64:
			resultsMap[key] = t.Int64
		case *sql.NullFloat64:
			resultsMap[key] = t.Float64
		case *sql.NullString:
			resultsMap[key] = t.String
		case *sql.NullTime:
			if t.Valid {
				resultsMap[key] = t.Time.In(engine.TZLocation).Format("2006-01-02 15:04:05")
			} else {
				resultsMap[key] = nil
			}
		case *sql.RawBytes:
			if t == nil {
				resultsMap[key] = nil
			} else {
				resultsMap[key] = []byte(*t)
			}
		default:
			return nil, fmt.Errorf("unknow type: %v", t)
		}
	}
	return resultsMap, nil
}
