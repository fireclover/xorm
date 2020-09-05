// Copyright 2017 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integrations

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStoreEngine(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	assert.NoError(t, testEngine.DropTables("user_store_engine"))

	type UserinfoStoreEngine struct {
		Id   int64
		Name string
	}

	assert.NoError(t, testEngine.StoreEngine("InnoDB").Table("user_store_engine").CreateTable(&UserinfoStoreEngine{}))
}

func TestCreateTable(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	assert.NoError(t, testEngine.DropTables("user_user"))

	type UserinfoCreateTable struct {
		Id   int64
		Name string
	}

	assert.NoError(t, testEngine.Table("user_user").CreateTable(&UserinfoCreateTable{}))
}

func TestCreateMultiTables(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	session := testEngine.NewSession()
	defer session.Close()

	type UserinfoMultiTable struct {
		Id   int64
		Name string
	}

	user := &UserinfoMultiTable{}
	assert.NoError(t, session.Begin())

	for i := 0; i < 10; i++ {
		tableName := fmt.Sprintf("user_%v", i)

		assert.NoError(t, session.DropTable(tableName))

		assert.NoError(t, session.Table(tableName).CreateTable(user))
	}

	assert.NoError(t, session.Commit())
}

type SyncTable1 struct {
	Id   int64
	Name string
	Dev  int `xorm:"index"`
}

type SyncTable2 struct {
	Id     int64
	Name   string `xorm:"unique"`
	Number string `xorm:"index"`
	Dev    int
	Age    int
}

func (SyncTable2) TableName() string {
	return "sync_table1"
}

type SyncTable3 struct {
	Id     int64
	Name   string `xorm:"unique"`
	Number string `xorm:"index"`
	Dev    int
	Age    int
}

func (s *SyncTable3) TableName() string {
	return "sync_table1"
}

func TestSyncTable(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	assert.NoError(t, testEngine.Sync2(new(SyncTable1)))

	tables, err := testEngine.DBMetas()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, len(tables))
	assert.EqualValues(t, "sync_table1", tables[0].Name)
	tableInfo, err := testEngine.TableInfo(new(SyncTable1))
	assert.NoError(t, err)
	assert.EqualValues(t, testEngine.Dialect().SQLType(tables[0].GetColumn("name")), testEngine.Dialect().SQLType(tableInfo.GetColumn("name")))

	assert.NoError(t, testEngine.Sync2(new(SyncTable2)))

	tables, err = testEngine.DBMetas()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, len(tables))
	assert.EqualValues(t, "sync_table1", tables[0].Name)
	tableInfo, err = testEngine.TableInfo(new(SyncTable2))
	assert.NoError(t, err)
	assert.EqualValues(t, testEngine.Dialect().SQLType(tables[0].GetColumn("name")), testEngine.Dialect().SQLType(tableInfo.GetColumn("name")))

	assert.NoError(t, testEngine.Sync2(new(SyncTable3)))

	tables, err = testEngine.DBMetas()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, len(tables))
	assert.EqualValues(t, "sync_table1", tables[0].Name)
	tableInfo, err = testEngine.TableInfo(new(SyncTable3))
	assert.NoError(t, err)
	assert.EqualValues(t, testEngine.Dialect().SQLType(tables[0].GetColumn("name")), testEngine.Dialect().SQLType(tableInfo.GetColumn("name")))
}

func TestSyncTable2(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	assert.NoError(t, testEngine.Table("sync_tablex").Sync2(new(SyncTable1)))

	tables, err := testEngine.DBMetas()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, len(tables))
	assert.EqualValues(t, "sync_tablex", tables[0].Name)
	assert.EqualValues(t, 3, len(tables[0].Columns()))

	type SyncTable4 struct {
		SyncTable1 `xorm:"extends"`
		NewCol     string
	}

	assert.NoError(t, testEngine.Table("sync_tablex").Sync2(new(SyncTable4)))
	tables, err = testEngine.DBMetas()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, len(tables))
	assert.EqualValues(t, "sync_tablex", tables[0].Name)
	assert.EqualValues(t, 4, len(tables[0].Columns()))
	assert.EqualValues(t, colMapper.Obj2Table("NewCol"), tables[0].Columns()[3].Name)
}

type SyncTable4 struct {
	Id   int64
	Name string
	Text string `xorm:"TEXT"`
	Char byte   `xorm:"CHAR"`
}

func (s *SyncTable4) TableName() string {
	return "sync_table1"
}

func TestSyncTable3(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	assert.NoError(t, testEngine.Sync2(new(SyncTable4)))

	tables, err := testEngine.DBMetas()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, len(tables))
	assert.EqualValues(t, "sync_table1", tables[0].Name)
	tableInfo, err := testEngine.TableInfo(new(SyncTable4))
	assert.NoError(t, err)
	assert.EqualValues(t, testEngine.Dialect().SQLType(tables[0].GetColumn("name")), testEngine.Dialect().SQLType(tableInfo.GetColumn("name")))

	if *doNVarcharTest {
		var oldDefaultVarchar string
		var oldDefaultChar string
		oldDefaultVarchar, *defaultVarchar = *defaultVarchar, "nvarchar"
		oldDefaultChar, *defaultChar = *defaultChar, "nchar"
		testEngine.Dialect().SetParams(map[string]string{
			"DEFAULT_VARCHAR": *defaultVarchar,
			"DEFAULT_CHAR":    *defaultChar,
		})
		defer func() {
			*defaultVarchar = oldDefaultVarchar
			*defaultChar = oldDefaultChar
			testEngine.Dialect().SetParams(map[string]string{
				"DEFAULT_VARCHAR": *defaultVarchar,
				"DEFAULT_CHAR":    *defaultChar,
			})
		}()
		assert.NoError(t, PrepareEngine())

		assert.NoError(t, testEngine.Sync2(new(SyncTable4)))

		tables, err := testEngine.DBMetas()
		assert.NoError(t, err)
		assert.EqualValues(t, 1, len(tables))
		assert.EqualValues(t, "sync_table1", tables[0].Name)
		tableInfo, err := testEngine.TableInfo(new(SyncTable4))
		assert.NoError(t, err)
		assert.EqualValues(t, testEngine.Dialect().SQLType(tables[0].GetColumn("name")), testEngine.Dialect().SQLType(tableInfo.GetColumn("name")))
	}
}

func TestIsTableExist(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	exist, err := testEngine.IsTableExist(new(CustomTableName))
	assert.NoError(t, err)
	assert.False(t, exist)

	assert.NoError(t, testEngine.CreateTables(new(CustomTableName)))

	exist, err = testEngine.IsTableExist(new(CustomTableName))
	assert.NoError(t, err)
	assert.True(t, exist)
}

func TestIsTableEmpty(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	type NumericEmpty struct {
		Numeric float64 `xorm:"numeric(26,2)"`
	}

	type PictureEmpty struct {
		Id          int64
		Url         string `xorm:"unique"` //image's url
		Title       string
		Description string
		Created     time.Time `xorm:"created"`
		ILike       int
		PageView    int
		From_url    string
		Pre_url     string `xorm:"unique"` //pre view image's url
		Uid         int64
	}

	assert.NoError(t, testEngine.DropTables(&PictureEmpty{}, &NumericEmpty{}))

	assert.NoError(t, testEngine.Sync2(new(PictureEmpty), new(NumericEmpty)))

	isEmpty, err := testEngine.IsTableEmpty(&PictureEmpty{})
	assert.NoError(t, err)
	assert.True(t, isEmpty)

	tbName := testEngine.GetTableMapper().Obj2Table("PictureEmpty")
	isEmpty, err = testEngine.IsTableEmpty(tbName)
	assert.NoError(t, err)
	assert.True(t, isEmpty)
}

type CustomTableName struct {
	Id   int64
	Name string
}

func (c *CustomTableName) TableName() string {
	return "customtablename"
}

func TestCustomTableName(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	c := new(CustomTableName)
	assert.NoError(t, testEngine.DropTables(c))

	assert.NoError(t, testEngine.CreateTables(c))
}

type IndexOrUnique struct {
	Id        int64
	Index     int `xorm:"index"`
	Unique    int `xorm:"unique"`
	Group1    int `xorm:"index(ttt)"`
	Group2    int `xorm:"index(ttt)"`
	UniGroup1 int `xorm:"unique(lll)"`
	UniGroup2 int `xorm:"unique(lll)"`
}

func TestIndexAndUnique(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	assert.NoError(t, testEngine.CreateTables(&IndexOrUnique{}))

	assert.NoError(t, testEngine.DropTables(&IndexOrUnique{}))

	assert.NoError(t, testEngine.CreateTables(&IndexOrUnique{}))

	assert.NoError(t, testEngine.CreateIndexes(&IndexOrUnique{}))

	assert.NoError(t, testEngine.CreateUniques(&IndexOrUnique{}))

	assert.NoError(t, testEngine.DropIndexes(&IndexOrUnique{}))
}

func TestMetaInfo(t *testing.T) {
	assert.NoError(t, PrepareEngine())
	assert.NoError(t, testEngine.Sync2(new(CustomTableName), new(IndexOrUnique)))

	tables, err := testEngine.DBMetas()
	assert.NoError(t, err)
	assert.EqualValues(t, 2, len(tables))
	tableNames := []string{tables[0].Name, tables[1].Name}
	assert.Contains(t, tableNames, "customtablename")
	assert.Contains(t, tableNames, "index_or_unique")
}

func TestCharst(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	err := testEngine.DropTables("user_charset")
	assert.NoError(t, err)

	err = testEngine.Charset("utf8").Table("user_charset").CreateTable(&Userinfo{})
	assert.NoError(t, err)
}

func TestSync2_1(t *testing.T) {
	type WxTest struct {
		Id                 int   `xorm:"not null pk autoincr INT(64)"`
		Passport_user_type int16 `xorm:"null int"`
		Id_delete          int8  `xorm:"null int default 1"`
	}

	assert.NoError(t, PrepareEngine())

	assert.NoError(t, testEngine.DropTables("wx_test"))
	assert.NoError(t, testEngine.Sync2(new(WxTest)))
	assert.NoError(t, testEngine.Sync2(new(WxTest)))
}

func TestUnique_1(t *testing.T) {
	type UserUnique struct {
		Id        int64
		UserName  string    `xorm:"unique varchar(25) not null"`
		Password  string    `xorm:"varchar(255) not null"`
		Admin     bool      `xorm:"not null"`
		CreatedAt time.Time `xorm:"created"`
		UpdatedAt time.Time `xorm:"updated"`
	}

	assert.NoError(t, PrepareEngine())

	assert.NoError(t, testEngine.DropTables("user_unique"))
	assert.NoError(t, testEngine.Sync2(new(UserUnique)))

	assert.NoError(t, testEngine.DropTables("user_unique"))
	assert.NoError(t, testEngine.CreateTables(new(UserUnique)))
	assert.NoError(t, testEngine.CreateUniques(new(UserUnique)))
}

func TestSync2_2(t *testing.T) {
	type TestSync2Index struct {
		Id     int64
		UserId int64 `xorm:"index"`
	}

	assert.NoError(t, PrepareEngine())

	var tableNames = make(map[string]bool)
	for i := 0; i < 10; i++ {
		tableName := fmt.Sprintf("test_sync2_index_%d", i)
		tableNames[tableName] = true
		assert.NoError(t, testEngine.Table(tableName).Sync2(new(TestSync2Index)))

		exist, err := testEngine.IsTableExist(tableName)
		assert.NoError(t, err)
		assert.True(t, exist)
	}

	tables, err := testEngine.DBMetas()
	assert.NoError(t, err)

	for _, table := range tables {
		assert.True(t, tableNames[table.Name])
	}
}

func TestSync2_Default(t *testing.T) {
	type TestSync2Default struct {
		Id       int64
		UserId   int64  `xorm:"default(1)"`
		IsMember bool   `xorm:"default(true)"`
		Name     string `xorm:"default('my_name')"`
	}

	assert.NoError(t, PrepareEngine())
	assertSync(t, new(TestSync2Default))
	assert.NoError(t, testEngine.Sync2(new(TestSync2Default)))
}
