// Copyright 2017 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"strconv"

	"github.com/stretchr/testify/assert"
)

const (
	// DatetimeFormat
	DatetimeFormat = "2006-01-02 15:04:05"
	// DateFormat
	DateFormat = "2006-01-02"
	// TimeFormat
	TimeFormat = "15:04:05"
)

var (
	// TimeFormats
	TimeFormats = []string{
		DateFormat,
		DatetimeFormat,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		time.Kitchen,
		time.Stamp,
		time.StampMilli,
		time.StampMicro,
		time.StampNano,
		"2006年01月02日",
		"2006年1月2日",
	}
)

// Time custom time
type Time time.Time

func (t Time) String() string {
	return t.Origin().Format(time.RFC3339Nano)
}

func (t Time) MarshalJSON() ([]byte, error) {
	ot := time.Time(t)
	return json.Marshal(ot)
}

func (t *Time) UnmarshalJSON(b []byte) error {
	var err error
	var ot time.Time
	var value string
	if err = json.Unmarshal(b, &value); err != nil {
		return err
	}
	for _, layout := range TimeFormats {
		ot, err = time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			break
		}
	}
	*t = Time(ot)
	return err
}

// Origin origin time(time.Time)
func (t *Time) Origin() *time.Time {
	if t != nil {
		ot := time.Time(*t)
		return &ot
	}
	return nil
}

// TimeFrom
func TimeFrom(t time.Time) Time {
	return Time(t)
}

func (t *Time) FromDB(b []byte) error {
	var err error
	var ot time.Time
	var value string
	value = string(b)
	if len(b) > 0 {
		for _, layout := range TimeFormats {
			ot, err = time.ParseInLocation(layout, value, time.Local)
			if err == nil {
				break
			}
		}
		*t = Time(ot)
		return err
	}
	return nil
}
func (t *Time) ToDB() ([]byte, error) {
	if t == nil {
		return nil, nil
	}
	str := t.String()
	log.Printf("[Time.ToDB] t.2=%+v\n", str)
	if str == "" {
		return nil, nil
	}
	return []byte(str), nil

}

// JSON 统一JSON处理
type JSON string

func (j *JSON) FromDB(b []byte) error {
	if len(b) > 0 {
		*j = JSON(string(b))
		return nil
	}
	return nil
}
func (j *JSON) ToDB() ([]byte, error) {
	if j == nil {
		return nil, nil
	}
	str := string(*j)
	if str == "" {
		return []byte("{}"), nil
	}
	return []byte(str), nil
}

type PointerModel struct {
	ID          string  `xorm:"varchar(20) pk unique 'id'" json:"id"`
	Username    string  `xorm:"varchar(100) notnull" json:"username"`
	Nickname    *string `xorm:"varchar(50) null" json:"nickname"`
	Like        JSON    `xorm:"json default('{}')" json:"like"`
	TJson       *JSON   `xorm:"json null" json:"t_json"`
	TNumeric    float64 `xorm:"numeric"`
	Description string  `xorm:"text" json:"description"`
	FoundedDate *Time   `xorm:"date null" json:"founded_date"`
	FetchedAt   *Time   `xorm:"timestampz null" json:"fetched_at"`
	StartTime   Time    `xorm:"timestampz null" json:"start_time"`
	EndTime     *Time   `xorm:"timestamp null" json:"end_time"`
	IsApproved  bool    `xorm:"default(false)" json:"is_approved"`

	CreatedAt Time  `xorm:"timestampz created notnull" json:"created_at"`
	UpdatedAt Time  `xorm:"updated timestampz notnull" json:"updated_at"`
	DeletedAt *Time `xorm:"deleted timestampz" json:"deleted_at"`
}

// TableName return table name
func (m PointerModel) TableName() string {
	return "pointer_model"
}

func TestPointerModelInsertUpdate(t *testing.T) {
	assert.NoError(t, prepareEngine())
	assertSync(t, new(PointerModel))

	now := time.Now()
	fd := TimeFrom(now)
	ct := TimeFrom(now)
	et := TimeFrom(now.Add(time.Minute * 15))
	id := strconv.FormatInt(now.UnixNano(), 10)
	item := PointerModel{
		ID:          id,
		Username:    "pinter property insert test",
		FoundedDate: &fd,
		StartTime:   et,
		// EndTime:   et,
		CreatedAt: ct,
		UpdatedAt: ct,
	}
	// insert
	_, err := testEngine.Insert(&item)
	if err != nil {
		t.Fatal("insert", err)
	}
	var result PointerModel
	_, err = testEngine.ID(id).Get(&result)
	if err != nil {
		t.Fatal("get", err)
	}
	assert.NotNil(t, result.FoundedDate)
	assert.Nil(t, result.FetchedAt)
	assert.Nil(t, result.EndTime)
	t.Logf("[Get] insert result=%+v\n", result)

	// update
	result.FoundedDate = nil
	result.FetchedAt = &fd
	result.EndTime = &et
	item.Username = "pointer property update test"
	_, err = testEngine.Where("id=?", id).AllCols().Update(&result)
	if err != nil {
		t.Fatal("update", err)
	}
	var upResult PointerModel
	_, err = testEngine.ID(id).Get(&upResult)
	if err != nil {
		t.Fatal("get", err)
	}
	assert.Nil(t, upResult.FoundedDate)
	assert.NotNil(t, upResult.FetchedAt)
	assert.NotNil(t, upResult.EndTime)
	t.Logf("[Get] update result=%+v\n", upResult)
}
