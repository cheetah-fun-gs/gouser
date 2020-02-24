package usermgr

import (
	"fmt"
	"strings"

	"github.com/cheetah-fun-gs/goplus/locker"
	redigo "github.com/gomodule/redigo/redis"
)

func getCodeKey(name, code string, args ...interface{}) string {
	splits := []string{name, code}
	for _, arg := range args {
		splits = append(splits, fmt.Sprintf("%v", arg))
	}
	splits = append(splits, "code")
	return strings.Join(splits, ":")
}

func getCodeLockKey(name, lockname string) string {
	return fmt.Sprintf("%s:%s:code:lock", name, lockname)
}

// ApplyCode 申请一个验证码, args用来区分场景
func (mgr *UserMgr) ApplyCode(expire int, args ...interface{}) (code string, expire0 int, err error) {
	if expire == 0 {
		expire = mgr.config.CodeExpire
	}

	conn := mgr.pool.Get()
	defer conn.Close()

	code = mgr.generateCode()
	codeKey := getCodeKey(mgr.name, code, args...)

	var result string
	result, err = redigo.String(conn.Do("SET", codeKey, "1", "EX", expire, "NX"))
	if err != nil {
		return
	}

	if result != "OK" {
		err = fmt.Errorf("code duplicate")
		return
	}

	expire0 = expire
	return
}

// ApplyCodeAntiReplay 申请一个防重放验证码, args用来区分场景
func (mgr *UserMgr) ApplyCodeAntiReplay(lockname string, expire, retry int, args ...interface{}) (code string, expire0, retry0 int, err error) {
	if expire == 0 {
		expire = mgr.config.CodeExpire
	}
	if retry == 0 {
		retry = mgr.config.CodeRetry
	}

	conn := mgr.pool.Get()
	defer conn.Close()

	if err = locker.Lock(conn, getCodeLockKey(mgr.name, lockname), retry); err == locker.ErrorLocked {
		err = ErrorLocked
		return
	} else if err != nil {
		return
	}

	code = mgr.generateCode()
	codeKey := getCodeKey(mgr.name, code, args...)

	var result string
	result, err = redigo.String(conn.Do("SET", codeKey, "1", "EX", expire, "NX"))
	if err != nil {
		return
	}

	if result != "OK" {
		err = fmt.Errorf("code duplicate")
		return
	}

	expire0 = expire
	retry0 = retry
	return
}

// VerifyCode 申请验证码 args和ApplyCode时保持一致
func (mgr *UserMgr) VerifyCode(code string, args ...interface{}) (bool, error) {
	conn := mgr.pool.Get()
	defer conn.Close()

	ok, err := redigo.Int(conn.Do("EXISTS", getCodeKey(mgr.name, code, args...)))
	if err != nil {
		return false, err
	}
	return ok == 1, nil
}
