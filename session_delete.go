// Copyright 2016 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"errors"
	"strconv"
	"time"

	"xorm.io/xorm/caches"
	"xorm.io/xorm/schemas"
)

func (session *Session) cacheDelete(table *schemas.Table, tableName, sqlStr string, args ...interface{}) error {
	if table == nil ||
		session.tx != nil {
		return ErrCacheFailed
	}

	for _, filter := range session.engine.dialect.Filters() {
		sqlStr = filter.Do(sqlStr)
	}

	newsql := session.statement.ConvertIDSQL(sqlStr)
	if newsql == "" {
		return ErrCacheFailed
	}

	cacher := session.engine.cacherMgr.GetCacher(tableName)
	pkColumns := table.PKColumns()
	ids, err := caches.GetCacheSql(cacher, tableName, newsql, args)
	if err != nil {
		resultsSlice, err := session.queryBytes(newsql, args...)
		if err != nil {
			return err
		}
		ids = make([]schemas.PK, 0)
		if len(resultsSlice) > 0 {
			for _, data := range resultsSlice {
				var id int64
				var pk schemas.PK = make([]interface{}, 0)
				for _, col := range pkColumns {
					if v, ok := data[col.Name]; !ok {
						return errors.New("no id")
					} else if col.SQLType.IsText() {
						pk = append(pk, string(v))
					} else if col.SQLType.IsNumeric() {
						id, err = strconv.ParseInt(string(v), 10, 64)
						if err != nil {
							return err
						}
						pk = append(pk, id)
					} else {
						return errors.New("not supported primary key type")
					}
				}
				ids = append(ids, pk)
			}
		}
	}

	for _, id := range ids {
		session.engine.logger.Debugf("[cache] delete cache obj: %v, %v", tableName, id)
		sid, err := id.ToString()
		if err != nil {
			return err
		}
		cacher.DelBean(tableName, sid)
	}
	session.engine.logger.Debugf("[cache] clear cache table: %v", tableName)
	cacher.ClearIds(tableName)
	return nil
}

// Delete records, bean's non-empty fields are conditions
func (session *Session) Delete(bean interface{}) (int64, error) {
	if session.isAutoClose {
		defer session.Close()
	}

	if session.statement.LastError != nil {
		return 0, session.statement.LastError
	}

	if err := session.statement.SetRefBean(bean); err != nil {
		return 0, err
	}

	// handle before delete processors
	for _, closure := range session.beforeClosures {
		closure(bean)
	}
	cleanupProcessorsClosures(&session.beforeClosures)

	if processor, ok := interface{}(bean).(BeforeDeleteProcessor); ok {
		processor.BeforeDelete()
	}

	realSQL, deleteSQL, condArgs, now, err := session.statement.GenDeleteSQL(bean)
	if err != nil {
		return 0, err
	}

	argsForCache := make([]interface{}, 0, len(condArgs)*2)
	copy(argsForCache, condArgs)
	argsForCache = append(condArgs, argsForCache...)

	if !session.statement.GetUnscoped() && session.statement.RefTable.DeletedColumn() != nil {
		deletedColumn := session.statement.RefTable.DeletedColumn()

		session.afterClosures = append(session.afterClosures, func(col *schemas.Column, t time.Time) func(interface{}) {
			return func(bean interface{}) {
				setColumnTime(bean, col, t)
			}
		}(deletedColumn, now.In(session.engine.TZLocation)))
	}

	var tableNameNoQuote = session.statement.TableName()
	if cacher := session.engine.GetCacher(tableNameNoQuote); cacher != nil && session.statement.UseCache {
		session.cacheDelete(session.statement.RefTable, tableNameNoQuote, deleteSQL, argsForCache...)
	}

	res, err := session.exec(realSQL, condArgs...)
	if err != nil {
		return 0, err
	}

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
		if lenAfterClosures > 0 {
			if value, has := session.afterDeleteBeans[bean]; has && value != nil {
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
	cleanupProcessorsClosures(&session.afterClosures)
	// --

	return res.RowsAffected()
}
