package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cheetah-fun-gs/gouser"
	_ "github.com/go-sql-driver/mysql"
	redigo "github.com/gomodule/redigo/redis"
)

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
	usermgr := gouser.New(name, secret, pool, db)
	// 设置认证
	usermgr.SetAuthMgr(&testAuth{})

	if err := usermgr.EnsureTables(); err != nil {
		panic(err)
	}

	for _, tableName := range usermgr.TableNames() {
		if _, err = db.Exec(fmt.Sprintf("truncate table %s;", tableName)); err != nil {
			panic(err)
		}
	}

	redisConn := pool.Get()
	redisConn.Do("flushdb")
	redisConn.Close()

	// 游客注册
	user, err := usermgr.RegisterTourist()
	if err != nil {
		panic(err)
	}

	token, _, err := user.Login()
	if err != nil {
		panic(err)
	}

	ok, err := usermgr.VerifyToken(user.UID, token)
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

	ok, user, err = usermgr.FindUserByUID(testuid)
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
	ok, _, err = usermgr.FindUserByEmail(testemail)
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
	ok, _, err = usermgr.FindUserByMobile(testmobile)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("testemail not found")
	}

	// 游客登录
	user, _, _, err = usermgr.LoginTourist()
	if err != nil {
		panic(err)
	}

	// 绑定第三方
	if err = user.BindAuth(testAuthName, user.UID); err != nil {
		panic(err)
	}

	// 第三方登录
	authcode := "testabc"
	if _, _, _, err = usermgr.LoginAuth(testAuthName, authcode); err != nil {
		panic(err)
	}

	ok, _, err = usermgr.FindUserByAuth(testAuthName, authcode+"_testAuth")
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("auth user not found")
	}

	// lapd注册
	lapdUID := "test_lapd"
	lapdPass := "test_lapd"
	user, err = usermgr.RegisterLAPD(lapdUID, lapdPass)
	if err != nil {
		panic(err)
	}
	token, _, err = user.Login()
	if err != nil {
		panic(err)
	}
	ok, err = usermgr.VerifyToken(user.UID, token)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("token Verify fail")
	}

	ok, _, err = usermgr.FindUserByAny(lapdUID)
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
	if _, _, _, err = usermgr.LoginLAPD(testuid, testpassword); err != nil {
		panic(err)
	}

	// 邮箱注册
	email := "test_email@abc.com"
	emailcode, _, err = usermgr.RegisterEmailApplyCode(email)
	if err != nil {
		panic(err)
	}

	user, err = usermgr.RegisterEmail(email, emailcode)
	if err != nil {
		panic(err)
	}
	token, _, err = user.Login()
	if err != nil {
		panic(err)
	}
	ok, err = usermgr.VerifyToken(user.UID, token)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("token Verify fail")
	}

	ok, _, err = usermgr.FindUserByEmail(email)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("email not found")
	}

	// 手机注册
	mobile := "13000000001"
	mobilecode, _, _, err = usermgr.RegisterMobileApplyCode(mobile)
	if err != nil {
		panic(err)
	}

	user, err = usermgr.RegisterMobile(mobile, mobilecode)
	if err != nil {
		panic(err)
	}
	token, _, err = user.Login()
	if err != nil {
		panic(err)
	}
	ok, err = usermgr.VerifyToken(user.UID, token)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("token Verify fail")
	}

	ok, _, err = usermgr.FindUserByMobile(mobile)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("mobile not found")
	}

	// 手机直接登录
	mobile = "13000000002"
	mobilecode, _, _, err = usermgr.LoginMobileApplyCode(mobile)
	if err != nil {
		panic(err)
	}
	user, _, _, err = usermgr.LoginMobile(mobile, mobilecode)
	if err != nil {
		panic(err)
	}
}