// Copyright 2016 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"errors"

	"xorm.io/builder"
)

// ErrNeedDeletedCond delete needs less one condition error
var ErrNeedDeletedCond = errors.New("Delete action needs at least one condition")

// Delete records, bean's non-empty fields are conditions
// At least one condition must be set.
func (session *Session) Delete(beans ...interface{}) (int64, error) {
	return session.delete(beans, true)
}

// Truncate records, bean's non-empty fields are conditions
// In contrast to Delete this method allows deletes without conditions.
func (session *Session) Truncate(beans ...interface{}) (int64, error) {
	return session.delete(beans, false)
}

func (session *Session) delete(beans []interface{}, mustHaveConditions bool) (int64, error) {
	if session.isAutoClose {
		defer session.Close()
	}

	if session.statement.LastError != nil {
		return 0, session.statement.LastError
	}

	var (
		err  error
		bean interface{}
	)
	if len(beans) > 0 {
		bean = beans[0]
		if err = session.statement.SetRefBean(bean); err != nil {
			return 0, err
		}

		executeBeforeClosures(session, bean)

		if processor, ok := interface{}(bean).(BeforeDeleteProcessor); ok {
			processor.BeforeDelete()
		}

		if err = session.statement.MergeConds(bean); err != nil {
			return 0, err
		}
	}

	pLimitN := session.statement.LimitN
	if mustHaveConditions && !session.statement.Conds().IsValid() && (pLimitN == nil || *pLimitN == 0) {
		return 0, ErrNeedDeletedCond
	}

	table := session.statement.RefTable

	realSQLWriter := builder.NewWriter()
	if err := session.statement.WriteDelete(realSQLWriter, session.engine.nowTime); err != nil {
		return 0, err
	}

	// if tag "deleted" is enabled, then set the field as deleted value
	if !session.statement.GetUnscoped() && table != nil && table.DeletedColumn() != nil {
		deletedColumn := table.DeletedColumn()
		_, t, err := session.engine.nowTime(deletedColumn)
		if err != nil {
			return 0, err
		}

		colName := deletedColumn.Name
		session.afterClosures = append(session.afterClosures, func(bean interface{}) {
			col := table.GetColumn(colName)
			setColumnTime(bean, col, t)
		})
	}

	session.statement.RefTable = table
	res, err := session.exec(realSQLWriter.String(), realSQLWriter.Args()...)
	if err != nil {
		return 0, err
	}

	if bean != nil {
		// handle after delete processors
		if session.isAutoCommit {
			for _, closure := range session.afterClosures {
				closure(bean)
			}
			if processor, ok := interface{}(bean).(AfterDeleteProcessor); ok {
				processor.AfterDelete()
			}
		} else {
			lenAfterClosures := len(session.afterClosures)
			if lenAfterClosures > 0 && len(beans) > 0 {
				if value, has := session.afterDeleteBeans[beans[0]]; has && value != nil {
					*value = append(*value, session.afterClosures...)
				} else {
					afterClosures := make([]func(interface{}), lenAfterClosures)
					copy(afterClosures, session.afterClosures)
					session.afterDeleteBeans[bean] = &afterClosures
				}
			} else {
				if _, ok := interface{}(bean).(AfterDeleteProcessor); ok {
					session.afterDeleteBeans[bean] = nil
				}
			}
		}
	}
	cleanupProcessorsClosures(&session.afterClosures)
	// --

	return res.RowsAffected()
}
