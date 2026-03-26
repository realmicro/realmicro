package models

import "xorm.io/xorm"

type Options struct {
	BdPath    string
	Password  string
	IfShowSql bool
	IfSyncDB  bool
	AfterInit func(x *xorm.Engine)
}

type Option func(o *Options)

func WithPassword(pass string) Option {
	return func(o *Options) {
		o.Password = pass
	}
}

func WithBdPath(path string) Option {
	return func(o *Options) {
		o.BdPath = path
	}
}

func IfShowSql(b bool) Option {
	return func(o *Options) {
		o.IfShowSql = b
	}
}

func IfSyncDB(b bool) Option {
	return func(o *Options) {
		o.IfSyncDB = b
	}
}

func AfterInit(after func(x *xorm.Engine)) Option {
	return func(o *Options) {
		o.AfterInit = after
	}
}
