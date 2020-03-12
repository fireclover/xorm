// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm/log"
)

func TestEngineGroup(t *testing.T) {
	assert.NoError(t, prepareEngine())

	master := testEngine.(*Engine)
	eg, err := NewEngineGroup(master, []*Engine{master})
	assert.NoError(t, err)

	eg.SetMaxIdleConns(10)
	eg.SetMaxOpenConns(100)
	eg.SetTableMapper(master.GetTableMapper())
	eg.SetColumnMapper(master.GetColumnMapper())
	eg.SetLogLevel(log.LOG_INFO)
	eg.ShowSQL(true)
}
