// Copyright 2017 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExistStruct(t *testing.T) {
	assert.NoError(t, prepareEngine())

	type RecordExist struct {
		Id   int64
		Name string
	}

	assertSync(t, new(RecordExist))

	has, err := testEngine.Exist(new(RecordExist))
	assert.NoError(t, err)
	assert.False(t, has)

	cnt, err := testEngine.Insert(&RecordExist{
		Name: "test1",
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 1, cnt)

	has, err = testEngine.Exist(new(RecordExist))
	assert.NoError(t, err)
	assert.True(t, has)

	has, err = testEngine.Exist(&RecordExist{
		Name: "test1",
	})
	assert.NoError(t, err)
	assert.True(t, has)

	has, err = testEngine.Exist(&RecordExist{
		Name: "test2",
	})
	assert.NoError(t, err)
	assert.False(t, has)

	has, err = testEngine.Where("name = ?", "test1").Exist(&RecordExist{})
	assert.NoError(t, err)
	assert.True(t, has)

	has, err = testEngine.Where("name = ?", "test2").Exist(&RecordExist{})
	assert.NoError(t, err)
	assert.False(t, has)

	has, err = testEngine.SQL("select * from "+testEngine.TableName("record_exist", true)+" where name = ?", "test1").Exist()
	assert.NoError(t, err)
	assert.True(t, has)

	has, err = testEngine.SQL("select * from "+testEngine.TableName("record_exist", true)+" where name = ?", "test2").Exist()
	assert.NoError(t, err)
	assert.False(t, has)

	has, err = testEngine.Table("record_exist").Exist()
	assert.NoError(t, err)
	assert.True(t, has)

	has, err = testEngine.Table("record_exist").Where("name = ?", "test1").Exist()
	assert.NoError(t, err)
	assert.True(t, has)

	has, err = testEngine.Table("record_exist").Where("name = ?", "test2").Exist()
	assert.NoError(t, err)
	assert.False(t, has)
}

func TestExistStructForJoin(t *testing.T) {
	assert.NoError(t, prepareEngine())

	type Salary struct {
		Id  int64
		Lid int64
	}

	type CheckList struct {
		Id  int64
		Eid int64
	}

	type Empsetting struct {
		Id   int64
		Name string
	}

	assert.NoError(t, testEngine.Sync2(new(Salary), new(CheckList), new(Empsetting)))

	var emp Empsetting
	cnt, err := testEngine.Insert(&emp)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, cnt)

	var checklist = CheckList{
		Eid: emp.Id,
	}
	cnt, err = testEngine.Insert(&checklist)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, cnt)

	var salary = Salary{
		Lid: checklist.Id,
	}
	cnt, err = testEngine.Insert(&salary)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, cnt)

	session := testEngine.NewSession()
	defer session.Close()

	session.Table("salary").
		Join("INNER", "check_list", "check_list.id = salary.lid").
		Join("LEFT", "empsetting", "empsetting.id = check_list.eid").
		Where("salary.lid = ?", 1)
	has, err := session.Exist()
	assert.NoError(t, err)
	assert.True(t, has)

	session.Table("salary").
		Join("INNER", "check_list", "check_list.id = salary.lid").
		Join("LEFT", "empsetting", "empsetting.id = check_list.eid").
		Where("salary.lid = ?", 2)
	has, err = session.Exist()
	assert.NoError(t, err)
	assert.False(t, has)

	session.Table("salary").
		Select("check_list.id").
		Join("INNER", "check_list", "check_list.id = salary.lid").
		Join("LEFT", "empsetting", "empsetting.id = check_list.eid").
		Where("check_list.id = ?", 1)
	has, err = session.Exist()
	assert.NoError(t, err)
	assert.True(t, has)

	session.Table("salary").
		Select("empsetting.id").
		Join("INNER", "check_list", "check_list.id = salary.lid").
		Join("LEFT", "empsetting", "empsetting.id = check_list.eid").
		Where("empsetting.id = ?", 2)
	has, err = session.Exist()
	assert.NoError(t, err)
	assert.False(t, has)

	session.Table("salary").
		Select("empsetting.id").
		Join("INNER", "check_list", "check_list.id = salary.lid").
		Join("LEFT", "empsetting", "empsetting.id = check_list.eid")
	has, err = session.Exist()
	assert.NoError(t, err)
	assert.True(t, has)

	err = session.DropTable("check_list")
	assert.NoError(t, err)

	session.Table("salary").
		Select("empsetting.id").
		Join("INNER", "check_list", "check_list.id = salary.lid").
		Join("LEFT", "empsetting", "empsetting.id = check_list.eid")
	has, err = session.Exist()
	assert.Error(t, err)
	assert.False(t, has)

	session.Table("salary").
		Select("empsetting.id").
		Join("LEFT", "empsetting", "empsetting.id = salary.lid")
	has, err = session.Exist()
	assert.NoError(t, err)
	assert.True(t, has)
}
