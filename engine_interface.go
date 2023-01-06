//go:build !go1.15
// +build !go1.15

package xorm

import (
	"context"
	"database/sql"
	"reflect"
	"time"

	"xorm.io/xorm/caches"
	"xorm.io/xorm/contexts"
	"xorm.io/xorm/dialects"
	"xorm.io/xorm/log"
	"xorm.io/xorm/names"
	"xorm.io/xorm/schemas"
)

// EngineInterface defines the interface which Engine, EngineGroup will implementate.
type EngineInterface interface {
	Interface

	Before(func(interface{})) *Session
	Charset(charset string) *Session
	ClearCache(...interface{}) error
	Context(context.Context) *Session
	CreateTables(...interface{}) error
	DBMetas() ([]*schemas.Table, error)
	DBVersion() (*schemas.Version, error)
	Dialect() dialects.Dialect
	DriverName() string
	DropTables(...interface{}) error
	DumpAllToFile(fp string, tp ...schemas.DBType) error
	GetCacher(string) caches.Cacher
	GetColumnMapper() names.Mapper
	GetDefaultCacher() caches.Cacher
	GetTableMapper() names.Mapper
	GetTZDatabase() *time.Location
	GetTZLocation() *time.Location
	ImportFile(fp string) ([]sql.Result, error)
	MapCacher(interface{}, caches.Cacher) error
	NewSession() *Session
	NoAutoTime() *Session
	Prepare() *Session
	Quote(string) string
	SetCacher(string, caches.Cacher)
	SetConnMaxLifetime(time.Duration)
	SetColumnMapper(names.Mapper)
	SetTagIdentifier(string)
	SetDefaultCacher(caches.Cacher)
	SetLogger(logger interface{})
	SetLogLevel(log.LogLevel)
	SetMapper(names.Mapper)
	SetMaxOpenConns(int)
	SetMaxIdleConns(int)
	SetQuotePolicy(dialects.QuotePolicy)
	SetSchema(string)
	SetTableMapper(names.Mapper)
	SetTZDatabase(tz *time.Location)
	SetTZLocation(tz *time.Location)
	AddHook(hook contexts.Hook)
	ShowSQL(show ...bool)
	Sync(...interface{}) error
	Sync2(...interface{}) error
	StoreEngine(storeEngine string) *Session
	TableInfo(bean interface{}) (*schemas.Table, error)
	TableName(interface{}, ...bool) string
	UnMapType(reflect.Type)
	EnableSessionID(bool)
}

var (
	_ EngineInterface = &Engine{}
	_ EngineInterface = &EngineGroup{}
)
