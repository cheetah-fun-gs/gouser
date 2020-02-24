package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cheetah-fun-gs/gouser"
	_ "github.com/go-sql-driver/mysql"
	redigo "github.com/gomodule/redigo/redis"
)

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
	if err := usermgr.EnsureTables(); err != nil {
		panic(err)
	}

	for _, tableName := range usermgr.TableNames() {
		if _, err = db.Exec(fmt.Sprintf("truncate table %s;", tableName)); err != nil {
			panic(err)
		}
	}

	// 游客注册
	user1, err := usermgr.RegisterTourist()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", user1.UserData)
	token1, _, err := user1.Login()
	if err != nil {
		panic(err)
	}
	ok1, err := usermgr.VerifyToken(user1.UID, token1)
	if err != nil {
		panic(err)
	}
	if !ok1 {
		panic("token1 Verify fail")
	}

	ok1, _, err = usermgr.FindUserByUID(user1.UID)
	if err != nil {
		panic(err)
	}
	if !ok1 {
		panic("uid not found")
	}
	// 游客登录
	_, _, _, err = usermgr.LoginTourist()
	if err != nil {
		panic(err)
	}

	// lapd注册
	lapdUID := "test_lapd"
	lapdPass := "test_lapd"
	user2, err := usermgr.RegisterLAPD(lapdUID, lapdPass)
	if err != nil {
		panic(err)
	}
	token2, _, err := user2.Login()
	if err != nil {
		panic(err)
	}
	ok2, err := usermgr.VerifyToken(user2.UID, token2)
	if err != nil {
		panic(err)
	}
	if !ok2 {
		panic("token2 Verify fail")
	}

	ok2, _, err = usermgr.FindUserByAny(lapdUID)
	if err != nil {
		panic(err)
	}
	if !ok2 {
		panic("lapdUID not found")
	}

	// 邮箱注册
	email := "test_email@abc.com"
	emailcode, _, err := usermgr.RegisterEmailApplyCode(email)
	if err != nil {
		panic(err)
	}

	user3, err := usermgr.RegisterEmail(email, emailcode)
	if err != nil {
		panic(err)
	}
	token3, _, err := user3.Login()
	if err != nil {
		panic(err)
	}
	ok3, err := usermgr.VerifyToken(user3.UID, token3)
	if err != nil {
		panic(err)
	}
	if !ok3 {
		panic("token3 Verify fail")
	}

	ok3, _, err = usermgr.FindUserByEmail(email)
	if err != nil {
		panic(err)
	}
	if !ok3 {
		panic("email not found")
	}

	// 手机注册
	mobile := "13000000000"
	mobilecode, _, _, err := usermgr.RegisterMobileApplyCode(mobile)
	if err != nil {
		panic(err)
	}

	user4, err := usermgr.RegisterMobile(mobile, mobilecode)
	if err != nil {
		panic(err)
	}
	token4, _, err := user4.Login()
	if err != nil {
		panic(err)
	}
	ok4, err := usermgr.VerifyToken(user4.UID, token4)
	if err != nil {
		panic(err)
	}
	if !ok4 {
		panic("token4 Verify fail")
	}

	ok4, _, err = usermgr.FindUserByMobile(mobile)
	if err != nil {
		panic(err)
	}
	if !ok4 {
		panic("mobile not found")
	}

	// 手机直接登录
	mobilecode, _, _, err = usermgr.LoginMobileApplyCode(mobile)
	if err != nil {
		panic(err)
	}
	user4, _, _, err = usermgr.LoginMobile(mobile, mobilecode)
	if err != nil {
		panic(err)
	}
}
