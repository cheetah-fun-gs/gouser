package usermgr

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/cheetah-fun-gs/goplus/cacher"
	randplus "github.com/cheetah-fun-gs/goplus/math/rand"
	uuidplus "github.com/cheetah-fun-gs/goplus/uuid"
	"github.com/cheetah-fun-gs/gouser/authmgr"
	"github.com/cheetah-fun-gs/gouser/tokenmgr"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	fromDefault = "default"
)

type modelTable struct {
	Name      string
	CreateSQL string
}

// UserMgr 用户管理器
type UserMgr struct {
	tokenmgr           tokenmgr.TokenMgr                               // token 管理器
	tableUser          *modelTable                                     // 用户表
	tableUserAuth      *modelTable                                     // 第三方认证表
	tableUserAccessKey *modelTable                                     // 访问密钥表
	generateUID        func() (uid, nickname, avatar, extra string)    // 生成一个全新的uid和扩展信息
	generateCode       func() string                                   // 生成一个校验码
	generateAccessKey  func() string                                   // 生成一个全新的AccessKey
	generateSign       func(accessKey string, data interface{}) string // AccessKey校验算法
	authMgrs           []authmgr.AuthMgr                               // 支持的第三方认证方式
	accessKeyCacher    *cacher.Cacher
	pool               *redigo.Pool
	db                 *sql.DB
	config             *Config
	name               string
	secret             string // 密钥
	mlogname           string
}

// Config ...
type Config struct {
	TokenExpire       int  // token 超时时间
	CodeExpire        int  // 验证码过期时间
	CodeRetry         int  // 验证码重试间隔
	IsEnableAccessKey bool // 是否支持访问密钥
}

func defaultGenerateUID() (uid, nickname, avatar, extra string) {
	uid = uuidplus.NewV4().Base62()
	return
}

func defaultGenerateAccessKey() string {
	return uuidplus.NewV4().Base62()
}

func defaultGenerateSign(accessKey string, data interface{}) string {
	ts := data.(int64)
	h := md5.New()
	h.Write([]byte(accessKey))
	h.Write([]byte(strconv.Itoa(int(ts))))
	return hex.EncodeToString(h.Sum(nil))
}

func defaultGenerateCode() string {
	return fmt.Sprintf("%03d", randplus.MustRandint(0, 999999))
}

// New 一个新的用户管理器
func New(name, secret string, pool *redigo.Pool, db *sql.DB, configs ...Config) *UserMgr {
	var config *Config
	if len(configs) == 0 {
		config = &Config{}
	} else {
		config = &configs[0]
	}
	if config.TokenExpire == 0 {
		config.TokenExpire = 3600 * 2
	}
	if config.CodeExpire == 0 {
		config.CodeExpire = 600
	}
	if config.CodeRetry == 0 {
		config.CodeRetry = 60
	}

	tableUserName := name + "_user"
	tableUserAuthName := name + "_user_auth"
	tableUserAccessKeyName := name + "_user_access_key"

	mgr := &UserMgr{
		name:     name,
		secret:   secret,
		mlogname: "default",
		config:   config,
		pool:     pool,
		db:       db,
		tokenmgr: tokenmgr.New(name, pool, config.TokenExpire),
		tableUser: &modelTable{
			Name:      tableUserName,
			CreateSQL: fmt.Sprintf(TableUser, tableUserName),
		},
		tableUserAuth: &modelTable{
			Name:      tableUserAuthName,
			CreateSQL: fmt.Sprintf(TableUserAuth, tableUserAuthName),
		},
		tableUserAccessKey: &modelTable{
			Name:      tableUserAccessKeyName,
			CreateSQL: fmt.Sprintf(TableUserAccessKey, tableUserAccessKeyName),
		},
		generateUID:       defaultGenerateUID,
		generateAccessKey: defaultGenerateAccessKey,
		generateSign:      defaultGenerateSign,
		generateCode:      defaultGenerateCode,
	}
	if config.IsEnableAccessKey {
		mgr.accessKeyCacher = cacher.New(tableUserAccessKeyName, pool, &accessKeyCacher{
			db: db,
			tableUserAccessKey: &modelTable{
				Name:      tableUserAccessKeyName,
				CreateSQL: fmt.Sprintf(TableUserAccessKey, tableUserAccessKeyName),
			}})
	}
	return mgr
}

// SetMLogName 设置日志
func (mgr *UserMgr) SetMLogName(name string) {
	mgr.mlogname = name
	mgr.accessKeyCacher.SetMLogName(name)
}

// SetAuthMgr 设置第三方认证
func (mgr *UserMgr) SetAuthMgr(args ...authmgr.AuthMgr) {
	mgr.authMgrs = args
}

// SetTokenMgr 设置token管理器
func (mgr *UserMgr) SetTokenMgr(arg tokenmgr.TokenMgr) {
	mgr.tokenmgr = arg
}

// SetGenerateUID ...
func (mgr *UserMgr) SetGenerateUID(arg func() (uid, nickname, avatar, extra string)) {
	mgr.generateUID = arg
}

// SetGenerateCode ...
func (mgr *UserMgr) SetGenerateCode(arg func() string) {
	mgr.generateCode = arg
}

// SetGenerateAccessKey ...
func (mgr *UserMgr) SetGenerateAccessKey(arg func() string) {
	mgr.generateAccessKey = arg
}

// SetGenerateSign ...
func (mgr *UserMgr) SetGenerateSign(arg func(accessKey string, data interface{}) string) {
	mgr.generateSign = arg
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
	if mgr.config.IsEnableAccessKey {
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
	if mgr.config.IsEnableAccessKey {
		result = append(result, mgr.tableUserAccessKey.CreateSQL)
	}
	return result
}

func (mgr *UserMgr) getPassword(rawPassword string) string {
	return uuidplus.NewV5(mgr.secret, rawPassword).Base62()
}

// VerifyToken 验证token
func (mgr *UserMgr) VerifyToken(uid, from, token string) (ok bool, err error) {
	return mgr.tokenmgr.Verify(uid, from, token)
}

// VerifySign 验证sign: sign由access key和请求数据(或请求数据部分字段)计算得到
func (mgr *UserMgr) VerifySign(uid string, accessKeyID int, data interface{}, sign string) (ok bool, err error) {
	if !mgr.config.IsEnableAccessKey {
		return false, fmt.Errorf("IsEnableAccessKey is not enable")
	}
	var accessKey string
	if ok, err := mgr.accessKeyCacher.Get(&accessKey, uid, accessKeyID); err != nil {
		return false, err
	} else if !ok {
		return false, fmt.Errorf("accessKey not found")
	}

	return sign == mgr.generateSign(accessKey, data), nil
}

// VerifyAuth 验证第三方凭证
func (mgr *UserMgr) VerifyAuth(authName string, v interface{}) (authUID, authExtra string, err error) {
	for _, auth := range mgr.authMgrs {
		if auth.GetName() == authName {
			return auth.Verify(v)
		}
	}
	return "", "", fmt.Errorf("authName is not support")
}
