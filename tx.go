// Copyright 2016 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"hash/crc32"

	"xorm.io/xorm/v2/internal/core"
)

func (session *Session) doPrepareTx(sqlStr string) (stmt *core.Stmt, err error) {
	crc := crc32.ChecksumIEEE([]byte(sqlStr))
	// TODO try hash(sqlStr+len(sqlStr))
	var has bool
	stmt, has = session.txStmtCache[crc]
	if !has {
		stmt, err = session.tx.PrepareContext(session.ctx, sqlStr)
		if err != nil {
			return nil, err
		}
		session.txStmtCache[crc] = stmt
	}
	return
}

// Begin a transaction
func (session *Session) Begin() error {
	if session.isAutoCommit {
		tx, err := session.db().BeginTx(session.ctx, nil)
		if err != nil {
			return err
		}
		session.isAutoCommit = false
		session.isCommitedOrRollbacked = false
		session.tx = tx

		session.saveLastSQL("BEGIN TRANSACTION")
	}
	return nil
}

// Rollback When using transaction, you can rollback if any error
func (session *Session) Rollback() error {
	if !session.isAutoCommit && !session.isCommitedOrRollbacked {
		session.saveLastSQL("ROLL BACK")
		session.isCommitedOrRollbacked = true
		session.isAutoCommit = true

		return session.tx.Rollback()
	}
	return nil
}

// Commit When using transaction, Commit will commit all operations.
func (session *Session) Commit() error {
	if !session.isAutoCommit && !session.isCommitedOrRollbacked {
		session.saveLastSQL("COMMIT")
		session.isCommitedOrRollbacked = true
		session.isAutoCommit = true

		if err := session.tx.Commit(); err != nil {
			return err
		}

		// handle processors after tx committed
		closureCallFunc := func(closuresPtr *[]func(any), bean any) {
			if closuresPtr != nil {
				for _, closure := range *closuresPtr {
					closure(bean)
				}
			}
		}

		for bean, closuresPtr := range session.afterInsertBeans {
			closureCallFunc(closuresPtr, bean)

			if processor, ok := any(bean).(AfterInsertProcessor); ok {
				processor.AfterInsert()
			}
		}
		for bean, closuresPtr := range session.afterUpdateBeans {
			closureCallFunc(closuresPtr, bean)

			if processor, ok := any(bean).(AfterUpdateProcessor); ok {
				processor.AfterUpdate()
			}
		}
		for bean, closuresPtr := range session.afterDeleteBeans {
			closureCallFunc(closuresPtr, bean)

			if processor, ok := any(bean).(AfterDeleteProcessor); ok {
				processor.AfterDelete()
			}
		}
		cleanUpFunc := func(slices *map[any]*[]func(any)) {
			if len(*slices) > 0 {
				*slices = make(map[any]*[]func(any))
			}
		}
		cleanUpFunc(&session.afterInsertBeans)
		cleanUpFunc(&session.afterUpdateBeans)
		cleanUpFunc(&session.afterDeleteBeans)
	}
	return nil
}

// IsInTx if current session is in a transaction
func (session *Session) IsInTx() bool {
	return !session.isAutoCommit
}
