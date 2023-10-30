// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"context"
	"database/sql"

	"xorm.io/builder"
	"xorm.io/xorm/v2"
)

type Executor[T any] struct {
	client   xorm.Interface
	tableObj *T
}

func New[T any](c xorm.Interface) *Executor[T] {
	return &Executor[T]{
		client:   c,
		tableObj: new(T),
	}
}

func (q *Executor[T]) Where(cond builder.Cond) *Executor[T] {
	q.client.Where(cond)
	return q
}

func (q *Executor[T]) Get(ctx context.Context) (*T, error) {
	var result T
	if has, err := q.client.Get(&result); err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	return &result, nil
}

func (q *Executor[T]) InsertOne(ctx context.Context, obj *T) error {
	_, err := q.client.InsertOne(obj)
	return err
}

func (q *Executor[T]) InsertMap(ctx context.Context, result map[string]any) error {
	_, err := q.client.Table(q.tableObj).Insert(result)
	return err
}

func (q *Executor[T]) Insert(ctx context.Context, results []T) error {
	_, err := q.client.Insert(results)
	return err
}

func (q *Executor[T]) InsertMaps(ctx context.Context, results []map[string]any) error {
	_, err := q.client.Table(q.tableObj).Insert(results)
	return err
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
