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

gouser.New(name, secret, pool, db)
gouser.New(name, secret, pool, db, usermgr.Config{})
```

### 注册用户
```golang
func (mgr *UserMgr) RegisterAuth(authName string, v interface{}) (*User, error)
    RegisterAuth 第三方认证注册

func (mgr *UserMgr) RegisterEmail(email, code string) (*User, error)
    RegisterEmail 邮件用户注册

func (mgr *UserMgr) RegisterEmailApplyCode(email string) (code string, expire int, err error)
    RegisterEmailApplyCode 邮件用户注册申请code

func (mgr *UserMgr) RegisterLAPD(uid, rawPassword string) (*User, error)
    RegisterLAPD 密码用户注册

func (mgr *UserMgr) RegisterMobile(mobile, code string) (*User, error)
    RegisterMobile 手机用户注册

func (mgr *UserMgr) RegisterMobileApplyCode(mobile string) (code string, expire, retry int, err error)
    RegisterMobileApplyCode 手机用户注册申请code

func (mgr *UserMgr) RegisterTourist() (*User, error)
    RegisterTourist 游客注册

```

### 快速登录（不存在自动注册）
```golang
func (mgr *UserMgr) LoginAuth(authName string, v interface{}) (user *User, token string, deadline int64, err error)
    LoginAuth 第三方登录

func (mgr *UserMgr) LoginAuthWithFrom(authName string, v interface{}, from string) (user *User, token string, deadline int64, err error)
    LoginAuthWithFrom 第三方登录 带来源

func (mgr *UserMgr) LoginLAPD(uid, rawPassword string) (user *User, token string, deadline int64, err error)
    LoginLAPD 密码登录

func (mgr *UserMgr) LoginLAPDWithFrom(uid, rawPassword, from string) (user *User, token string, deadline int64, err error)
    LoginLAPDWithFrom 密码登录 带来源

func (mgr *UserMgr) LoginMobile(mobile, code string) (user *User, token string, deadline int64, err error)
    LoginMobile 手机验证码登录

func (mgr *UserMgr) LoginMobileApplyCode(mobile string) (code string, expire, retry int, err error)
    LoginMobileApplyCode 手机验证码登录 申请验证码

func (mgr *UserMgr) LoginMobileWithFrom(mobile, code, from string) (user *User, token string, deadline int64, err error)
    LoginMobileWithFrom 手机验证码登录 带来源

func (mgr *UserMgr) LoginTourist() (user *User, token string, deadline int64, err error)
    LoginTourist 游客登录

func (mgr *UserMgr) LoginTouristWithFrom(from string) (user *User, token string, deadline int64, err error)
    LoginTouristWithFrom 游客登录 带来源
```

### 查找用户
```golang
func (mgr *UserMgr) FindUserByAny(any string) (bool, *User, error)
    FindUserByAny 根据用户名/邮箱/手机号 查找用户

func (mgr *UserMgr) FindUserByAuth(authName, authUID string) (bool, *User, error)
    FindUserByAuth 根据第三方认证 查找用户

func (mgr *UserMgr) FindUserByEmail(email string) (bool, *User, error)
    FindUserByEmail 根据邮箱 查找用户

func (mgr *UserMgr) FindUserByMobile(mobile string) (bool, *User, error)
    FindUserByMobile 根据手机号 查找用户

func (mgr *UserMgr) FindUserByUID(uid string) (bool, *User, error)
    FindUserByUID 根据用户名 查找用户
```

### 校验token
```golang
func (mgr *UserMgr) VerifyToken(uid, token string) (ok bool, err error)
    VerifyToken 验证token

func (mgr *UserMgr) VerifyTokenWithFrom(uid, from, token string) (ok bool, err error)
    VerifyTokenWithFrom 验证token 带来源
```

### 校验sign
```golang
func (mgr *UserMgr) VerifySign(uid string, accessKeyID int, data interface{}, sign string) (ok bool, err error)
    VerifySign 验证sign: sign由access key和请求数据(或请求数据部分字段)计算得到 
```

### 校验第三方认证
```
func (mgr *UserMgr) VerifyAuth(authName string, v interface{}) (authUID, authExtra string, err error)
    VerifyAuth 验证第三方凭证
```

### 定制
```golang
func (mgr *UserMgr) SetAuthMgr(args ...authmgr.AuthMgr)
    SetAuthMgr 设置第三方认证

func (mgr *UserMgr) SetGenerateAccessKey(arg func() string)
    SetGenerateAccessKey 设置生成accesskey的方法

func (mgr *UserMgr) SetGenerateCode(arg func() string)
    SetGenerateCode 设置生成验证码的方法

func (mgr *UserMgr) SetGenerateSign(arg func(accessKey string, data interface{}) string)
    SetGenerateSign 设置根据accesskey计算sign的方法

func (mgr *UserMgr) SetGenerateUID(arg func() (uid, nickname, avatar, extra string))
    SetGenerateUID 设置生成用户信息的方法 如果uid格式改变，可能需要修改sql表结构

func (mgr *UserMgr) SetMLogName(name string)
    SetMLogName 设置日志

func (mgr *UserMgr) SetTableAccessKey(tableName, tableCreateSQL string) error
    SetTableAccessKey 设置accessKey表表名和表结构

func (mgr *UserMgr) SetTableAuth(tableName, tableCreateSQL string) error
    SetTableAuth 设置第三方验证表表名和表结构

func (mgr *UserMgr) SetTableUser(tableName, tableCreateSQL string) error
    SetTableUser 设置用户表表名和表结构

func (mgr *UserMgr) SetTokenMgr(arg tokenmgr.TokenMgr)
    SetTokenMgr 设置token管理器
```

### sql表
```golang
func (mgr *UserMgr) EnsureTables() error
    EnsureTables 确保sql表已建立

func (mgr *UserMgr) TableNames() []string
    TableNames 获得表名

func (mgr *UserMgr) TablesCreateSQL() []string
    TablesCreateSQL 获得建表语句
```

### 验证码
```golang
func (mgr *UserMgr) ApplyCode(expire int, args ...interface{}) (code string, expire0 int, err error)
    ApplyCode 申请一个验证码, args用来区分场景

func (mgr *UserMgr) ApplyCodeAntiReplay(lockname string, expire, retry int, args ...interface{}) (code string, expire0, retry0 int, err error)
    ApplyCodeAntiReplay 申请一个防重放验证码, args用来区分场景

func (mgr *UserMgr) VerifyCode(code string, args ...interface{}) (bool, error)
    VerifyCode 申请验证码 args和ApplyCode时保持一致
```

### 用户
```golang
type User struct {
        *UserData
        // Has unexported fields.
}
    User 用户

func (user *User) BindAuth(authName string, v interface{}) error
    BindAuth 绑定第三方认证

func (user *User) Clean() error
    Clean 清除用户

func (user *User) DeleteAccessKey(accessKeyID int) error
    DeleteAccessKey 删除一个 access key

func (user *User) GenerateAccessKey(comment string, expireAts ...time.Time) (*UserAccessKey, error)
    GenerateAccessKey 生成一个 access key

func (user *User) GetAccessKeys(isAll bool) ([]*UserAccessKey, error)
    GetAccessKeys 获取accesskeys isAll 是否包含过期的访问秘钥

func (user *User) GetAuths() ([]*UserAuth, error)
    GetAuths 获得第三方认证信息

func (user *User) Login() (token string, deadline int64, err error)
    Login 登录

func (user *User) LoginWithFrom(from string) (token string, deadline int64, err error)
    LoginWithFrom 登录 带来源

func (user *User) Logout() error
    Logout 登出

func (user *User) LogoutWithFrom(from string) error
    LogoutWithFrom 登出 带来源

func (user *User) UnbindAuth(authName string) error
    UnbindAuth 解绑第三方认证

func (user *User) UpdateAccessKeyComment(accessKeyID int, comment string) error
    UpdateAccessKeyComment 更新一个 access key 的 comment

func (user *User) UpdateAccessKeyExpireAt(accessKeyID int, expireAt *time.Time) error
    UpdateAccessKeyExpireAt 更新一个 access key的超时设置 expireAt为空表示永久有效

func (user *User) UpdateAuthInfo(authName, authExtra string) error
    UpdateAuthInfo 更新第三方认证信息

func (user *User) UpdateEmail(email, code string) error
    UpdateEmail 更新邮箱

func (user *User) UpdateEmailApplyCode() (code string, expire int, err error)
    UpdateEmailApplyCode 更新邮箱申请验证码

func (user *User) UpdateInfo(nickname, avatar, extra *string) error
    UpdateInfo 更新用户信息

func (user *User) UpdateMobile(mobile, code string) error
    UpdateMobile 更新手机号

func (user *User) UpdateMobileApplyCode(mobile string) (code string, expire, retry int, err error)
    UpdateMobileApplyCode 更新手机号申请验证码

func (user *User) UpdatePasswordApplyCode() (code string, expire int, err error)
    UpdatePasswordApplyCode 更改密码申请验证码

func (user *User) UpdatePasswordWithCode(rawPassword, code string) error
    UpdatePasswordWithCode 通过验证码更改密码

func (user *User) UpdatePasswordWithPassword(oldRawPassword, newRawPassword string) error
    UpdatePasswordWithPassword 通过旧密码更改密码

func (user *User) UpdateUID(uid string) error
    UpdateUID 更新uid
```

## 示例
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
