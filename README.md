# gouser
golang用户管理库  

## 特性
1. 游客注册、用户名+密码注册、邮箱+验证码注册、手机号+验证注册、第三方注册
2. 游客直接登录、用户名+密码直接登录、手机号+验证直接登录、第三方直接登录
3. 按uid、邮箱、手机号、第三方uid查找用户
4. 用户管理
5. 多端登录
6. 登录凭证（token）的验证
7. 访问秘钥（accesskey）的验证和管理
8. 验证码的生成和验证
9. 多种模块的自定义

## 安装
```bash
go get github.com/cheetah-fun-gs/gouser
```

## 使用说明
### 创建用户管理对象
```golang
import (
    "github.com/cheetah-fun-gs/gouser"
	"github.com/cheetah-fun-gs/gouser/usermgr"
)

func main() {
    mgr := gouser.New(name, secret, pool, db)
    mgr := gouser.New(name, secret, pool, db, usermgr.Config{})
    // 设定支持的第三方验证方式
    mgr.SetAuthMgr(...)
}
```

### 注册
```golang
// 用户名+密码注册
mgr.RegisterLAPD(uid, rawPassword string) (*User, error)
// 邮件+验证码注册
mgr.RegisterEmailApplyCode(email string) (code string, expire int, err error) // 生成验证码 需要自己发邮件
mgr.RegisterEmail(email, code string) (*User, error) // 注册用户
// 手机+验证码注册
mgr.RegisterMobileApplyCode(mobile string) (code string, expire, retry int, err error) // 生成验证码 需要自己发短信
mgr.RegisterMobile(mobile, code string) (*User, error) // 注册用户
// 游客注册
mgr.RegisterTourist() (*User, error)
// 第三方注册
mgr.RegisterAuth(authName string, v interface{}) (*User, error)
```

### 登录（不存在则自动注册）
```golang
// 游客登录
mgr.LoginTourist() (user *User, token string, deadline int64, err error) 
mgr.LoginTouristWithFrom(from string) (user *User, token string, deadline int64, err error) // 带来源

// 密码登录
mgr.LoginLAPD(uid, rawPassword string) (user *User, token string, deadline int64, err error)
mgr.LoginLAPDWithFrom(uid, rawPassword, from string) (user *User, token string, deadline int64, err error) // 带来源

// 手机验证码登录
mgr.LoginMobileApplyCode(mobile string) (code string, expire, retry int, err error) // 申请验证码 需要自己发短信
mgr.LoginMobile(mobile, code string) (user *User, token string, deadline int64, err error) 
mgr.LoginMobileWithFrom(mobile, code, from string) (user *User, token string, deadline int64, err error)  // 带来源

// LoginAuth 第三方登录
mgr.LoginAuth(authName string, v interface{}) (user *User, token string, deadline int64, err error) 
mgr.LoginAuthWithFrom(authName string, v interface{}, from string) (user *User, token string, deadline int64, err error) // 带来源
```

### 查找用户
```golang
// FindUserByAny 根据用户名/邮箱/手机号 查找用户
mgr.FindUserByAny(any string) (bool, *User, error) 
// FindUserByUID 根据用户名 查找用户
mgr.FindUserByUID(uid string) (bool, *User, error)
// FindUserByEmail 根据邮箱 查找用户
mgr.FindUserByEmail(email string) (bool, *User, error) 
// FindUserByMobile 根据手机号 查找用户
mgr.FindUserByMobile(mobile string) (bool, *User, error) 
// FindUserByAuth 根据第三方认证 查找用户
mgr.FindUserByAuth(authName, authUID string) (bool, *User, error) 
```

### 校验token
```golang
mgr.VerifyToken(uid, token string) (ok bool, err error)
mgr.VerifyTokenWithFrom(uid, from, token string) (ok bool, err error) // 带来源
```

### 校验sign
```golang
// 验证sign: sign由access key和请求数据(或请求数据部分字段)计算得到，前后端保持一致
mgr.VerifySign(uid string, accessKeyID int, data interface{}, sign string) (ok bool, err error) 
```

### 定制
```golang
// 设置第三方认证
mgrSetAuthMgr(args ...authmgr.AuthMgr)
// 设置token管理器
mgr.SetTokenMgr(arg tokenmgr.TokenMgr) 
// 设置生成uid的方法 uid格式改变可能需要修改sql表结构
mgr.SetGenerateUID(arg func() (uid, nickname, avatar, extra string)) 
// 设置生成验证码的方法
mgr.SetGenerateCode(arg func() string)
// 设置生成accessKey的方法
mgr.SetGenerateAccessKey(arg func() string) 
// 设置根据accessKey的方法
mgr.SetGenerateSign(arg func(accessKey string, data interface{}) string) 
// 设置用户表的表名和表结构
mgr.SetTableUser(tableName, tableCreateSQL string) error 
// 设置第三方认证表的表名和表结构
mgr.SetTableAuth(tableName, tableCreateSQL string) error 
// 设置accessKey表的表名和表结构
mgr.SetTableAccessKey(tableName, tableCreateSQL string) error
```

### 示例
```golang
package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/cheetah-fun-gs/gouser"
	"github.com/cheetah-fun-gs/gouser/usermgr"
	_ "github.com/go-sql-driver/mysql"
	redigo "github.com/gomodule/redigo/redis"
)

func defaultGenerateSign(accessKey string, data interface{}) string {
	ts := data.(int64)
	h := md5.New()
	h.Write([]byte(accessKey))
	h.Write([]byte(strconv.Itoa(int(ts))))
	return hex.EncodeToString(h.Sum(nil))
}

const (
	testAuthName = "testAuth"
)

type testAuth struct{}

func (auth *testAuth) GetName() string {
	return testAuthName
}
func (auth *testAuth) Verify(v interface{}) (uid, extra string, err error) {
	uid = v.(string) + "_testAuth"
	return
}

func dial() (redigo.Conn, error) {
	return redigo.DialTimeout("tcp", "127.0.0.1:6379", 2*time.Second, 2*time.Second, 2*time.Second)
}

func main() {
	pool := &redigo.Pool{
		Dial: dial,
	}
	defer pool.Close()

	db, err := sql.Open("mysql", "admin:admin123@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&loc=Asia%2FShanghai")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	name := "demo"
	secret := "tZli3W^4Rb#V"
	mgr := gouser.New(name, secret, pool, db, usermgr.Config{IsEnableAccessKey: true})
	// 设置认证
	mgr.SetAuthMgr(&testAuth{})

	if err := mgr.EnsureTables(); err != nil {
		panic(err)
	}

	for _, tableName := range mgr.TableNames() {
		if _, err = db.Exec(fmt.Sprintf("truncate table %s;", tableName)); err != nil {
			panic(err)
		}
	}

	redisConn := pool.Get()
	redisConn.Do("flushdb")
	redisConn.Close()

	// 游客注册
	user, err := mgr.RegisterTourist()
	if err != nil {
		panic(err)
	}

	token, _, err := user.Login()
	if err != nil {
		panic(err)
	}

	ok, err := mgr.VerifyToken(user.UID, token)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("token Verify fail")
	}

	// 修改uid
	testuid := "test_uid"
	if err = user.UpdateUID(testuid); err != nil {
		panic(err)
	}

	ok, user, err = mgr.FindUserByUID(testuid)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("testuid not found")
	}

	// 修改email
	time.Sleep(200 * time.Millisecond)
	testemail := "test123@123.com"
	emailcode, _, err := user.UpdateEmailApplyCode()
	if err != nil {
		panic(err)
	}
	if err = user.UpdateEmail(testemail, emailcode); err != nil {
		panic(err)
	}
	ok, _, err = mgr.FindUserByEmail(testemail)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("testemail not found")
	}

	// 修改mobile
	time.Sleep(200 * time.Millisecond)
	testmobile := "13000000000"
	mobilecode, _, _, err := user.UpdateMobileApplyCode(testmobile)
	if err != nil {
		panic(err)
	}
	if err = user.UpdateMobile(testmobile, mobilecode); err != nil {
		panic(err)
	}
	ok, _, err = mgr.FindUserByMobile(testmobile)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("testemail not found")
	}

	// 游客登录
	user, _, _, err = mgr.LoginTourist()
	if err != nil {
		panic(err)
	}

	// accesskey
	accessKey, err := user.GenerateAccessKey("test")
	if err != nil {
		panic(err)
	}

	ts := time.Now().Unix()
	sign := defaultGenerateSign(accessKey.AccessKey, ts)
	ok, err = mgr.VerifySign(user.UID, accessKey.ID, ts, sign)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("access key Verify error")
	}

	// 绑定第三方
	if err = user.BindAuth(testAuthName, user.UID); err != nil {
		panic(err)
	}

	// 第三方登录
	authcode := "testabc"
	if _, _, _, err = mgr.LoginAuth(testAuthName, authcode); err != nil {
		panic(err)
	}

	ok, _, err = mgr.FindUserByAuth(testAuthName, authcode+"_testAuth")
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("auth user not found")
	}

	// lapd注册
	lapdUID := "test_lapd"
	lapdPass := "test_lapd"
	user, err = mgr.RegisterLAPD(lapdUID, lapdPass)
	if err != nil {
		panic(err)
	}
	token, _, err = user.Login()
	if err != nil {
		panic(err)
	}
	ok, err = mgr.VerifyToken(user.UID, token)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("token Verify fail")
	}

	ok, _, err = mgr.FindUserByAny(lapdUID)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("lapdUID not found")
	}

	// 修改密码
	time.Sleep(200 * time.Millisecond)
	testpassword := "test_uid123"
	passwardcode, _, err := user.UpdatePasswordApplyCode()
	if err != nil {
		panic(err)
	}
	if err = user.UpdatePasswordWithCode(testpassword, passwardcode); err != nil {
		panic(err)
	}
	if _, _, _, err = mgr.LoginLAPD(testuid, testpassword); err != nil {
		panic(err)
	}

	// 邮箱注册
	email := "test_email@abc.com"
	emailcode, _, err = mgr.RegisterEmailApplyCode(email)
	if err != nil {
		panic(err)
	}

	user, err = mgr.RegisterEmail(email, emailcode)
	if err != nil {
		panic(err)
	}
	token, _, err = user.Login()
	if err != nil {
		panic(err)
	}
	ok, err = mgr.VerifyToken(user.UID, token)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("token Verify fail")
	}

	ok, _, err = mgr.FindUserByEmail(email)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("email not found")
	}

	// 手机注册
	mobile := "13000000001"
	mobilecode, _, _, err = mgr.RegisterMobileApplyCode(mobile)
	if err != nil {
		panic(err)
	}

	user, err = mgr.RegisterMobile(mobile, mobilecode)
	if err != nil {
		panic(err)
	}
	token, _, err = user.Login()
	if err != nil {
		panic(err)
	}
	ok, err = mgr.VerifyToken(user.UID, token)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("token Verify fail")
	}

	ok, _, err = mgr.FindUserByMobile(mobile)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("mobile not found")
	}

	// 手机直接登录
	mobile = "13000000002"
	mobilecode, _, _, err = mgr.LoginMobileApplyCode(mobile)
	if err != nil {
		panic(err)
	}
	user, _, _, err = mgr.LoginMobile(mobile, mobilecode)
	if err != nil {
		panic(err)
	}
}
```
