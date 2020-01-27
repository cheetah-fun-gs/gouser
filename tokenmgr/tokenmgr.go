// Package tokenmgr 支持多端登录，生成新token后旧token还有5分钟有效期
package tokenmgr

import (
	"fmt"
	"time"

	"github.com/cheetah-fun-gs/goplus/locker"
	mlogger "github.com/cheetah-fun-gs/goplus/multier/multilogger"
	uuidplus "github.com/cheetah-fun-gs/goplus/uuid"
	"github.com/cheetah-fun-gs/gouser"
	redigo "github.com/gomodule/redigo/redis"
)

// TokenMgr Token管理器定义
type TokenMgr interface {
	Generate(uid, from string) (token string, deadline int64, err error) // 生成一个新的token
	Verify(uid, from, token string) (ok bool, err error)                 // 验证token是否有效
	Clean(uid, from, token string) error                                 // 清除token
}

// New ...
func New(name string, pool *redigo.Pool, expire int) *DefaultMgr {
	return &DefaultMgr{
		name:   name,
		pool:   pool,
		expire: expire,
	}
}

// DefaultMgr 默认管理器
type DefaultMgr struct {
	name   string // 管理器名称
	expire int    // 凭证的有效时间 应该大于5分钟
	pool   *redigo.Pool
}

// map[token]create_time
func getTokenKey(name, uid, from string) string {
	return fmt.Sprintf("%s:%s:%s", name, uid, from)
}

// Generate ...
func (s *DefaultMgr) Generate(uid, from string) (token string, deadline int64, err error) {
	conn := s.pool.Get()
	defer conn.Close()

	tokenKey := getTokenKey(s.name, uid, from)
	lockName := tokenKey + ":locker"

	// 加锁
	var l *locker.Locker
	l, err = locker.New(s.pool, lockName)
	if err != nil {
		return
	}
	defer l.Close()

	now := time.Now()
	token = uuidplus.NewV4().Base62()
	deadline = now.Unix() + int64(s.expire)

	var result map[string]int64
	result, err = redigo.Int64Map(conn.Do("HGETALL", tokenKey))
	if err != nil {
		return
	}

	var latestDeadline int64
	for _, oldDeadline := range result {
		if oldDeadline > latestDeadline {
			latestDeadline = oldDeadline
		}
	}

	commandCount := 0
	for oldToken, oldDeadline := range result {
		if oldDeadline < latestDeadline || oldDeadline < now.Unix() {
			if err = conn.Send("HDEL", tokenKey, oldToken); err != nil { // 废弃或过期的token全部删除
				return
			}
			commandCount++
		} else if oldDeadline > now.Unix()+300 {
			if err = conn.Send("HSET", tokenKey, oldToken, now.Unix()+300); err != nil { // 未失效的token 5分钟后失效
				return
			}
			commandCount++
		}
	}

	// 设定token key的超时时间
	if err = conn.Send("EXPIREAT", tokenKey, deadline); err != nil {
		return
	}
	commandCount++

	// 设定token和deadline
	if err = conn.Send("HSET", tokenKey, token, deadline); err != nil {
		return
	}
	commandCount++

	// 执行
	if err = conn.Flush(); err != nil {
		return
	}

	for i := 1; i < commandCount; i++ {
		if _, err = conn.Receive(); err != nil {
			mlogger.WarnN(gouser.MLoggerName, "Generate token err: command, %v/%v; err %v", i, commandCount, err)
		}
	}

	if _, err = conn.Receive(); err != nil {
		return
	}

	return
}

// Verify ...
func (s *DefaultMgr) Verify(uid, from, token string) (ok bool, err error) {
	conn := s.pool.Get()
	defer conn.Close()

	tokenKey := getTokenKey(s.name, uid, from)

	var deadline int64
	deadline, err = redigo.Int64(conn.Do("HGET", tokenKey, token))
	if err != nil && err != redigo.ErrNil {
		return
	}

	return deadline > time.Now().Unix(), nil
}

// Clean ...
func (s *DefaultMgr) Clean(uid, from, token string) (err error) {
	conn := s.pool.Get()
	defer conn.Close()

	tokenKey := getTokenKey(s.name, uid, from)
	_, err = conn.Do("HDEL", tokenKey, token)
	return
}
