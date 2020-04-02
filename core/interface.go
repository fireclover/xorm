package core

import (
	"context"
	"database/sql"

	"xorm.io/xorm/log"
)

type Hook interface {
	BeforeProcess(ctx context.Context, query string, args ...interface{}) (context.Context, error)
	AfterProcess(logContext *log.LogContext) error
}

// Queryer represents an interface to query a SQL to get data from database
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*Rows, error)
}

// Executer represents an interface to execute a SQL
type Executer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// QueryExecuter combines the Queryer and Executer
type QueryExecuter interface {
	Queryer
	Executer
}
