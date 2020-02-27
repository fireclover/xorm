// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"context"
	"fmt"
	"time"
)

// LogContext represents a log context
type LogContext struct {
	Ctx         context.Context
	LogLevel    LogLevel
	LogData     string        // log content or SQL
	Args        []interface{} // if it's a SQL, it's the arguments
	IsSQL       bool
	ExecuteTime time.Duration
}

// ContextLogger represents a logger interface with context
type ContextLogger interface {
	Debug(ctx LogContext)
	Error(ctx LogContext)
	Info(ctx LogContext)
	Warn(ctx LogContext)
	Before(context LogContext)

	Level() LogLevel
	SetLevel(l LogLevel)

	ShowSQL(show ...bool)
	IsShowSQL() bool
}

var (
	_ ContextLogger = &LoggerAdapter{}
)

// LoggerAdapter wraps a Logger interafce as LoggerContext interface
type LoggerAdapter struct {
	logger Logger
}

func (l *LoggerAdapter) Before(ctx LogContext) {}

func (l *LoggerAdapter) Debug(ctx LogContext) {
	l.logger.Debug(ctx.LogData)
}

func (l *LoggerAdapter) Error(ctx LogContext) {
	l.logger.Error(ctx.LogData)
}

func (l *LoggerAdapter) Info(ctx LogContext) {
	if !l.logger.IsShowSQL() && ctx.IsSQL {
		return
	}
	if ctx.IsSQL {
		l.logger.Info(fmt.Sprintf("[SQL] %v %v", ctx.LogData, ctx.Args))
	} else {
		l.logger.Info(ctx.LogData)
	}
}

func (l *LoggerAdapter) Warn(ctx LogContext) {
	l.logger.Warn(ctx.LogData)
}

func (l *LoggerAdapter) Level() LogLevel {
	return l.logger.Level()
}

func (l *LoggerAdapter) SetLevel(lv LogLevel) {
	l.logger.SetLevel(lv)
}

func (l *LoggerAdapter) ShowSQL(show ...bool) {
	l.logger.ShowSQL(show...)
}

func (l *LoggerAdapter) IsShowSQL() bool {
	return l.logger.IsShowSQL()
}
