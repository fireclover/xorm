// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tags

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm/caches"
	"xorm.io/xorm/dialects"
	"xorm.io/xorm/names"
	"xorm.io/xorm/schemas"
)

type ParseTableName1 struct{}

type ParseTableName2 struct{}

func (p ParseTableName2) TableName() string {
	return "p_parseTableName"
}

func TestParseTableName(t *testing.T) {
	parser := NewParser(
		"xorm",
		dialects.QueryDialect("mysql"),
		names.SnakeMapper{},
		names.SnakeMapper{},
		caches.NewManager(),
	)
	table, err := parser.Parse(reflect.ValueOf(new(ParseTableName1)))
	assert.NoError(t, err)
	assert.EqualValues(t, "parse_table_name1", table.Name)

	table, err = parser.Parse(reflect.ValueOf(new(ParseTableName2)))
	assert.NoError(t, err)
	assert.EqualValues(t, "p_parseTableName", table.Name)

	table, err = parser.Parse(reflect.ValueOf(ParseTableName2{}))
	assert.NoError(t, err)
	assert.EqualValues(t, "p_parseTableName", table.Name)
}

func TestUnexportField(t *testing.T) {
	parser := NewParser(
		"xorm",
		dialects.QueryDialect("mysql"),
		names.SnakeMapper{},
		names.SnakeMapper{},
		caches.NewManager(),
	)

	type VanilaStruct struct {
		private int
		Public  int
	}
	table, err := parser.Parse(reflect.ValueOf(new(VanilaStruct)))
	assert.NoError(t, err)
	assert.EqualValues(t, "vanila_struct", table.Name)
	assert.EqualValues(t, 1, len(table.Columns()))

	for _, col := range table.Columns() {
		assert.EqualValues(t, "public", col.Name)
		assert.NotEqual(t, "private", col.Name)
	}

	type TaggedStruct struct {
		private int `xorm:"private"`
		Public  int `xorm:"-"`
	}
	table, err = parser.Parse(reflect.ValueOf(new(TaggedStruct)))
	assert.NoError(t, err)
	assert.EqualValues(t, "tagged_struct", table.Name)
	assert.EqualValues(t, 1, len(table.Columns()))

	for _, col := range table.Columns() {
		assert.EqualValues(t, "private", col.Name)
		assert.NotEqual(t, "public", col.Name)
	}
}

type ParseTableName3 struct {
	Name string `orm:"varchar(50)"`
}

func TestParseTableNameWithOtherIdentifier(t *testing.T) {
	parser := NewParser(
		"xorm",
		dialects.QueryDialect("mysql"),
		names.SnakeMapper{},
		names.SnakeMapper{},
		caches.NewManager(),
	)
	parser.SetIdentifier("orm")
	table, err := parser.Parse(reflect.ValueOf(new(ParseTableName3)))
	assert.NoError(t, err)
	cols := table.Columns()
	assert.EqualValues(t, 1, len(cols))
	assert.EqualValues(t, "name", cols[0].Name)
	assert.EqualValues(t, schemas.SQLType{
		Name: "VARCHAR",
	}, cols[0].SQLType)
	assert.EqualValues(t, 50, cols[0].Length)
	assert.EqualValues(t, true, cols[0].Nullable)
}
