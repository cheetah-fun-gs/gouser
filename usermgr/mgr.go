package usermgr

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/cheetah-fun-gs/goplus/cacher"
	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
	randplus "github.com/cheetah-fun-gs/goplus/math/rand"
	mlogger "github.com/cheetah-fun-gs/goplus/multier/multilogger"
	uuidplus "github.com/cheetah-fun-gs/goplus/uuid"
	"github.com/cheetah-fun-gs/gouser"
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
	tokenmgr           tokenmgr.TokenMgr                               // token 管理器
	tableUser          *modelTable                                     // 用户表
	tableUserAuth      *modelTable                                     // 第三方认证表
	tableUserAccessKey *modelTable                                     // 访问密钥表
	sendEmailCode      func(email, code string) error                  // 发送邮箱验证码
	sendMobileCode     func(mobile, code string) error                 // 发送短信验证码
	generateUID        func() (uid, nickname, avatar, extra string)    // 生成一个全新的uid和扩展信息
	generateCode       func() (code string, expire int)                // 生成一个校验码
	generateAccessKey  func() string                                   // 生成一个全新的AccessKey
	generateSign       func(accessKey string, data interface{}) string // AccessKey校验算法
	authMgrs           []authmgr.AuthMgr                               // 支持的第三方认证方式
	accessKeyCacher    *cacher.Cacher
	pool               *redigo.Pool
	db                 *sql.DB
	config             *Config
	name               string
	secret             string // 密钥
}

// Config ...
type Config struct {
	TokenExpire       int  // token 超时时间
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

func defaultSendEmailCode(email, code string) error {
	return nil
}

func defaultSendMobileCode(mobile, code string) error {
	return nil
}

func defaultGenerateCode() (code string, expire int) {
	return fmt.Sprintf("%03d", randplus.MustRandint(0, 999999)), 600
}

func getCodeKey(name, code string) string {
	return fmt.Sprintf("%s:%s:code", name, code)
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

	tableUserName := name + "_user"
	tableUserAuthName := name + "_user_auth"
	tableUserAccessKeyName := name + "_user_access_key"

	mgr := &UserMgr{
		name:     name,
		secret:   secret,
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
		sendEmailCode:     defaultSendEmailCode,
		sendMobileCode:    defaultSendMobileCode,
	}
	if config.IsEnableAccessKey {
		mgr.accessKeyCacher = cacher.New(tableUserAccessKeyName, pool, &accessKeyMgr{
			db: db,
			tableUserAccessKey: &modelTable{
				Name:      tableUserAccessKeyName,
				CreateSQL: fmt.Sprintf(TableUserAccessKey, tableUserAccessKeyName),
			}})
	}
	return mgr
}

// SetAuthMgr 设置第三方认证
func (mgr *UserMgr) SetAuthMgr(args ...authmgr.AuthMgr) {
	mgr.authMgrs = args
}

// SetTokenMgr 设置token管理器
func (mgr *UserMgr) SetTokenMgr(arg tokenmgr.TokenMgr) {
	mgr.tokenmgr = arg
}

// SetSendEmailCode ...
func (mgr *UserMgr) SetSendEmailCode(arg func(email, code string) error) {
	mgr.sendEmailCode = arg
}

// SetSendMobileCode ...
func (mgr *UserMgr) SetSendMobileCode(arg func(mobile, code string) error) {
	mgr.sendMobileCode = arg
}

// SetGenerateUID ...
func (mgr *UserMgr) SetGenerateUID(arg func() (uid, nickname, avatar, extra string)) {
	mgr.generateUID = arg
}

// SetGenerateCode ...
func (mgr *UserMgr) SetGenerateCode(arg func() (code string, expire int)) {
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

func (mgr *UserMgr) applyCode(content string) (code string, expire int, err error) {
	code, expire = mgr.generateCode()
	conn := mgr.pool.Get()
	defer conn.Close()

	codeKey := getCodeKey(mgr.name, code)
	var result string
	result, err = redigo.String(conn.Do("SET", codeKey, content, "EX", expire, "NX"))
	if err != nil {
		return
	}
	if result != "OK" {
		return "", 0, fmt.Errorf("code duplicate")
	}
	return
}

func (mgr *UserMgr) checkCode(code string) (ok bool, content string, err error) {
	conn := mgr.pool.Get()
	defer conn.Close()

	codeKey := getCodeKey(mgr.name, code)
	content, err = redigo.String(conn.Do("GET", codeKey))
	if err != nil && err != redigo.ErrNil {
		return
	}
	if err == redigo.ErrNil {
		return false, "", nil
	}
	return
}

func (mgr *UserMgr) getPassword(rawPassword string) string {
	return uuidplus.NewV5(mgr.secret, rawPassword).Base62()
}

// RegisterLAPD 密码用户注册
func (mgr *UserMgr) RegisterLAPD(uid, rawPassword string) (*User, error) {
	now := time.Now()
	_, nickname, avatar, extra := mgr.generateUID()

	data := &ModelUser{
		UID:       uid,
		Password:  mgr.getPassword(rawPassword),
		Nickname:  nickname,
		Avatar:    avatar,
		Extra:     extra,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	return &User{
		mgr:       mgr,
		ID:        int(aid),
		UID:       uid,
		Nickname:  nickname,
		Avatar:    avatar,
		LastLogin: now.Unix(),
		Created:   now.Unix(),
	}, nil
}

// RegisterEmailApplyCode 邮件用户注册申请code
func (mgr *UserMgr) RegisterEmailApplyCode() (code string, expire int, err error) {
	return mgr.applyCode("RegisterEmail")
}

// RegisterEmail 邮件用户注册
func (mgr *UserMgr) RegisterEmail(email, code string) (*User, error) {
	ok, _, err := mgr.checkCode(code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("code is invalid")
	}

	now := time.Now()
	uid, nickname, avatar, extra := mgr.generateUID()
	data := &ModelUser{
		UID:       uid,
		Email:     sql.NullString{Valid: true, String: email},
		Nickname:  nickname,
		Avatar:    avatar,
		Extra:     extra,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	return &User{
		mgr:       mgr,
		ID:        int(aid),
		UID:       uid,
		Email:     email,
		Nickname:  nickname,
		Avatar:    avatar,
		LastLogin: now.Unix(),
		Created:   now.Unix(),
	}, nil
}

// RegisterMobileApplyCode 手机用户注册申请code
func (mgr *UserMgr) RegisterMobileApplyCode() (code string, expire int, err error) {
	return mgr.applyCode("RegisterMobile")
}

// RegisterMobile 手机用户注册
func (mgr *UserMgr) RegisterMobile(mobile, code string) (*User, error) {
	ok, _, err := mgr.checkCode(code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("code is invalid")
	}

	now := time.Now()
	uid, nickname, avatar, extra := mgr.generateUID()
	data := &ModelUser{
		UID:       uid,
		Mobile:    sql.NullString{Valid: true, String: mobile},
		Nickname:  nickname,
		Avatar:    avatar,
		Extra:     extra,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        int(aid),
		UID:       uid,
		Mobile:    mobile,
		Nickname:  nickname,
		Avatar:    avatar,
		LastLogin: now.Unix(),
		Created:   now.Unix(),
	}, nil
}

// RegisterTourist 游客注册
func (mgr *UserMgr) RegisterTourist() (*User, error) {
	now := time.Now()
	uid, nickname, avatar, extra := mgr.generateUID()
	data := &ModelUser{
		UID:       uid,
		Nickname:  nickname,
		Avatar:    avatar,
		Extra:     extra,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	return &User{
		mgr:       mgr,
		ID:        int(aid),
		UID:       uid,
		Nickname:  nickname,
		Avatar:    avatar,
		LastLogin: now.Unix(),
		Created:   now.Unix(),
	}, nil
}

// RegisterAuth 第三方注册
func (mgr *UserMgr) RegisterAuth(nickname, avatar, authName, authUID, authExtra string) (*User, error) {
	now := time.Now()
	uid, _, _, _ := mgr.generateUID()

	tx, err := mgr.db.Begin()
	if err != nil {
		return nil, err
	}

	data := &ModelUser{
		UID:       uid,
		Nickname:  nickname,
		Avatar:    avatar,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(tx.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	authData := &ModelUserAuth{
		UID:       uid,
		AuthName:  authName,
		AuthUID:   authUID,
		AuthExtra: authExtra,
		Created:   now,
		Updated:   now,
	}
	authQuery, authArgs := sqlplus.GenInsert(mgr.tableUserAuth.Name, authData)
	aidAuth, err := sqlplus.LastInsertId(tx.Exec(authQuery, authArgs...))
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			mlogger.WarnN(gouser.MLoggerName, "RegisterAuth Rollback err: %v", errRollback)
		}
		return nil, err
	}

	return &User{
		mgr:       mgr,
		ID:        int(aid),
		UID:       uid,
		Nickname:  nickname,
		Avatar:    avatar,
		LastLogin: now.Unix(),
		Created:   now.Unix(),
		Auths: []*UserAuth{
			&UserAuth{
				ID:        int(aidAuth),
				AuthName:  authName,
				AuthUID:   authUID,
				AuthExtra: authExtra,
				Created:   now.Unix(),
			},
		},
	}, nil
}

// VerifyToken 验证token
func (mgr *UserMgr) VerifyToken(uid, from, token string) (ok bool, err error) {
	return mgr.tokenmgr.Verify(uid, from, token)
}

// VerifySign 验证sign: sign由access key和请求数据(或请求数据部分字段)计算得到
func (mgr *UserMgr) VerifySign(uid string, accessKeyID int, data interface{}, sign string) (ok bool, err error) {
	var accessKey string
	if ok, err := mgr.accessKeyCacher.Get(&accessKey, uid, accessKeyID); err != nil {
		return false, err
	} else if !ok {
		return false, fmt.Errorf("accessKey not found")
	}

	return sign == mgr.generateSign(accessKey, data), nil
}
