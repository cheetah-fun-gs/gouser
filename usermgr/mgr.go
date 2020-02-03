package usermgr

import (
	"database/sql"
	"fmt"
	"time"

	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
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
	tokenmgr           tokenmgr.TokenMgr                            // token 管理器
	tableUser          *modelTable                                  // 用户表
	tableUserAuth      *modelTable                                  // 第三方认证表
	tableUserAccessKey *modelTable                                  // 访问密钥表
	sendEmailCode      func(email, code string) error               // 发送邮箱验证码
	sendMobileCode     func(mobile, code string) error              // 发送短信验证码
	generateUID        func() (uid, nickname, avatar, extra string) // 生成一个全新的uid和扩展信息
	generateCode       func() (code string, expire int)             // 生成一个校验码
	generateAccessKey  func() string                                // 生成一个全新的AccessKey
	authMgrs           []authmgr.AuthMgr                            // 支持的第三方认证方式
	pool               *redigo.Pool
	db                 *sql.DB
	config             *Config
	name               string
	secret             string // 密钥
}

// Config ...
type Config struct {
	TokenExpire        int  // token 超时时间
	IsSupportAccessKey bool // 是否支持访问密钥
}

func defaultGenerateUID() (uid, nickname, avatar, extra string) {
	uid = uuidplus.NewV4().Base62()
	return
}

func defaultGenerateAccessKey() string {
	return uuidplus.NewV4().Base62()
}

func defaultSendEmailCode(email, code string) error {
	return nil
}

func defaultSendMobileCode(mobile, code string) error {
	return nil
}

func defaultGenerateCode() (string, int) {
	return "", 600
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

	mgr := &UserMgr{
		name:     name,
		secret:   secret,
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
		generateCode:      defaultGenerateCode,
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
func (mgr *UserMgr) SetGenerateUID(v func() (uid, nickname, avatar, extra string)) {
	mgr.generateUID = v
}

// SetGenerateCode ...
func (mgr *UserMgr) SetGenerateCode(v func() (code string, expire int)) {
	mgr.generateCode = v
}

// SetGenerateAccessKey ...
func (mgr *UserMgr) SetGenerateAccessKey(v func() string) {
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
