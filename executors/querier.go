// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executors

import (
	"context"

	"xorm.io/xorm/v2"
)

type Querier[T any] struct {
	client xorm.Interface
}

func NewQuerier[T any](c xorm.Interface) *Querier[T] {
	return &Querier[T]{
		client: c,
	}
}

func (q *Querier[T]) All(ctx context.Context) ([]T, error) {
	var result []T
	return result, q.client.Find(&result)
}

type Filter interface{}

func (q *Querier[T]) Filter(ctx context.Context, filter ...Filter) ([]T, error) {
	// implementation
	return nil, nil
}
