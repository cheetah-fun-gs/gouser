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
	updatedCount, errUpdate := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if errUpdate != nil {
		mlogger.WarnN(gouser.MLoggerName, "UserLogin Update err: %v", errUpdate)
	} else if updatedCount == 0 {
		mlogger.WarnN(gouser.MLoggerName, "UserLogin Update err: not found")
	}
	return
}

// Logout 登出
func (user *User) Logout(from, token string) error {
	return user.mgr.tokenmgr.Clean(user.UID, from, token)
}

// Clean 清除账号
func (user *User) Clean() error {
	if len(user.mgr.authMgrs) == 0 && !user.mgr.config.IsSupportAccessKey {
		query := fmt.Sprintf("DELETE FROM %v WHERE id = ?;", user.mgr.tableUser.Name)
		args := []interface{}{user.ID}
		_, err := user.mgr.db.Exec(query, args...)
		return err
	}

	tx, err := user.mgr.db.Begin()
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %v WHERE id = ?;", user.mgr.tableUser.Name)
	args := []interface{}{user.ID}
	_, err = user.mgr.db.Exec(query, args...)
	if err != nil {
		return err
	}

	if len(user.mgr.authMgrs) > 0 {
		queryAuth := fmt.Sprintf("DELETE FROM %v WHERE uid = ?;", user.mgr.tableUserAuth.Name)
		argsAuth := []interface{}{user.UID}
		_, err = user.mgr.db.Exec(queryAuth, argsAuth...)
		if err != nil {
			return err
		}
	}

	if user.mgr.config.IsSupportAccessKey {
		queryAccessKey := fmt.Sprintf("DELETE FROM %v WHERE uid = ?;", user.mgr.tableUserAccessKey.Name)
		argsAccessKey := []interface{}{user.UID}
		_, err = user.mgr.db.Exec(queryAccessKey, argsAccessKey...)
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			mlogger.WarnN(gouser.MLoggerName, "UserClean Rollback err: %v", errRollback)
		}
		return err
	}
	return nil
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
	aidAuth, err := sqlplus.LastInsertId(user.mgr.db.Exec(authQuery, authArgs...))
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

// UnbindAuth 解绑第三方认证
func (user *User) UnbindAuth(authName string) error {
	query := fmt.Sprintf("DELETE FROM %v WHERE uid = ? AND auth_name = ?;",
		user.mgr.tableUserAuth.Name)
	args := []interface{}{user.UID, authName}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	auths := []*UserAuth{}
	for _, v := range user.Auths {
		if v.AuthName != authName {
			auths = append(auths, v)
		}
	}
	user.Auths = auths
	return nil
}

// UpdateInfo 更新用户信息
func (user *User) UpdateInfo(nickname, avatar, extra string) error {
	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set nickname = ?, avatar = ?, extra = ?, updated = ? WHERE id = ?;",
		user.mgr.tableUser.Name)
	args := []interface{}{nickname, avatar, extra, now, user.ID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
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
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	userAuth.AuthExtra = authExtra
	return nil
}

// UpdateUID 更新uid
func (user *User) UpdateUID(uid string) error {
	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set uid = ?, updated = ? WHERE id = ?;", user.mgr.tableUser.Name)
	args := []interface{}{uid, now, user.ID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	user.UID = uid
	return nil
}

// UpdateEmailApplyCode 更新邮箱申请code
func (user *User) UpdateEmailApplyCode() (code string, expire int, err error) {
	return user.mgr.applyCode("UpdateEmail")
}

// UpdateEmail 更新uid
func (user *User) UpdateEmail(email, code string) error {
	ok, _, err := user.mgr.checkCode(code)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("code is invalid")
	}

	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set email = ?, updated = ? WHERE id = ?;", user.mgr.tableUser.Name)
	args := []interface{}{email, now, user.ID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	user.Email = email
	return nil
}

// UpdateMobileApplyCode 更新手机号申请code
func (user *User) UpdateMobileApplyCode() (code string, expire int, err error) {
	return user.mgr.applyCode("UpdateMobile")
}

// UpdateMobile 更新手机号
func (user *User) UpdateMobile(mobile, code string) error {
	ok, _, err := user.mgr.checkCode(code)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("code is invalid")
	}

	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set mobile = ?, updated = ? WHERE id = ?;", user.mgr.tableUser.Name)
	args := []interface{}{mobile, now, user.ID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	user.Mobile = mobile
	return nil
}

// UpdatePasswordApplyCode 更改密码申请code
func (user *User) UpdatePasswordApplyCode() (code string, expire int, err error) {
	return user.mgr.applyCode("UpdatePassword")
}

// UpdatePasswordWithCode 通过验证码更改密码
func (user *User) UpdatePasswordWithCode(rawPassword, code string) error {
	ok, _, err := user.mgr.checkCode(code)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("code is invalid")
	}

	password := user.mgr.getPassword(rawPassword)
	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set password = ?, updated = ? WHERE id = ?;", user.mgr.tableUser.Name)
	args := []interface{}{password, now, user.ID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}
	return nil
}

// UpdatePasswordWithPassword 通过旧密码更改密码
func (user *User) UpdatePasswordWithPassword(oldRawPassword, newRawPassword string) error {
	oldPassword := user.mgr.getPassword(oldRawPassword)
	newPassword := user.mgr.getPassword(newRawPassword)
	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set password = ?, updated = ? WHERE id = ? AND password = ?;", user.mgr.tableUser.Name)
	args := []interface{}{newPassword, now, user.ID, oldPassword}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}
	return nil
}
