// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialects

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm/schemas"
)

type dialect struct {
	dbType schemas.DBType
	Dialect
}

func (d dialect) URI() *URI {
	return &URI{
		DBType: d.dbType,
	}
}

func TestFormatTime(t *testing.T) {
	date := time.Date(2020, 10, 23, 10, 14, 15, 123456, time.Local)
	tests := []struct {
		name        string
		dialect     Dialect
		sqlTypeName string
		t           time.Time
		want        interface{}
	}{
		{
			"test time",
			dialect{},
			schemas.Time,
			date,
			date.Format("2006-01-02 15:04:05")[11:19],
		},
		{
			"test date",
			dialect{},
			schemas.Date,
			date,
			date.Format("2006-01-02"),
		},
		{
			"test varchar",
			dialect{},
			schemas.Varchar,
			date,
			date.Format("2006-01-02 15:04:05"),
		},
		{
			"test timestamp and postgres",
			dialect{dbType: schemas.POSTGRES},
			schemas.TimeStamp,
			date,
			date.Format("2006-01-02T15:04:05.999999"),
		},
		{
			"test datetime and mysql",
			dialect{dbType: schemas.MYSQL},
			schemas.DateTime,
			date,
			date.Format("2006-01-02T15:04:05.999999"),
		},
		{
			"test datetime",
			dialect{},
			schemas.DateTime,
			date,
			date.Format("2006-01-02 15:04:05"),
		},
		{
			"test timestampz",
			dialect{dbType: schemas.MSSQL},
			schemas.TimeStampz,
			date,
			date.Format("2006-01-02T15:04:05.9999999Z07:00"),
		},
		{
			"test bigint",
			dialect{},
			schemas.BigInt,
			date,
			date.Unix(),
		},
		{
			"test int",
			dialect{},
			schemas.Int,
			date,
			date.Unix(),
		},
		{
			"test default",
			dialect{},
			"",
			date,
			date,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTime(tt.dialect, tt.sqlTypeName, tt.t)
			assert.Equal(t, tt.want, got)
		})
	}
}
