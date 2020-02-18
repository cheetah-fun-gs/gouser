package gouser

import (
	"database/sql"

	"github.com/cheetah-fun-gs/gouser/usermgr"
	redigo "github.com/gomodule/redigo/redis"
)

// New ...
func New(name, secret string, pool *redigo.Pool, db *sql.DB, configs ...usermgr.Config) *usermgr.UserMgr {
	return usermgr.New(name, secret, pool, db, configs...)
}
