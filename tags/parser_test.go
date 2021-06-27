// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tags

import (
	"reflect"
	"testing"
	"time"

	"xorm.io/xorm/caches"
	"xorm.io/xorm/dialects"
	"xorm.io/xorm/names"

	"github.com/stretchr/testify/assert"
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
		private int // unexported fields will be ignored
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
		private int `xorm:"private"` // unexported fields will be ignored
		Public  int `xorm:"-"`
	}
	table, err = parser.Parse(reflect.ValueOf(new(TaggedStruct)))
	assert.NoError(t, err)
	assert.EqualValues(t, "tagged_struct", table.Name)
	assert.EqualValues(t, 0, len(table.Columns()))
}

func TestParseWithOtherIdentifier(t *testing.T) {
	parser := NewParser(
		"xorm",
		dialects.QueryDialect("mysql"),
		names.SameMapper{},
		names.SnakeMapper{},
		caches.NewManager(),
	)

	type StructWithDBTag struct {
		FieldFoo string `db:"foo"`
	}

	parser.SetIdentifier("db")
	table, err := parser.Parse(reflect.ValueOf(new(StructWithDBTag)))
	assert.NoError(t, err)
	assert.EqualValues(t, "StructWithDBTag", table.Name)
	assert.EqualValues(t, 1, len(table.Columns()))

	for _, col := range table.Columns() {
		assert.EqualValues(t, "foo", col.Name)
	}
}

func TestParseWithIgnore(t *testing.T) {
	parser := NewParser(
		"db",
		dialects.QueryDialect("mysql"),
		names.SameMapper{},
		names.SnakeMapper{},
		caches.NewManager(),
	)

	type StructWithIgnoreTag struct {
		FieldFoo string `db:"-"`
	}

	table, err := parser.Parse(reflect.ValueOf(new(StructWithIgnoreTag)))
	assert.NoError(t, err)
	assert.EqualValues(t, "StructWithIgnoreTag", table.Name)
	assert.EqualValues(t, 0, len(table.Columns()))
}

func TestParseWithAutoincrement(t *testing.T) {
	parser := NewParser(
		"db",
		dialects.QueryDialect("mysql"),
		names.SnakeMapper{},
		names.GonicMapper{},
		caches.NewManager(),
	)

	type StructWithAutoIncrement struct {
		ID int64
	}

	table, err := parser.Parse(reflect.ValueOf(new(StructWithAutoIncrement)))
	assert.NoError(t, err)
	assert.EqualValues(t, "struct_with_auto_increment", table.Name)
	assert.EqualValues(t, 1, len(table.Columns()))
	assert.EqualValues(t, "id", table.Columns()[0].Name)
	assert.True(t, table.Columns()[0].IsAutoIncrement)
	assert.True(t, table.Columns()[0].IsPrimaryKey)
}

func TestParseWithAutoincrement2(t *testing.T) {
	parser := NewParser(
		"db",
		dialects.QueryDialect("mysql"),
		names.SnakeMapper{},
		names.GonicMapper{},
		caches.NewManager(),
	)

	type StructWithAutoIncrement2 struct {
		ID int64 `db:"pk autoincr"`
	}

	table, err := parser.Parse(reflect.ValueOf(new(StructWithAutoIncrement2)))
	assert.NoError(t, err)
	assert.EqualValues(t, "struct_with_auto_increment2", table.Name)
	assert.EqualValues(t, 1, len(table.Columns()))
	assert.EqualValues(t, "id", table.Columns()[0].Name)
	assert.True(t, table.Columns()[0].IsAutoIncrement)
	assert.True(t, table.Columns()[0].IsPrimaryKey)
	assert.False(t, table.Columns()[0].Nullable)
}

func TestParseWithNullable(t *testing.T) {
	parser := NewParser(
		"db",
		dialects.QueryDialect("mysql"),
		names.SnakeMapper{},
		names.GonicMapper{},
		caches.NewManager(),
	)

	type StructWithNullable struct {
		Name     string `db:"notnull"`
		FullName string `db:"null comment('column comment,字段注释')"`
	}

	table, err := parser.Parse(reflect.ValueOf(new(StructWithNullable)))
	assert.NoError(t, err)
	assert.EqualValues(t, "struct_with_nullable", table.Name)
	assert.EqualValues(t, 2, len(table.Columns()))
	assert.EqualValues(t, "name", table.Columns()[0].Name)
	assert.EqualValues(t, "full_name", table.Columns()[1].Name)
	assert.False(t, table.Columns()[0].Nullable)
	assert.True(t, table.Columns()[1].Nullable)
	assert.EqualValues(t, "column comment,字段注释", table.Columns()[1].Comment)
}

func TestParseWithTimes(t *testing.T) {
	parser := NewParser(
		"db",
		dialects.QueryDialect("mysql"),
		names.SnakeMapper{},
		names.GonicMapper{},
		caches.NewManager(),
	)

	type StructWithTimes struct {
		Name      string    `db:"notnull"`
		CreatedAt time.Time `db:"created"`
		UpdatedAt time.Time `db:"updated"`
		DeletedAt time.Time `db:"deleted"`
	}

	table, err := parser.Parse(reflect.ValueOf(new(StructWithTimes)))
	assert.NoError(t, err)
	assert.EqualValues(t, "struct_with_times", table.Name)
	assert.EqualValues(t, 4, len(table.Columns()))
	assert.EqualValues(t, "name", table.Columns()[0].Name)
	assert.EqualValues(t, "created_at", table.Columns()[1].Name)
	assert.EqualValues(t, "updated_at", table.Columns()[2].Name)
	assert.EqualValues(t, "deleted_at", table.Columns()[3].Name)
	assert.False(t, table.Columns()[0].Nullable)
	assert.True(t, table.Columns()[1].Nullable)
	assert.True(t, table.Columns()[1].IsCreated)
	assert.True(t, table.Columns()[2].Nullable)
	assert.True(t, table.Columns()[2].IsUpdated)
	assert.True(t, table.Columns()[3].Nullable)
	assert.True(t, table.Columns()[3].IsDeleted)
}
