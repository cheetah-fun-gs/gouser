package usermgr

import (
	"time"
)

// User 用户
type User struct {
	mgr        *UserMgr
	ID         int              `json:"id,omitempty"`
	UID        string           `json:"uid,omitempty"`
	Email      string           `json:"email,omitempty"`
	Mobile     string           `json:"mobile,omitempty"`
	Nickname   string           `json:"nickname,omitempty"`
	Avatar     string           `json:"avatar,omitempty"`
	Extra      string           `json:"extra,omitempty"`
	LastLogin  time.Time        `json:"last_login,omitempty"`
	Created    time.Time        `json:"created,omitempty"`
	Auths      []*UserAuth      `json:"auths,omitempty"`
	AccessKeys []*UserAccessKey `json:"access_keys,omitempty"`
}

// UserAuth 第三方认证
type UserAuth struct {
	ID        int       `json:"id,omitempty"`
	AuthName  string    `json:"auth_name,omitempty"`
	AuthUID   string    `json:"auth_uid,omitempty"`
	AuthExtra string    `json:"auth_extra,omitempty"`
	Created   time.Time `json:"created,omitempty"`
}

// UserAccessKey 访问密钥
type UserAccessKey struct {
	ID        int       `json:"id,omitempty"`
	AccessKey string    `json:"access_key,omitempty"`
	ExpireAt  time.Time `json:"expire_at,omitempty"`
	Comment   string    `json:"comment,omitempty"`
	Created   time.Time `json:"created,omitempty"`
}
