package usermgr

import (
	"fmt"
	"time"

	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
	mlogger "github.com/cheetah-fun-gs/goplus/multier/multilogger"
	"github.com/cheetah-fun-gs/gouser"
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

// Login 登录
func (user *User) Login(from string) (token string, deadline int64, err error) {
	token, deadline, err = user.mgr.tokenmgr.Generate(user.UID, from)
	if err != nil {
		return
	}

	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set last_login = ?, updated = ? WHERE id = ?;",
		user.mgr.tableUser.Name)
	args := []interface{}{now, now, user.ID}
	_, errUpdate := user.mgr.db.Exec(query, args...)
	if errUpdate != nil {
		mlogger.WarnN(gouser.MLoggerName, "UserLogin Update err: %v", errUpdate)
	}
	return
}

// Logout 登出
func (user *User) Logout(from, token string) error {
	return user.mgr.tokenmgr.Clean(user.UID, from, token)
}

// BindAuth 绑定第三方认证
func (user *User) BindAuth(authName, authUID, authExtra string) error {
	now := time.Now()
	authData := &ModelUserAuth{
		UID:       user.UID,
		AuthName:  authName,
		AuthUID:   authUID,
		AuthExtra: authExtra,
		Created:   now,
		Updated:   now,
	}
	authQuery, authArgs := sqlplus.GenInsert(user.mgr.tableUserAuth.Name, authData)
	authResult, err := user.mgr.db.Exec(authQuery, authArgs...)
	if err != nil {
		return err
	}

	aidAuth, err := authResult.LastInsertId()
	if err != nil {
		return err
	}

	user.Auths = append(user.Auths, &UserAuth{
		ID:        int(aidAuth),
		AuthName:  authName,
		AuthUID:   authUID,
		AuthExtra: authExtra,
		Created:   now,
	})
	return nil
}

// UpdateInfo 更新用户信息
func (user *User) UpdateInfo(nickname, avatar, extra string) error {
	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set nickname = ?, avatar = ?, extra = ?, updated = ? WHERE id = ?;",
		user.mgr.tableUser.Name)
	args := []interface{}{nickname, avatar, extra, now, user.ID}
	_, err := user.mgr.db.Exec(query, args...)
	if err != nil {
		return err
	}

	user.Nickname = nickname
	user.Avatar = avatar
	user.Extra = extra
	return nil
}

// UpdateAuthInfo 更新第三方信息
func (user *User) UpdateAuthInfo(authName, authExtra string) error {
	var userAuth *UserAuth

	for _, v := range user.Auths {
		if v.AuthName == authName {
			userAuth = v
		}
	}
	if userAuth == nil {
		return fmt.Errorf("auth not found: %v", authName)
	}

	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set auth_extra = ?, updated = ? WHERE id = ?;",
		user.mgr.tableUserAuth.Name)
	args := []interface{}{authExtra, now, userAuth.ID}
	_, err := user.mgr.db.Exec(query, args...)
	if err != nil {
		return err
	}

	userAuth.AuthExtra = authExtra
	return nil
}
