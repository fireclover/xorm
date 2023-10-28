// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"context"
	"database/sql"

	"xorm.io/xorm/v2"
)

type Executor[T any] struct {
	client xorm.Interface
}

func New[T any](c xorm.Interface) *Executor[T] {
	return &Executor[T]{
		client: c,
	}
}

func (q *Executor[T]) Exec(ctx context.Context) (sql.Result, error) {
	return q.client.Exec()
}

func (q *Executor[T]) All(ctx context.Context) ([]T, error) {
	var result []T
	return result, q.client.Find(&result)
}

type Filter interface{}

func (q *Executor[T]) Filter(ctx context.Context, filter ...Filter) ([]T, error) {
	// implementation
	return nil, nil
}
