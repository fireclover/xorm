// Copyright 2018 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialects

import (
	"context"
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

func TestFullTableName(t *testing.T) {
	dialect, err := OpenDialect("mysql", "root:root@tcp(127.0.0.1:3306)/test?charset=utf8")
	if err != nil {
		panic("unknow dialect")
	}
	dialect.SetShadowable(NewTrueShadow())
	assert.EqualValues(t, "shadow_test.mcc", FullTableName(context.Background(), dialect, names.SnakeMapper{}, &MCC{}))
	assert.EqualValues(t, "shadow_test.mcc", FullTableName(context.Background(), dialect, names.SnakeMapper{}, "mcc"))
	dialect.SetShadowable(NewFalseShadow())
	assert.EqualValues(t, "mcc", FullTableName(context.Background(), dialect, names.SnakeMapper{}, &MCC{}))
	assert.EqualValues(t, "mcc", FullTableName(context.Background(), dialect, names.SnakeMapper{}, "mcc"))
}
