// Package gouser 查找用户
package gouser

import (
	"database/sql"
	"fmt"

	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
)

type userDataUIDCacher struct {
	db        *sql.DB
	tableUser *modelTable // 用户表
}

// Get 回源方法
func (uduc *userDataUIDCacher) Get(dest interface{}, args ...interface{}) (bool, error) {
	uid := args[0].(string)
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ?;", uduc.tableUser.Name)
	queryArgs := []interface{}{uid}

	ok, err := findUserData(uduc.db, dest.(*UserData), query, queryArgs...)
	if err != nil || !ok {
		return ok, err
	}
	return true, nil
}

// Set 仅管理缓存, 外部管理源
func (uduc *userDataUIDCacher) Set(data interface{}, args ...interface{}) error {
	return nil
}

// Del 仅管理缓存, 外部管理源
func (uduc *userDataUIDCacher) Del(args ...interface{}) error {
	return nil
}

func findUserData(db *sql.DB, userData *UserData, query string, args ...interface{}) (bool, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	result := &ModelUser{}
	if err = sqlplus.Get(rows, result); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	userData.ID = result.ID
	userData.UID = result.UID
	userData.Email = result.Email.String
	userData.Mobile = result.Mobile.String
	userData.Nickname = result.Nickname
	userData.Avatar = result.Avatar
	userData.Extra = result.Extra
	userData.LastLogin = result.LastLogin.Unix()
	userData.Created = result.Created.Unix()
	return true, nil
}

func (mgr *UserMgr) findUser(query string, args ...interface{}) (bool, *User, error) {
	result := &UserData{}
	ok, err := findUserData(mgr.db, result, query, args...)
	if err != nil || !ok {
		return ok, nil, err
	}

	user := &User{
		mgr:      mgr,
		UserData: result,
	}
	return true, user, nil
}

// FindUserByAny 根据用户名/邮箱/手机号 查找用户
func (mgr *UserMgr) FindUserByAny(any string) (bool, *User, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ? OR email = ? OR mobile = ?;", mgr.tableUser.Name)
	args := []interface{}{any, any, any}

	return mgr.findUser(query, args...)
}

// FindUserByUID 根据用户名 查找用户
func (mgr *UserMgr) FindUserByUID(uid string) (bool, *User, error) {
	result := &UserData{}
	ok, err := mgr.userDataUIDCacher.Get(result, uid)
	if err != nil || !ok {
		return ok, nil, err
	}

	user := &User{
		mgr:      mgr,
		UserData: result,
	}
	return true, user, nil
}

// FindUserByEmail 根据邮箱 查找用户
func (mgr *UserMgr) FindUserByEmail(email string) (bool, *User, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE email = ?;", mgr.tableUser.Name)
	args := []interface{}{email}

	return mgr.findUser(query, args...)
}

// FindUserByMobile 根据手机号 查找用户
func (mgr *UserMgr) FindUserByMobile(mobile string) (bool, *User, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE mobile = ?;", mgr.tableUser.Name)
	args := []interface{}{mobile}

	return mgr.findUser(query, args...)
}

// FindUserByAuth 根据第三方认证 查找用户
func (mgr *UserMgr) FindUserByAuth(authName, authUID string) (bool, *User, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE auth_name = ? AND auth_uid = ?;", mgr.tableUserAuth.Name)
	args := []interface{}{authName, authUID}

	rows, err := mgr.db.Query(query, args...)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	result := &ModelUserAuth{}
	if err = sqlplus.Get(rows, result); err == sql.ErrNoRows {
		return false, nil, nil
	} else if err != nil {
		return false, nil, err
	}

	return mgr.FindUserByUID(result.UID)
}
