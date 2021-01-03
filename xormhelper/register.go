package xormhelper

import (
	"errors"
	"gitea.com/laixyz/xorm"
	"gitea.com/laixyz/xorm/log"
	"gitea.com/laixyz/xorm/names"
	_ "github.com/go-sql-driver/mysql"
	"sync"
	"time"
)

type XormSession struct {
	Engine  *xorm.Engine
	Session *xorm.Session
	Name    string
	DSN     string
	Prefix  string
	Debug   bool
}

func (xc *XormSession) Connect() (err error) {
	xc.Engine, err = xorm.NewEngine("mysql", xc.DSN)
	if err != nil {
		return err
	}
	if xc.Prefix != "" {
		tbMapper := names.NewPrefixMapper(names.GonicMapper{}, xc.Prefix)
		xc.Engine.SetTableMapper(tbMapper)
	}
	xc.Engine.SetConnMaxLifetime(25 * time.Second)
	xc.Engine.SetMaxIdleConns(64)
	xc.Engine.SetMaxOpenConns(16)
	if xc.Debug {
		xc.Engine.ShowSQL(true)
		//xorm 日志设置，只显示错误日志
		xc.Engine.Logger().SetLevel(log.LOG_DEBUG)
	} else {
		xc.Engine.Logger().SetLevel(log.LOG_ERR)
	}
	xc.Session = xc.Engine.NewSession()
	return nil
}
func (xc *XormSession) Close() error {
	return xc.Engine.Close()
}

type Xorms struct {
	Engine map[string]*XormSession
	sync.RWMutex
}

var PublicXorms Xorms = Xorms{Engine: make(map[string]*XormSession)}

/*
Register 注册一个mysql连接池
范例:
	Register("test:test@(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local")
	给连接取个名称
	Register("default","test:test@(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local")
	指定表前缀
	Register("default","test:test@(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local","prefix_")
	设置调试模式，会打印sql语句
	Register("default","test:test@(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local","prefix_","true")
*/
func MySQLRegister(params ...string) error {
	var newXormConfig XormSession
	paramsLen := len(params)
	if paramsLen >= 4 {
		if params[3] == "true" {
			newXormConfig.Debug = true
		}
	}
	if paramsLen >= 3 {
		newXormConfig.Name = params[0]
		newXormConfig.DSN = params[1]
		newXormConfig.Prefix = params[2]
	} else if paramsLen == 2 {
		newXormConfig.Name = params[0]
		newXormConfig.DSN = params[1]
	} else if paramsLen == 1 {
		newXormConfig.Name = "default"
		newXormConfig.DSN = params[0]
	} else {
		return errors.New("the XORM Register function param failure")
	}
	err := newXormConfig.Connect()
	if err != nil {
		return err
	}
	PublicXorms.Lock()
	defer PublicXorms.Unlock()
	if _, ok := PublicXorms.Engine[newXormConfig.Name]; ok {
		delete(PublicXorms.Engine, newXormConfig.Name)
	}
	PublicXorms.Engine[newXormConfig.Name] = &newXormConfig
	return nil
}

// Using 使用指定的名字的连接池， 不指定默认为default
func Using(params ...string) (*xorm.Session, *xorm.Engine, error) {
	PublicXorms.Lock()
	defer PublicXorms.Unlock()
	var name string
	if len(params) == 0 {
		name = "default"
	}
	if _, ok := PublicXorms.Engine[name]; ok {
		if err := PublicXorms.Engine[name].Engine.Ping(); err != nil {
			err = PublicXorms.Engine[name].Connect()
			if err != nil {
				return nil, nil, err
			}
			return PublicXorms.Engine[name].Session, PublicXorms.Engine[name].Engine, err
		}
		return PublicXorms.Engine[name].Session, PublicXorms.Engine[name].Engine, nil
	}
	return nil, nil, errors.New("the mysql connect not register")
}

/* Close 关闭连接
 范例：
	Close() 关闭名字为default的连接
	Close("my") 关闭名字为my的连接
	Close("my","your") 关闭多个连接
	Close("__all__") 关闭所有的连接。
*/
func Close(params ...string) (err error) {
	PublicXorms.Lock()
	defer PublicXorms.Unlock()
	var names []string
	if len(params) == 0 {
		names = []string{"default"}
	} else if len(params) == 1 && params[0] == "__all__" {
		for _, xc := range PublicXorms.Engine {
			err = xc.Close()
			if err != nil {
				return
			}
		}
		return nil
	} else {
		names = params
	}
	for _, name := range names {
		if _, ok := PublicXorms.Engine[name]; ok {
			err = PublicXorms.Engine[name].Close()
			if err != nil {
				return
			}
		}
	}
	return nil
}
