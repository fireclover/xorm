// Copyright 2018 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialects

import (
	"testing"

	"xorm.io/xorm/names"

	"github.com/stretchr/testify/assert"
)

type MCC struct {
	ID          int64  `xorm:"pk 'id'"`
	Code        string `xorm:"'code'"`
	Description string `xorm:"'description'"`
}

func (mcc *MCC) TableName() string {
	return "mcc"
}

func TestTableName1(t *testing.T) {
	dialect := QueryDialect("mysql")

	assert.EqualValues(t, "mcc", TableName(dialect, names.SnakeMapper{}, MCC))
	assert.EqualValues(t, "mcc", TableName(dialect, names.SnakeMapper{}, "mcc"))
}
