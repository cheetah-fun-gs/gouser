package usermgr

import "time"

// ModelUser 用户表
type ModelUser struct {
	ID        int       `json:"id,omitempty"`
	UID       string    `json:"uid,omitempty"`
	Password  string    `json:"password,omitempty"`
	Email     string    `json:"email,omitempty"`
	Mobile    string    `json:"mobile,omitempty"`
	Extra     string    `json:"extra,omitempty"`
	IsDeleted bool      `json:"is_deleted,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	Updated   time.Time `json:"updated,omitempty"`
}

// ModelAuth 用户和第三方认证绑定表
type ModelAuth struct {
	ID        int       `json:"id,omitempty"`
	UID       string    `json:"uid,omitempty"` // ModelUser UID
	AuthName  string    `json:"auth_name,omitempty"`
	AuthUID   string    `json:"auth_uid,omitempty"`
	AuthExtra string    `json:"auth_extra,omitempty"`
	IsDeleted bool      `json:"is_deleted,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	Updated   time.Time `json:"updated,omitempty"`
}

// ModelAccessKey 访问密钥
type ModelAccessKey struct {
	ID      int       `json:"id,omitempty"`
	Key     string    `json:"key,omitempty"`
	UID     string    `json:"uid,omitempty"` // ModelUser UID
	Comment string    `json:"comment,omitempty"`
	Created time.Time `json:"created,omitempty"`
	Updated time.Time `json:"updated,omitempty"`
}
