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
