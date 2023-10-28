// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executors

import (
	"context"
	"testing"

	"xorm.io/xorm/v2"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestQuerier(t *testing.T) {
	type User struct {
		Id   int64
		Name string
	}

	engine, err := xorm.NewEngine("sqlite3", "file::memory:?cache=shared")
	assert.NoError(t, err)
	assert.NoError(t, engine.Sync(new(User)))

	// create querier
	querier := NewQuerier[User](engine)

	users, err := querier.All(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(users), 0)
}
