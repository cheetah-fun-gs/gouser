package usermgr

import "time"

// ModelUser 用户表
type ModelUser struct {
	ID        int
	UID       string
	Password  string
	Email     string
	Mobile    string
	Extra     string
	IsDeleted bool
	Created   time.Time
	Updated   time.Time
}

// ModelAuth 用户和第三方认证绑定表
type ModelAuth struct {
	ID        int
	UID       string // ModelUser UID
	AuthName  string
	AuthUID   string
	AuthExtra string
	IsDeleted bool
	Created   time.Time
	Updated   time.Time
}

// ModelAccessKey 访问密钥
type ModelAccessKey struct {
	ID      int
	Key     string
	UID     string // ModelUser UID
	Comment string
	Created time.Time
	Updated time.Time
}
