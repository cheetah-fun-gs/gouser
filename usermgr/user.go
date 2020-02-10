package usermgr

import (
	"database/sql"
	"fmt"
	"strings"
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
	LastLogin  int64            `json:"last_login,omitempty"`
	Created    int64            `json:"created,omitempty"`
	Auths      []*UserAuth      `json:"auths,omitempty"`
	AccessKeys []*UserAccessKey `json:"access_keys,omitempty"`
}

// UserAuth 第三方认证
type UserAuth struct {
	ID        int    `json:"id,omitempty"`
	AuthName  string `json:"auth_name,omitempty"`
	AuthUID   string `json:"auth_uid,omitempty"`
	AuthExtra string `json:"auth_extra,omitempty"`
	Created   int64  `json:"created,omitempty"`
}

// UserAccessKey 访问密钥
type UserAccessKey struct {
	ID        int    `json:"id,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	ExpireAt  int64  `json:"expire_at,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Created   int64  `json:"created,omitempty"`
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
	if len(user.mgr.authMgrs) == 0 && !user.mgr.config.IsEnableAccessKey {
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

	if user.mgr.config.IsEnableAccessKey {
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
		Created:   now.Unix(),
	})
	return nil
}

// UnbindAuth 解绑第三方认证
func (user *User) UnbindAuth(authName string) error {
	var isMatch bool

	for _, v := range user.Auths {
		if v.AuthName == authName {
			isMatch = true
		}
	}
	if !isMatch {
		return fmt.Errorf("auth not found: %v", authName)
	}

	query := fmt.Sprintf("DELETE FROM %v WHERE uid = ? AND auth_name = ?;", user.mgr.tableUserAuth.Name)
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
func (user *User) UpdateInfo(nickname, avatar, extra *string) error {
	if nickname == nil && avatar == nil && extra == nil {
		return fmt.Errorf("no valid params")
	}

	splits := []string{}
	args := []interface{}{}
	if nickname != nil {
		splits = append(splits, "nickname = ?")
		args = append(args, nickname)
	}
	if avatar != nil {
		splits = append(splits, "avatar = ?")
		args = append(args, avatar)
	}
	if extra != nil {
		splits = append(splits, "extra = ?")
		args = append(args, extra)
	}

	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set %v, updated = ? WHERE id = ?;",
		strings.Join(splits, ", "), user.mgr.tableUser.Name)
	args = append(args, now, user.ID)
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	if nickname != nil {
		user.Nickname = *nickname
	}
	if avatar != nil {
		user.Avatar = *avatar
	}
	if extra != nil {
		user.Extra = *extra
	}
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

// GenerateAccessKey 生成一个 access key
func (user *User) GenerateAccessKey(comment string, expireAts ...time.Time) (*UserAccessKey, error) {
	expireAt := sql.NullTime{}
	if len(expireAts) > 0 {
		expireAt.Valid = true
		expireAt.Time = expireAts[0]
	}
	accessKey := user.mgr.generateAccessKey()
	now := time.Now()
	data := &ModelUserAccessKey{
		AccessKey: accessKey,
		UID:       user.UID,
		Comment:   comment,
		ExpireAt:  expireAt,
		Created:   now,
		Updated:   now,
	}
	query, args := sqlplus.GenInsert(user.mgr.tableUserAccessKey.Name, data)
	aid, err := sqlplus.LastInsertId(user.mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	userAccessKey := &UserAccessKey{
		ID:        int(aid),
		AccessKey: accessKey,
		Comment:   comment,
		Created:   now.Unix(),
	}
	if expireAt.Valid {
		userAccessKey.ExpireAt = expireAt.Time.Unix()
	}
	user.AccessKeys = append(user.AccessKeys, userAccessKey)
	return userAccessKey, nil
}

// UpdateAccessKey 更新一个 access key
func (user *User) UpdateAccessKey(accessKeyID int, comment *string, expireAt *time.Time) error {
	var userAccessKey *UserAccessKey

	for _, v := range user.AccessKeys {
		if v.ID == accessKeyID {
			userAccessKey = v
		}
	}

	if userAccessKey == nil {
		return fmt.Errorf("accessKey not found: %v", accessKeyID)
	}

	if comment == nil && expireAt == nil {
		return fmt.Errorf("no valid params")
	}

	splits := []string{}
	args := []interface{}{}
	if comment != nil {
		splits = append(splits, "comment = ?")
		args = append(args, comment)
	}
	if expireAt != nil {
		splits = append(splits, "expire_at = ?")
		args = append(args, expireAt)
	}

	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set %v, updated = ? WHERE id = ?;",
		strings.Join(splits, ", "), user.mgr.tableUserAccessKey.Name)
	args = append(args, now, accessKeyID)
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	if comment != nil {
		userAccessKey.Comment = *comment
	}
	if expireAt != nil {
		userAccessKey.ExpireAt = (*expireAt).Unix()
	}
	return nil
}

// DeleteAccessKey 更新一个 access key
func (user *User) DeleteAccessKey(accessKeyID int) error {
	var isMatch bool

	for _, v := range user.AccessKeys {
		if v.ID == accessKeyID {
			isMatch = true
		}
	}

	if !isMatch {
		return fmt.Errorf("accessKey not found: %v", accessKeyID)
	}

	query := fmt.Sprintf("DELETE FROM %v WHERE id = ?;", user.mgr.tableUserAccessKey.Name)
	args := []interface{}{accessKeyID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	accessKeys := []*UserAccessKey{}
	for _, v := range user.AccessKeys {
		if v.ID != accessKeyID {
			accessKeys = append(accessKeys, v)
		}
	}
	user.AccessKeys = accessKeys
	return nil
}
