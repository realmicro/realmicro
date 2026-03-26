package models

import (
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/realmicro/realmicro/logger"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

var x *xorm.Engine

type OrmFilter func(session *xorm.Session) *xorm.Session
type Handler func(session *xorm.Session) error

func GetDB() *xorm.Engine {
	return x
}

type Model struct {
	Id        int64     `xorm:"pk autoincr" mapstructure:"id"`
	CreatedAt time.Time `xorm:"created"`
	UpdatedAt time.Time `xorm:"updated"`
}

func Init(opts ...Option) {
	var ops Options
	for _, o := range opts {
		o(&ops)
	}

	var err error
	x, err = xorm.NewEngine("sqlite3", ops.BdPath)
	if err != nil {
		logger.Fatal(err)
	}
	x.SetMapper(names.GonicMapper{})
	x.TZLocation, _ = time.LoadLocation("Asia/Shanghai")
	// If you need show raw sql in log
	if ops.IfShowSql {
		x.ShowSQL(true)
	}

	x.SetMaxIdleConns(25)
	x.SetMaxOpenConns(100)
	x.SetConnMaxLifetime(time.Hour)

	if err = x.Ping(); err != nil {
		logger.Fatal(err)
	}

	if ops.AfterInit != nil {
		ops.AfterInit(x)
	}
}

func Transaction(handlers []Handler) (err error) {
	session := x.NewSession()
	if err = session.Begin(); err != nil {
		return
	}
	defer func() {
		if err != nil {
			session.Rollback()
		}
		session.Close()
	}()

	for i := 0; i < len(handlers); i++ {
		if err = handlers[i](session); err != nil {
			return
		}
	}

	return session.Commit()
}
