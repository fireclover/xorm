// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"context"
	"testing"

	"xorm.io/builder"
	"xorm.io/xorm/v2"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestExecutor(t *testing.T) {
	type User struct {
		Id   int64
		Name string
	}

	engine, err := xorm.NewEngine("sqlite3", "file::memory:?cache=shared")
	assert.NoError(t, err)
	assert.NoError(t, engine.Sync(new(User)))

	// create querier
	executor := New[User](engine)

	err = executor.InsertOne(context.Background(), &User{
		Name: "test",
	})
	assert.NoError(t, err)

	user, err := executor.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, user.Name, "test")
	assert.Equal(t, user.Id, int64(1))

	users, err := executor.All(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, len(users), 1)

	users, err = executor.Where(builder.Eq{"id": 1}).All(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, len(users), 1)

	err = executor.InsertMap(context.Background(), map[string]any{
		"name": "test2",
	})
	assert.NoError(t, err)

	users, err = executor.All(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, len(users), 2)
	assert.Equal(t, "test", users[0].Name)
	assert.Equal(t, "test2", users[1].Name)
}
