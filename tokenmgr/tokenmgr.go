// Package tokenmgr 支持多端登录，生成新token后旧token还有5分钟有效期
package tokenmgr

import (
	"fmt"
	"strings"
	"time"

	"github.com/cheetah-fun-gs/goplus/locker"
	mlogger "github.com/cheetah-fun-gs/goplus/multier/multilogger"
	uuidplus "github.com/cheetah-fun-gs/goplus/uuid"
	redigo "github.com/gomodule/redigo/redis"
)

// TokenMgr Token管理器定义
type TokenMgr interface {
	Generate(uid, from string) (token string, deadline int64, err error) // 生成一个新的token
	Verify(uid, from, token string) (ok bool, err error)                 // 验证token是否有效
	Clean(uid, from string) error                                        // 清除token
	CleanAll(uid string) error                                           // 清除token
}

// DefaultMgr 默认管理器
// 数据结构 uid : map[from-token]create_time
type DefaultMgr struct {
	name          string // 管理器名称
	expire1       int    // 凭证的超时时间, 不宜太短应该比expire2长
	expire2       int    // 被刷新凭证的保留时间, 不宜太长, 可为0
	pool          *redigo.Pool
	generateToken func(uid, from string) string
	mlogname      string
}

// New 获得一个新的token管理器
// expires[0]: expire1 凭证的超时时间, 不宜太短应该比expire2长, 默认1小时
// expires[1]: expire2 被刷新凭证的保留时间, 不宜太长, 可为0, 默认10分钟
func New(name string, pool *redigo.Pool, expires ...int) *DefaultMgr {
	mgr := &DefaultMgr{
		name:          name,
		pool:          pool,
		expire1:       3600,
		expire2:       300,
		generateToken: defaultGenerateToken,
		mlogname:      "default",
	}
	if len(expires) >= 1 && expires[0] != 0 {
		mgr.expire1 = expires[0]
	}
	if len(expires) == 2 {
		mgr.expire2 = expires[1]
	}
	if mgr.expire1 <= mgr.expire2 {
		panic("expire1 is below expire2")
	}
	return mgr
}

// SetMLogName 设置日志名
func (s *DefaultMgr) SetMLogName(name string) {
	s.mlogname = name
}

// SetGenerateToken 设置生成token方法
func (s *DefaultMgr) SetGenerateToken(v func(uid, from string) string) {
	s.generateToken = v
}

func defaultGenerateToken(uid, from string) string {
	return uuidplus.NewV4().Base62()
}

func getTokenKey(name, uid string) string {
	return fmt.Sprintf("%s:%s:token", name, uid)
}

func getTokenField(from, token string) string {
	return fmt.Sprintf("%s|%s", from, token)
}

func isFromToken(field, from string) bool {
	return strings.HasPrefix(field, from+"|")
}

// Generate ...
func (s *DefaultMgr) Generate(uid, from string) (token string, deadline int64, err error) {
	conn := s.pool.Get()
	defer conn.Close()

	tokenKey := getTokenKey(s.name, uid)
	lockName := fmt.Sprintf("%s:%s:locker", tokenKey, from)

	// 加锁
	var l *locker.Locker
	l, err = locker.New(s.pool, lockName)
	if err != nil {
		return
	}
	defer l.Close()

	now := time.Now()
	token = s.generateToken(uid, from)
	deadline = now.Unix() + int64(s.expire1)

	var result map[string]int64
	result, err = redigo.Int64Map(conn.Do("HGETALL", tokenKey))
	if err != nil {
		return
	}

	var latestDeadline int64
	for filed, oldDeadline := range result {
		if !isFromToken(filed, from) {
			delete(result, filed) // 不是该from的数据忽略
			continue
		}
		if oldDeadline > latestDeadline {
			latestDeadline = oldDeadline
		}
	}

	commands := []string{}
	for filed, oldDeadline := range result {
		if oldDeadline < latestDeadline || oldDeadline < now.Unix() {
			if err = conn.Send("HDEL", tokenKey, filed); err != nil { // 废弃或过期的token全部删除
				return
			}
			commands = append(commands, fmt.Sprint("HDEL", tokenKey, filed))
		} else if oldDeadline > now.Unix()+int64(s.expire2) {
			if err = conn.Send("HSET", tokenKey, filed, now.Unix()+int64(s.expire2)); err != nil { // 未失效的token 5分钟后失效
				return
			}
			commands = append(commands, fmt.Sprint("HSET", tokenKey, filed, now.Unix()+int64(s.expire2)))
		}
	}

	// 设定token key的超时时间
	if err = conn.Send("EXPIREAT", tokenKey, deadline); err != nil {
		return
	}
	commands = append(commands, fmt.Sprint("EXPIREAT", tokenKey, deadline))

	// 设定token和deadline
	tokenField := getTokenField(from, token)
	if err = conn.Send("HSET", tokenKey, tokenField, deadline); err != nil {
		return
	}
	commands = append(commands, fmt.Sprint("HSET", tokenKey, tokenField, deadline))

	// 执行
	if err = conn.Flush(); err != nil {
		return
	}

	for i := 1; i < len(commands); i++ {
		if _, err = conn.Receive(); err != nil {
			mlogger.WarnN(s.mlogname, "Generate token err: %v, %v", commands[i], err)
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

	tokenKey := getTokenKey(s.name, uid)
	tokenField := getTokenField(from, token)

	var deadline int64
	deadline, err = redigo.Int64(conn.Do("HGET", tokenKey, tokenField))
	if err != nil && err != redigo.ErrNil {
		return
	}
	return deadline > time.Now().Unix(), nil
}

// Clean ...
func (s *DefaultMgr) Clean(uid, from string) error {
	conn := s.pool.Get()
	defer conn.Close()

	tokenKey := getTokenKey(s.name, uid)
	fields, err := redigo.Strings(conn.Do("HKEYS", tokenKey))
	if err != nil {
		return err
	}

	for _, field := range fields {
		if isFromToken(field, from) {
			conn.Send("HDEL", tokenKey, field)
		}
	}

	return conn.Flush()
}

// CleanAll ...
func (s *DefaultMgr) CleanAll(uid string) (err error) {
	conn := s.pool.Get()
	defer conn.Close()

	tokenKey := getTokenKey(s.name, uid)
	_, err = conn.Do("DEL", tokenKey)
	return
}
