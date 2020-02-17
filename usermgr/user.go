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
	mgr       *UserMgr
	ID        int    `json:"id,omitempty"`
	UID       string `json:"uid,omitempty"`
	Email     string `json:"email,omitempty"`
	Mobile    string `json:"mobile,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	Extra     string `json:"extra,omitempty"`
	LastLogin int64  `json:"last_login,omitempty"`
	Created   int64  `json:"created,omitempty"`
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
func (user *User) Login() (token string, deadline int64, err error) {
	return user.LoginWithFrom(fromDefault)
}

// LoginWithFrom 登录
func (user *User) LoginWithFrom(from string) (token string, deadline int64, err error) {
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
func (user *User) Logout() error {
	return user.LogoutWithFrom(fromDefault)
}

// LogoutWithFrom 登出
func (user *User) LogoutWithFrom(from string) error {
	return user.mgr.tokenmgr.Clean(user.UID, from)
}

// Clean 清除账号
func (user *User) Clean() error {
	query := fmt.Sprintf("DELETE FROM %v WHERE id = ?;", user.mgr.tableUser.Name)
	args := []interface{}{user.ID}

	// 不支持accesskey 和 第三方认证. 直接执行
	if len(user.mgr.authMgrs) == 0 && !user.mgr.config.IsEnableAccessKey {
		_, err := user.mgr.db.Exec(query, args...)
		return err
	}

	// 使用事务
	tx, err := user.mgr.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				mlogger.WarnN(gouser.MLoggerName, "UserClean Rollback err: %v", errRollback)
			}
		}
	}()

	_, err = tx.Exec(query, args...)
	if err != nil {
		return err
	}

	if len(user.mgr.authMgrs) > 0 {
		queryAuth := fmt.Sprintf("DELETE FROM %v WHERE uid = ?;", user.mgr.tableUserAuth.Name)
		argsAuth := []interface{}{user.UID}
		_, err = tx.Exec(queryAuth, argsAuth...)
		if err != nil {
			return err
		}
	}

	if user.mgr.config.IsEnableAccessKey {
		queryAccessKey := fmt.Sprintf("DELETE FROM %v WHERE uid = ?;", user.mgr.tableUserAccessKey.Name)
		argsAccessKey := []interface{}{user.UID}
		_, err = tx.Exec(queryAccessKey, argsAccessKey...)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// BindAuth 绑定第三方认证
func (user *User) BindAuth(authName string, v interface{}) error {
	authUID, authExtra, err := user.mgr.VerifyAuth(authName, v)
	if err != nil {
		return err
	}

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
	_, err = sqlplus.LastInsertId(user.mgr.db.Exec(authQuery, authArgs...))
	return err
}

// UnbindAuth 解绑第三方认证
func (user *User) UnbindAuth(authName string) error {
	query := fmt.Sprintf("DELETE FROM %v WHERE uid = ? AND auth_name = ?;", user.mgr.tableUserAuth.Name)
	args := []interface{}{user.UID, authName}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}
	return nil
}

// GetAuths 获得第三方信息
func (user *User) GetAuths() ([]*UserAuth, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ?;", user.mgr.tableUserAuth.Name)
	args := []interface{}{user.UID}

	rows, err := user.mgr.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	result := []*ModelUserAuth{}
	if err = sqlplus.Select(rows, &result); err != nil {
		return nil, err
	}

	auths := []*UserAuth{}
	for _, val := range result {
		auths = append(auths, &UserAuth{
			ID:        val.ID,
			AuthName:  val.AuthName,
			AuthUID:   val.AuthUID,
			AuthExtra: val.AuthExtra,
			Created:   val.Created.Unix(),
		})
	}
	return auths, nil
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
	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set auth_extra = ?, updated = ? WHERE uid = ? AND auth_name = ?;",
		user.mgr.tableUserAuth.Name)
	args := []interface{}{authExtra, now, user.UID, authName}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}
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
	return user.mgr.ApplyCode(user.UID)
}

// UpdateEmail 更新uid
func (user *User) UpdateEmail(email, code string) error {
	ok, err := user.mgr.VerifyCode(code, user.UID)
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
func (user *User) UpdateMobileApplyCode(mobile string) (code string, expire, retry int, err error) {
	return user.mgr.ApplyCodeAntiReplay(mobile, user.UID)
}

// UpdateMobile 更新手机号
func (user *User) UpdateMobile(mobile, code string) error {
	ok, err := user.mgr.VerifyCode(code, user.UID)
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
	return user.mgr.ApplyCode(user.UID)
}

// UpdatePasswordWithCode 通过验证码更改密码
func (user *User) UpdatePasswordWithCode(rawPassword, code string) error {
	ok, err := user.mgr.VerifyCode(code, user.UID)
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

// GetAccessKeys 获取accesskeys isAll 是否包含过期的访问秘钥
func (user *User) GetAccessKeys(isAll bool) ([]*UserAccessKey, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ?", user.mgr.tableUserAccessKey.Name)
	args := []interface{}{user.UID}
	if !isAll {
		query += " AND (expire_at is NULL OR expire_at > ?);"
		args = append(args, time.Now())
	}

	rows, err := user.mgr.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	result := []*ModelUserAccessKey{}
	if err = sqlplus.Select(rows, &result); err != nil {
		return nil, err
	}

	accessKeys := []*UserAccessKey{}
	for _, val := range result {
		var expireAt int64
		if val.ExpireAt.Valid {
			expireAt = val.ExpireAt.Time.Unix()
		}
		accessKeys = append(accessKeys, &UserAccessKey{
			ID:        val.ID,
			AccessKey: val.AccessKey,
			ExpireAt:  expireAt,
			Comment:   val.Comment,
			Created:   val.Created.Unix(),
		})
	}
	return accessKeys, nil
}

// GenerateAccessKey 生成一个 access key
func (user *User) GenerateAccessKey(comment string, expireAts ...time.Time) (*UserAccessKey, error) {
	now := time.Now()

	expireAt := sql.NullTime{}
	if len(expireAts) > 0 {
		if expireAt.Time.Before(now) {
			return nil, fmt.Errorf("expire_at is before now")
		}
		expireAt.Valid = true
		expireAt.Time = expireAts[0]
	}
	accessKey := user.mgr.generateAccessKey()
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

	// 不用操作缓存 等自动回源
	return userAccessKey, nil
}

// UpdateAccessKeyComment 更新一个 access key 的 comment
func (user *User) UpdateAccessKeyComment(accessKeyID int, comment string) error {
	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set comment = ?, updated = ? WHERE id = ?;",
		user.mgr.tableUserAccessKey.Name)
	args := []interface{}{comment, now, accessKeyID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}
	return nil
}

// UpdateAccessKeyExpireAt 更新一个 access key的超时设置 expireAt为空表示永久有效
func (user *User) UpdateAccessKeyExpireAt(accessKeyID int, expireAt *time.Time) error {
	now := time.Now()
	query := fmt.Sprintf("UPDATE %v Set expire_at = ?, updated = ? WHERE id = ?;", user.mgr.tableUserAccessKey.Name)
	args := []interface{}{expireAt, now, accessKeyID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	// 失效删除缓存 生效的等自然回源
	if expireAt != nil && expireAt.Before(now) {
		if err = user.mgr.accessKeyCacher.Del(user.UID, accessKeyID); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAccessKey 更新一个 access key
func (user *User) DeleteAccessKey(accessKeyID int) error {
	query := fmt.Sprintf("DELETE FROM %v WHERE id = ?;", user.mgr.tableUserAccessKey.Name)
	args := []interface{}{accessKeyID}
	updateCount, err := sqlplus.RowsAffected(user.mgr.db.Exec(query, args...))
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrorNotFound
	}

	// 从缓存里删除
	if err = user.mgr.accessKeyCacher.Del(user.UID, accessKeyID); err != nil {
		return err
	}
	return nil
}
