// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import (
	"sort"
	"strings"
)

func MapToSlices(m map[string]interface{}, exclude []string, trimmer func(string) string) ([]string, []interface{}) {
	columns := make([]string, 0, len(m))

outer:
	for colName := range m {
		trimmed := trimmer(colName)
		for _, excluded := range exclude {
			if strings.EqualFold(excluded, trimmed) {
				continue outer
			}
		}
		columns = append(columns, colName)
	}
	sort.Strings(columns)

	args := make([]interface{}, 0, len(columns))
	for _, colName := range columns {
		args = append(args, m[colName])
	}

	return columns, args
}

func MapStringToSlices(m map[string]string, exclude []string, trimmer func(string) string) ([]string, []interface{}) {
	columns := make([]string, 0, len(m))

outer:
	for colName := range m {
		trimmed := trimmer(colName)
		for _, excluded := range exclude {
			if strings.EqualFold(excluded, trimmed) {
				continue outer
			}
		}
		columns = append(columns, colName)
	}
	sort.Strings(columns)

	args := make([]interface{}, 0, len(columns))
	for _, colName := range columns {
		args = append(args, m[colName])
	}

	return columns, args
}

func MultipleMapToSlices(maps []map[string]interface{}, exclude []string, trimmer func(string) string) ([]string, [][]interface{}) {
	columns := make([]string, 0, len(maps[0]))

outer:
	for colName := range maps[0] {
		trimmed := trimmer(colName)
		for _, excluded := range exclude {
			if strings.EqualFold(excluded, trimmed) {
				continue outer
			}
		}
		columns = append(columns, colName)
	}
	sort.Strings(columns)

	argss := make([][]interface{}, 0, len(maps))
	for _, m := range maps {
		args := make([]interface{}, 0, len(m))
		for _, colName := range columns {
			args = append(args, m[colName])
		}
		argss = append(argss, args)
	}

	return columns, argss
}

func MultipleMapStringToSlices(maps []map[string]string, exclude []string, trimmer func(string) string) ([]string, [][]interface{}) {
	columns := make([]string, 0, len(maps[0]))

outer:
	for colName := range maps[0] {
		trimmed := trimmer(colName)
		for _, excluded := range exclude {
			if strings.EqualFold(excluded, trimmed) {
				continue outer
			}
		}
		columns = append(columns, colName)
	}
	sort.Strings(columns)

	argss := make([][]interface{}, 0, len(maps))
	for _, m := range maps {
		args := make([]interface{}, 0, len(m))
		for _, colName := range columns {
			args = append(args, m[colName])
		}
		argss = append(argss, args)
	}

	return columns, argss
}
