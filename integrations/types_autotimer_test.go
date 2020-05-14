// Copyright 2017 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integrations

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type Timestamp int64

func FromTimeToTimestamp(t time.Time) Timestamp {
	return Timestamp(int64(t.UnixNano()) / 1e6)
}

func (t Timestamp) String() string {
	return strconv.FormatInt(int64(t), 10)
}
func (t *Timestamp) Time() time.Time {
	return time.Unix(int64(*t)/1e3, (int64(*t)%1e3)*1e6)
}

func (t *Timestamp) FromDB(b []byte) error {
	var err error
	var value int64
	value, err = strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return nil
	}
	*t = Timestamp(value)
	return nil
}

func (t *Timestamp) ToDB() ([]byte, error) {
	if t == nil {
		return nil, nil
	}
	data := strconv.FormatInt(int64(*t), 10)
	if len(data) == 0 {
		return []byte("0"), nil
	}
	return []byte(data), nil
}

func (t *Timestamp) AutoTime(now time.Time) (interface{}, error) {
	data := int64(now.UnixNano()) / 1e6
	return data, nil
}

type AutoTimerStruct struct {
	ID          string `xorm:"varchar(20) pk unique 'id'" json:"id"`
	Name        string `xorm:"varchar(100) notnull" json:"username"`
	Description string `xorm:"text" json:"description"`

	CreatedAt Timestamp  `xorm:"created bigint notnull" json:"created_at"`
	UpdatedAt Timestamp  `xorm:"updated bigint notnull" json:"updated_at"`
	DeletedAt *Timestamp `xorm:"deleted bigint null" json:"deleted_at"`
}

func TestAutoTimerStructInsert(t *testing.T) {
	assert.NoError(t, PrepareEngine())
	assertSync(t, new(AutoTimerStruct))
	// insert
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	item := AutoTimerStruct{
		ID:   id,
		Name: "AutoTimer Test:Insert",
	}
	_, err := testEngine.Insert(&item)
	assert.NoError(t, err)
	assert.EqualValues(t, id, item.ID)
	// get
	var result AutoTimerStruct
	has, err := testEngine.ID(id).Get(&result)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.NotEmpty(t, result.CreatedAt)
	assert.NotEmpty(t, result.UpdatedAt)
	assert.Nil(t, item.DeletedAt)
}
func TestAutoTimerStructUpdate(t *testing.T) {
	assert.NoError(t, PrepareEngine())
	assertSync(t, new(AutoTimerStruct))
	// insert
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	item := AutoTimerStruct{
		ID:   id,
		Name: "AutoTimer Test:Update",
	}
	_, err := testEngine.Insert(&item)
	assert.NoError(t, err)
	// update
	item.Description = "updated"
	time.Sleep(50 * time.Millisecond)
	_, err = testEngine.ID(id).Update(&item)
	assert.NoError(t, err)
	assert.Greater(t, item.UpdatedAt, item.CreatedAt)
	assert.Nil(t, item.DeletedAt)
}

func TestAutoTimerStructDelete(t *testing.T) {
	assert.NoError(t, PrepareEngine())
	assertSync(t, new(AutoTimerStruct))
	// insert
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	item := AutoTimerStruct{
		ID:   id,
		Name: "AutoTimer Test:Delete",
	}
	_, err := testEngine.Insert(&item)
	assert.NoError(t, err)
	// delete
	_, err = testEngine.ID(id).Delete(&item)
	assert.NoError(t, err)
	// get
	var result AutoTimerStruct
	has, err := testEngine.ID(id).Get(&result)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.NotEmpty(t, result.CreatedAt)
	assert.NotEmpty(t, result.UpdatedAt)
	assert.NotEmpty(t, result.DeletedAt)
}
