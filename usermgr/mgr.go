package usermgr

import (
	uuidplus "github.com/cheetah-fun-gs/goplus/uuid"
	"github.com/cheetah-fun-gs/gouser/authmgr"
	"github.com/cheetah-fun-gs/gouser/tokenmgr"
	redigo "github.com/gomodule/redigo/redis"
)

// UserMgr 用户管理器
type UserMgr struct {
	tokenmgr          tokenmgr.TokenMgr               // token 管理器
	sendEmailCode     func(email, code string) error  // 发送邮箱验证码
	sendMobileCode    func(mobile, code string) error // 发送短信验证码
	generateUID       func() (uid, extra string)      // 生成一个全新的uid和扩展信息
	generateAccessKey func(uid string) string         // 生成一个全新的AccessKey
	authMgrs          []authmgr.AuthMgr               // 支持的认证方式
	config            *Config
	name              string
}

// Config ...
type Config struct {
	TokenExpire        int    // token 超时时间
	IsOpenAccessKey    bool   // 是否打开访问密钥功能
	UIDFieldType       string // uid 字段类型 默认 char(22)
	AccessKeyFieldType string // access key 字段类型 默认 char(22)
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
			TokenExpire:        3600 * 2,
			UIDFieldType:       "Char(22)",
			AccessKeyFieldType: "Char(22)",
		}
	}

	mgr := &UserMgr{
		name:              name,
		config:            config,
		tokenmgr:          tokenmgr.New(name, pool, config.TokenExpire),
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
