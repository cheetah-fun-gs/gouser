package usermgr

import (
	"database/sql"
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
	tokenmgr           tokenmgr.TokenMgr               // token 管理器
	tableUser          *modelTable                     // 用户表
	tableUserAuth      *modelTable                     // 第三方认证表
	tableUserAccessKey *modelTable                     // 访问密钥表
	sendEmailCode      func(email, code string) error  // 发送邮箱验证码
	sendMobileCode     func(mobile, code string) error // 发送短信验证码
	generateUID        func() (uid, extra string)      // 生成一个全新的uid和扩展信息
	generateAccessKey  func(uid string) string         // 生成一个全新的AccessKey
	authMgrs           []authmgr.AuthMgr               // 支持的第三方认证方式
	pool               *redigo.Pool
	db                 *sql.DB
	config             *Config
	name               string
}

// Config ...
type Config struct {
	TokenExpire        int  // token 超时时间
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
func New(name string, pool *redigo.Pool, db *sql.DB, args ...interface{}) *UserMgr {
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
		pool:     pool,
		db:       db,
		tokenmgr: tokenmgr.New(name, pool, config.TokenExpire),
		tableUser: &modelTable{
			Name:      name + "_user",
			CreateSQL: fmt.Sprintf(TableUser, name+"_user"),
		},
		tableUserAuth: &modelTable{
			Name:      name + "_user_auth",
			CreateSQL: fmt.Sprintf(TableUserAuth, name+"_user_auth"),
		},
		tableUserAccessKey: &modelTable{
			Name:      name + "_user_access_key",
			CreateSQL: fmt.Sprintf(TableUserAccessKey, name+"_user_access_key"),
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
func (mgr *UserMgr) SetTableUser(tableName, tableCreateSQL string) error {
	mgr.tableUser = &modelTable{
		Name:      tableName,
		CreateSQL: tableCreateSQL,
	}
	return nil
}

// SetTableAuth ...
func (mgr *UserMgr) SetTableAuth(tableName, tableCreateSQL string) error {
	mgr.tableUserAuth = &modelTable{
		Name:      tableName,
		CreateSQL: tableCreateSQL,
	}
	return nil
}

// SetTableAccessKey ...
func (mgr *UserMgr) SetTableAccessKey(tableName, tableCreateSQL string) error {
	mgr.tableUserAccessKey = &modelTable{
		Name:      tableName,
		CreateSQL: tableCreateSQL,
	}
	return nil
}

// EnsureTables 确保sql表已建立
func (mgr *UserMgr) EnsureTables() error {
	if _, err := mgr.db.Exec(mgr.tableUser.CreateSQL); err != nil {
		return err
	}
	if len(mgr.authMgrs) > 0 {
		if _, err := mgr.db.Exec(mgr.tableUserAuth.CreateSQL); err != nil {
			return err
		}
	}
	if mgr.config.IsSupportAccessKey {
		if _, err := mgr.db.Exec(mgr.tableUserAccessKey.CreateSQL); err != nil {
			return err
		}
	}
	return nil
}

// TablesCreateSQL 建表语句
func (mgr *UserMgr) TablesCreateSQL() []string {
	result := []string{mgr.tableUser.CreateSQL}
	if len(mgr.authMgrs) > 0 {
		result = append(result, mgr.tableUserAuth.CreateSQL)
	}
	if mgr.config.IsSupportAccessKey {
		result = append(result, mgr.tableUserAccessKey.CreateSQL)
	}
	return result
}
