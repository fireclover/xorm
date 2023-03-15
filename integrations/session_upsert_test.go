// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integrations

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsertOnConflictDoNothing(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	t.Run("NoUnique", func(t *testing.T) {
		// InsertOnConflictDoNothing does not work if there is no unique constraint
		type NoUniques struct {
			ID   int64 `xorm:"pk autoincr"`
			Data string
		}
		assert.NoError(t, testEngine.Sync(new(NoUniques)))

		toInsert := &NoUniques{Data: "shouldErr"}
		n, err := testEngine.InsertOnConflictDoNothing(toInsert)
		assert.Error(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)

		toInsert = &NoUniques{Data: ""}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.Error(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)
	})

	t.Run("OneUnique", func(t *testing.T) {
		type OneUnique struct {
			ID   int64  `xorm:"pk autoincr"`
			Data string `xorm:"UNIQUE NOT NULL"`
		}

		assert.NoError(t, testEngine.Sync2(&OneUnique{}))
		_, _ = testEngine.Exec("DELETE FROM one_unique")

		// Insert with the default value for the unique field
		toInsert := &OneUnique{}
		n, err := testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// but not twice
		toInsert = &OneUnique{}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)

		// Successfully insert test
		toInsert = &OneUnique{Data: "test"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// Successfully insert test2
		toInsert = &OneUnique{Data: "test2"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// Successfully don't reinsert test
		toInsert = &OneUnique{Data: "test"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)
	})

	t.Run("MultiUnique", func(t *testing.T) {
		type MultiUnique struct {
			ID        int64 `xorm:"pk autoincr"`
			NotUnique string
			Data1     string `xorm:"UNIQUE(s) NOT NULL"`
			Data2     string `xorm:"UNIQUE(s) NOT NULL"`
		}

		assert.NoError(t, testEngine.Sync2(&MultiUnique{}))
		_, _ = testEngine.Exec("DELETE FROM multi_unique")

		// Insert with default values
		toInsert := &MultiUnique{}
		n, err := testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// successfully insert test, t1
		toInsert = &MultiUnique{Data1: "test", NotUnique: "t1"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// successfully insert test2, t1
		toInsert = &MultiUnique{Data1: "test2", NotUnique: "t1"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// successfully don't insert test2, t2
		toInsert = &MultiUnique{Data1: "test2", NotUnique: "t2"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)

		// successfully don't insert test, t2
		toInsert = &MultiUnique{Data1: "test", NotUnique: "t2"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)

		// successfully insert test/test2, t2
		toInsert = &MultiUnique{Data1: "test", Data2: "test2", NotUnique: "t1"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// successfully don't insert test/test2, t2
		toInsert = &MultiUnique{Data1: "test", Data2: "test2", NotUnique: "t2"}
		n, err = testEngine.InsertOnConflictDoNothing(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)
	})

	t.Run("MultiMultiUnique", func(t *testing.T) {
		type MultiMultiUnique struct {
			ID    int64  `xorm:"pk autoincr"`
			Data0 string `xorm:"UNIQUE NOT NULL"`
			Data1 string `xorm:"UNIQUE(s) NOT NULL"`
			Data2 string `xorm:"UNIQUE(s) NOT NULL"`
		}

		assert.NoError(t, testEngine.Sync2(&MultiMultiUnique{}))
		_, _ = testEngine.Exec("DELETE FROM multi_multi_unique")

		// Insert with default values
		n, err := testEngine.InsertOnConflictDoNothing(&MultiMultiUnique{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with value for t1, <test, "">
		n, err = testEngine.InsertOnConflictDoNothing(&MultiMultiUnique{Data1: "test", Data0: "t1"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Fail insert with value for t1, <test2, "">
		n, err = testEngine.InsertOnConflictDoNothing(&MultiMultiUnique{Data2: "test2", Data0: "t1"})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Insert with value for t2, <test2, "">
		n, err = testEngine.InsertOnConflictDoNothing(&MultiMultiUnique{Data2: "test2", Data0: "t2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Fail insert with value for t2, <test2, "">
		n, err = testEngine.InsertOnConflictDoNothing(&MultiMultiUnique{Data2: "test2", Data0: "t2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Fail insert with value for t2, <test, "">
		n, err = testEngine.InsertOnConflictDoNothing(&MultiMultiUnique{Data1: "test", Data0: "t2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Insert with value for t3, <test, test2>
		n, err = testEngine.InsertOnConflictDoNothing(&MultiMultiUnique{Data1: "test", Data2: "test2", Data0: "t3"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// fail insert with value for t2, <test, test2>
		n, err = testEngine.InsertOnConflictDoNothing(&MultiMultiUnique{Data1: "test", Data2: "test2", Data0: "t2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
	})

	t.Run("NoPK", func(t *testing.T) {
		type NoPrimaryKey struct {
			NotID   int64
			Uniqued string `xorm:"UNIQUE"`
		}

		assert.NoError(t, testEngine.Sync2(&NoPrimaryKey{}))
		_, _ = testEngine.Exec("DELETE FROM no_primary_unique")

		empty := &NoPrimaryKey{}

		// Insert default
		n, err := testEngine.InsertOnConflictDoNothing(empty)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with 1
		n, err = testEngine.InsertOnConflictDoNothing(&NoPrimaryKey{Uniqued: "1"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Fail reinsert default
		n, err = testEngine.InsertOnConflictDoNothing(&NoPrimaryKey{NotID: 1})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Fail reinsert default
		n, err = testEngine.InsertOnConflictDoNothing(&NoPrimaryKey{NotID: 2})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Insert with 2
		n, err = testEngine.InsertOnConflictDoNothing(&NoPrimaryKey{NotID: 2, Uniqued: "2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Fail reinsert with 2
		n, err = testEngine.InsertOnConflictDoNothing(&NoPrimaryKey{NotID: 1, Uniqued: "2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
	})

	// FIXME: needs tests for Map inserts and multi inserts
}

func TestUpsert(t *testing.T) {
	assert.NoError(t, PrepareEngine())

	t.Run("NoUnique", func(t *testing.T) {
		// Upsert does not work if there is no unique constraint
		type NoUniquesUpsert struct {
			ID   int64 `xorm:"pk autoincr"`
			Data string
		}
		assert.NoError(t, testEngine.Sync(new(NoUniquesUpsert)))

		toInsert := &NoUniquesUpsert{Data: "shouldErr"}
		n, err := testEngine.Upsert(toInsert)
		assert.Error(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)

		toInsert = &NoUniquesUpsert{Data: ""}
		n, err = testEngine.Upsert(toInsert)
		assert.Error(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)
	})

	t.Run("OneUnique", func(t *testing.T) {
		type OneUniqueUpsert struct {
			ID   int64  `xorm:"pk autoincr"`
			Data string `xorm:"UNIQUE NOT NULL"`
		}

		assert.NoError(t, testEngine.Sync2(&OneUniqueUpsert{}))
		_, _ = testEngine.Exec("DELETE FROM one_unique")

		// Insert with the default value for the unique field
		toInsert := &OneUniqueUpsert{}
		n, err := testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// Nothing to update
		toInsert = &OneUniqueUpsert{}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)

		// Successfully insert test
		toInsert = &OneUniqueUpsert{Data: "test"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// Successfully insert test2
		toInsert = &OneUniqueUpsert{Data: "test2"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// Successfully don't reinsert test or update
		toInsert = &OneUniqueUpsert{Data: "test"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)
	})

	t.Run("MultiUnique", func(t *testing.T) {
		type MultiUniqueUpsert struct {
			ID        int64 `xorm:"pk autoincr"`
			NotUnique string
			Data1     string `xorm:"UNIQUE(s) NOT NULL"`
			Data2     string `xorm:"UNIQUE(s) NOT NULL"`
		}

		assert.NoError(t, testEngine.Sync2(&MultiUniqueUpsert{}))
		_, _ = testEngine.Exec("DELETE FROM multi_unique")

		// Insert with default values
		toInsert := &MultiUniqueUpsert{}
		n, err := testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// successfully insert <test> with t1
		testt1 := &MultiUniqueUpsert{Data1: "test", NotUnique: "t1"}
		n, err = testEngine.Upsert(testt1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), testt1.ID)

		// successfully insert <test2> with t1
		test2t1 := &MultiUniqueUpsert{Data1: "test2", NotUnique: "t1"}
		n, err = testEngine.Upsert(test2t1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), test2t1.ID)

		// Update <test2> to t2
		toInsert = &MultiUniqueUpsert{Data1: "test2", NotUnique: "t2"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		if !assert.Equal(t, int64(1), n) {
			uniques := []MultiUniqueUpsert{}
			_ = testEngine.Find(&uniques)
			fmt.Println(uniques)
		}
		assert.Equal(t, test2t1.ID, toInsert.ID)

		// Update <test> to t2
		toInsert = &MultiUniqueUpsert{Data1: "test", NotUnique: "t2"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		if !assert.Equal(t, testt1.ID, toInsert.ID) {
			uniques := []MultiUniqueUpsert{}
			_ = testEngine.Find(&uniques)
			fmt.Println(uniques)
		}

		// Insert <test/test2>, t1
		testtest2t1 := &MultiUniqueUpsert{Data1: "test", Data2: "test2", NotUnique: "t1"}
		n, err = testEngine.Upsert(testtest2t1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), testtest2t1.ID)

		// Update <test/test2> to t2
		toInsert = &MultiUniqueUpsert{Data1: "test", Data2: "test2", NotUnique: "t2"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.Equal(t, testtest2t1.ID, toInsert.ID)
	})

	t.Run("MultiMultiUnique", func(t *testing.T) {
		type MultiMultiUniqueUpsert struct {
			ID        int64 `xorm:"pk autoincr"`
			NotUnique string
			Data0     string `xorm:"UNIQUE NOT NULL"`
			Data1     string `xorm:"UNIQUE(s) NOT NULL"`
			Data2     string `xorm:"UNIQUE(s) NOT NULL"`
		}

		assert.NoError(t, testEngine.Sync2(&MultiMultiUniqueUpsert{}))
		_, _ = testEngine.Exec("DELETE FROM multi_multi_unique")

		// Cannot upsert if there is more than one unique constraint
		n, err := testEngine.Upsert(&MultiMultiUniqueUpsert{})
		assert.Error(t, err)
		assert.Equal(t, int64(0), n)
	})

	t.Run("NoPK", func(t *testing.T) {
		type NoPrimaryKeyUpsert struct {
			NotID   int64
			Uniqued string `xorm:"UNIQUE"`
		}

		assert.NoError(t, testEngine.Sync2(&NoPrimaryKeyUpsert{}))
		_, _ = testEngine.Exec("DELETE FROM no_primary_unique")

		empty := &NoPrimaryKeyUpsert{}

		// Insert default
		n, err := testEngine.Upsert(empty)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with 1
		uniqued1 := &NoPrimaryKeyUpsert{Uniqued: "1"}
		n, err = testEngine.Upsert(uniqued1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Update default
		n, err = testEngine.Upsert(&NoPrimaryKeyUpsert{NotID: 1})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Update default again
		n, err = testEngine.Upsert(&NoPrimaryKeyUpsert{NotID: 2})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with 2
		n, err = testEngine.Upsert(&NoPrimaryKeyUpsert{NotID: 2, Uniqued: "2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Update 2
		n, err = testEngine.Upsert(&NoPrimaryKeyUpsert{NotID: 1, Uniqued: "2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
	})

	t.Run("NoAutoIncrementPK", func(t *testing.T) {
		type NoAutoIncrementPrimaryKey struct {
			Name      string `xorm:"pk"`
			Number    int    `xorm:"pk"`
			NotUnique string
		}

		assert.NoError(t, testEngine.Sync2(&NoAutoIncrementPrimaryKey{}))
		_, _ = testEngine.Exec("DELETE FROM no_primary_unique")

		empty := &NoAutoIncrementPrimaryKey{}

		// Insert default
		n, err := testEngine.Upsert(empty)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with 1
		one := &NoAutoIncrementPrimaryKey{Name: "one", Number: 1}
		n, err = testEngine.Upsert(one)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Update default
		n, err = testEngine.Upsert([]*NoAutoIncrementPrimaryKey{{NotUnique: "notunique"}})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Update again
		n, err = testEngine.Upsert(&NoAutoIncrementPrimaryKey{NotUnique: "again"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with 2
		n, err = testEngine.Upsert([]*NoAutoIncrementPrimaryKey{{Name: "two", Number: 2}})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Fail reinsert with 2
		n, err = testEngine.Upsert(&NoAutoIncrementPrimaryKey{Name: "one", Number: 1, NotUnique: "updated"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Upsert multiple with 2
		n, err = testEngine.Upsert([]*NoAutoIncrementPrimaryKey{{Name: "one", Number: 1, NotUnique: "updatedagain"}, {Name: "three", Number: 3}})
		assert.NoError(t, err)
		assert.Equal(t, int64(2), n)
	})
}

func TestInsertOnConflictDoNothingMap(t *testing.T) {
	type MultiUniqueMap struct {
		ID        int64 `xorm:"pk autoincr"`
		NotUnique string
		Data1     string `xorm:"UNIQUE(s) NOT NULL"`
		Data2     string `xorm:"UNIQUE(s) NOT NULL"`
	}

	assert.NoError(t, testEngine.Sync2(&MultiUniqueMap{}))
	_, _ = testEngine.Exec("DELETE FROM multi_unique_map")

	n, err := testEngine.Table(&MultiUniqueMap{}).InsertOnConflictDoNothing(map[string]interface{}{
		"not_unique": "",
		"data1":      "",
		"data2":      "",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = testEngine.Table(&MultiUniqueMap{}).InsertOnConflictDoNothing(map[string]interface{}{
		"not_unique": "",
		"data1":      "second",
		"data2":      "",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = testEngine.Table(&MultiUniqueMap{}).InsertOnConflictDoNothing(map[string]interface{}{
		"not_unique": "",
		"data1":      "",
		"data2":      "",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)

	n, err = testEngine.Table(&MultiUniqueMap{}).InsertOnConflictDoNothing(map[string]interface{}{
		"not_unique": "",
		"data1":      "",
		"data2":      "third",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = testEngine.Table(&MultiUniqueMap{}).InsertOnConflictDoNothing(map[string]interface{}{
		"not_unique": "",
		"data1":      "",
		"data2":      "third",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestUpsertMap(t *testing.T) {
	type MultiUniqueMapUpsert struct {
		ID        int64 `xorm:"pk autoincr"`
		NotUnique string
		Data1     string `xorm:"UNIQUE(s) NOT NULL"`
		Data2     string `xorm:"UNIQUE(s) NOT NULL"`
	}

	assert.NoError(t, testEngine.Sync2(&MultiUniqueMapUpsert{}))
	_, _ = testEngine.Exec("DELETE FROM multi_unique_map_upsert")

	n, err := testEngine.Table(&MultiUniqueMapUpsert{}).Upsert(map[string]interface{}{
		"not_unique": "",
		"data1":      "",
		"data2":      "",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)

	testCase := &MultiUniqueMapUpsert{}
	has, err := testEngine.Get(testCase)
	assert.NoError(t, err)
	assert.True(t, has)

	n, err = testEngine.Table(&MultiUniqueMapUpsert{}).Upsert(map[string]interface{}{
		"not_unique": "",
		"data1":      "second",
		"data2":      "",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)
	testCase = &MultiUniqueMapUpsert{
		Data1: "second",
	}
	has, err = testEngine.Get(testCase)
	assert.NoError(t, err)
	assert.True(t, has)

	n, err = testEngine.Table(&MultiUniqueMapUpsert{}).Upsert(map[string]interface{}{
		"not_unique": "updated",
		"data1":      "",
		"data2":      "",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)
	testCase = &MultiUniqueMapUpsert{
		Data1: "",
	}
	has, err = testEngine.Get(testCase)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, "updated", testCase.NotUnique)

	n, err = testEngine.Table(&MultiUniqueMapUpsert{}).Upsert(map[string]interface{}{
		"not_unique": "",
		"data1":      "",
		"data2":      "third",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)
	testCase = &MultiUniqueMapUpsert{
		Data2: "third",
	}
	has, err = testEngine.Get(testCase)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, "", testCase.NotUnique)

	n, err = testEngine.Table(&MultiUniqueMapUpsert{}).Upsert(map[string]interface{}{
		"not_unique": "updated",
		"data1":      "",
		"data2":      "third",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)
	testCase = &MultiUniqueMapUpsert{
		Data2: "third",
	}
	has, err = testEngine.Get(testCase)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, "updated", testCase.NotUnique)

	testCase = &MultiUniqueMapUpsert{
		Data1: "second",
	}
	has, err = testEngine.Get(testCase)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, "", testCase.NotUnique)
}
