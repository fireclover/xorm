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
		type NoUniques struct {
			ID   int64 `xorm:"pk autoincr"`
			Data string
		}
		assert.NoError(t, testEngine.Sync(new(NoUniques)))

		toInsert := &NoUniques{Data: "shouldErr"}
		n, err := testEngine.Upsert(toInsert)
		assert.Error(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)

		toInsert = &NoUniques{Data: ""}
		n, err = testEngine.Upsert(toInsert)
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
		n, err := testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// Nothing to update
		toInsert = &OneUnique{}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)
		assert.Equal(t, int64(0), toInsert.ID)

		// Successfully insert test
		toInsert = &OneUnique{Data: "test"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// Successfully insert test2
		toInsert = &OneUnique{Data: "test2"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// Successfully don't reinsert test or update
		toInsert = &OneUnique{Data: "test"}
		n, err = testEngine.Upsert(toInsert)
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
		n, err := testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), toInsert.ID)

		// successfully insert <test> with t1
		testt1 := &MultiUnique{Data1: "test", NotUnique: "t1"}
		n, err = testEngine.Upsert(testt1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), testt1.ID)

		// successfully insert <test2> with t1
		test2t1 := &MultiUnique{Data1: "test2", NotUnique: "t1"}
		n, err = testEngine.Upsert(test2t1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), test2t1.ID)

		// Update <test2> to t2
		toInsert = &MultiUnique{Data1: "test2", NotUnique: "t2"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		if !assert.Equal(t, int64(1), n) {
			uniques := []MultiUnique{}
			_ = testEngine.Find(&uniques)
			fmt.Println(uniques)
		}
		assert.Equal(t, test2t1.ID, toInsert.ID)

		// Update <test> to t2
		toInsert = &MultiUnique{Data1: "test", NotUnique: "t2"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		if !assert.Equal(t, testt1.ID, toInsert.ID) {
			uniques := []MultiUnique{}
			_ = testEngine.Find(&uniques)
			fmt.Println(uniques)
		}

		// Insert <test/test2>, t1
		testtest2t1 := &MultiUnique{Data1: "test", Data2: "test2", NotUnique: "t1"}
		n, err = testEngine.Upsert(testtest2t1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.NotEqual(t, int64(0), testtest2t1.ID)

		// Update <test/test2> to t2
		toInsert = &MultiUnique{Data1: "test", Data2: "test2", NotUnique: "t2"}
		n, err = testEngine.Upsert(toInsert)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
		assert.Equal(t, testtest2t1.ID, toInsert.ID)
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
		n, err := testEngine.Upsert(&MultiMultiUnique{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with value for t1, <test, "">
		n, err = testEngine.Upsert(&MultiMultiUnique{Data1: "test", Data0: "t1"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Fail insert with value for t1, <test2, "">
		n, err = testEngine.Upsert(&MultiMultiUnique{Data2: "test2", Data0: "t1"})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Insert with value for t2, <test2, "">
		n, err = testEngine.Upsert(&MultiMultiUnique{Data2: "test2", Data0: "t2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Fail insert with value for t2, <test2, "">
		n, err = testEngine.Upsert(&MultiMultiUnique{Data2: "test2", Data0: "t2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Fail insert with value for t2, <test, "">
		n, err = testEngine.Upsert(&MultiMultiUnique{Data1: "test", Data0: "t2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Insert with value for t3, <test, test2>
		n, err = testEngine.Upsert(&MultiMultiUnique{Data1: "test", Data2: "test2", Data0: "t3"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// fail insert with value for t2, <test, test2>
		n, err = testEngine.Upsert(&MultiMultiUnique{Data1: "test", Data2: "test2", Data0: "t2"})
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
		n, err := testEngine.Upsert(empty)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with 1
		uniqued1 := &NoPrimaryKey{Uniqued: "1"}
		n, err = testEngine.Upsert(uniqued1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Update default
		n, err = testEngine.Upsert(&NoPrimaryKey{NotID: 1})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Update default again
		n, err = testEngine.Upsert(&NoPrimaryKey{NotID: 2})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Insert with 2
		n, err = testEngine.Upsert(&NoPrimaryKey{NotID: 2, Uniqued: "2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// Update 2
		n, err = testEngine.Upsert(&NoPrimaryKey{NotID: 1, Uniqued: "2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), n)
	})
}
