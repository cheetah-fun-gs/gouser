package usermgr

import (
	"fmt"

	uuidplus "github.com/cheetah-fun-gs/goplus/uuid"
	"github.com/cheetah-fun-gs/gouser/authmgr"
	"github.com/cheetah-fun-gs/gouser/tokenmgr"
	redigo "github.com/gomodule/redigo/redis"
)

type modelTable struct {
	Name      string
	CreateSQL string
}

// UserMgr 用户管理器
type UserMgr struct {
	tokenmgr          tokenmgr.TokenMgr               // token 管理器
	tableUser         *modelTable                     // 用户表
	tableAuth         *modelTable                     // 第三方认证表
	tableAccessKey    *modelTable                     // 访问密钥表
	sendEmailCode     func(email, code string) error  // 发送邮箱验证码
	sendMobileCode    func(mobile, code string) error // 发送短信验证码
	generateUID       func() (uid, extra string)      // 生成一个全新的uid和扩展信息
	generateAccessKey func(uid string) string         // 生成一个全新的AccessKey
	authMgrs          []authmgr.AuthMgr               // 支持的第三方认证方式
	config            *Config
	name              string
}

// Config ...
type Config struct {
	TokenExpire        int  // token 超时时间
	IsSupportAuth      bool // 是否支持第三方认证
	IsSupportAccessKey bool // 是否支持访问密钥
}

func defaultGenerateUID() (uid, extra string) {
	uid = uuidplus.NewV4().Base62()
	return
}

func defaultGenerateAccessKey(uid string) string {
	return uuidplus.NewV4().Base62()
}

func defaultSendEmailCode(email, code string) error {
	return nil
}

func defaultSendMobileCode(mobile, code string) error {
	return nil
}

// New 一个新的用户管理器
func New(name string, pool *redigo.Pool, args ...interface{}) *UserMgr {
	var config *Config
	if len(args) > 0 {
		config = args[0].(*Config)
	} else {
		config = &Config{
			TokenExpire: 3600 * 2,
		}
	}

	mgr := &UserMgr{
		name:     name,
		config:   config,
		tokenmgr: tokenmgr.New(name, pool, config.TokenExpire),
		tableUser: &modelTable{
			Name:      name + "_user",
			CreateSQL: fmt.Sprintf(TableUser, name+"_user"),
		},
		tableAuth: &modelTable{
			Name:      name + "_user_auth",
			CreateSQL: fmt.Sprintf(TableAuth, name+"_user_auth"),
		},
		tableAccessKey: &modelTable{
			Name:      name + "_user_access_key",
			CreateSQL: fmt.Sprintf(TableAccessKey, name+"_user_access_key"),
		},
		generateUID:       defaultGenerateUID,
		generateAccessKey: defaultGenerateAccessKey,
		sendEmailCode:     defaultSendEmailCode,
		sendMobileCode:    defaultSendMobileCode,
	}
	return mgr
}

// SetAuthMgr 设置第三方认证
func (mgr *UserMgr) SetAuthMgr(v ...authmgr.AuthMgr) {
	mgr.authMgrs = v
}

// SetTokenMgr 设置token管理器
func (mgr *UserMgr) SetTokenMgr(v tokenmgr.TokenMgr) {
	mgr.tokenmgr = v
}

// SetSendEmailCode ...
func (mgr *UserMgr) SetSendEmailCode(v func(email, code string) error) {
	mgr.sendEmailCode = v
}

// SetSendMobileCode ...
func (mgr *UserMgr) SetSendMobileCode(v func(mobile, code string) error) {
	mgr.sendMobileCode = v
}

// SetGenerateUID ...
func (mgr *UserMgr) SetGenerateUID(v func() (uid, extra string)) {
	mgr.generateUID = v
}

// SetGenerateAccessKey ...
func (mgr *UserMgr) SetGenerateAccessKey(v func(uid string) string) {
	mgr.generateAccessKey = v
}

// SetTableUser ...
func (mgr *UserMgr) SetTableUser(tableName, tableCreateSQL string) {
	mgr.tableUser = &modelTable{
		Name:      tableName,
		CreateSQL: tableCreateSQL,
	}
}

// SetTableAuth ...
func (mgr *UserMgr) SetTableAuth(tableName, tableCreateSQL string) {
	mgr.tableAuth = &modelTable{
		Name:      tableName,
		CreateSQL: tableCreateSQL,
	}
}

// SetTableAccessKey ...
func (mgr *UserMgr) SetTableAccessKey(tableName, tableCreateSQL string) {
	mgr.tableAccessKey = &modelTable{
		Name:      tableName,
		CreateSQL: tableCreateSQL,
	}
}
