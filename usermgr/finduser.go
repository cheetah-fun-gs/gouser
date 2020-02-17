// Package usermgr 查找用户
package usermgr

import (
	"database/sql"
	"fmt"

	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
)

func (mgr *UserMgr) findUser(query string, args ...interface{}) (bool, *User, error) {
	rows, err := mgr.db.Query(query, args...)
	if err != nil {
		return false, nil, err
	}

	result := &ModelUser{}
	if err = sqlplus.Get(rows, result); err == sql.ErrNoRows {
		return false, nil, nil
	} else if err != nil {
		return false, nil, err
	}

	user := &User{
		mgr:       mgr,
		ID:        result.ID,
		UID:       result.UID,
		Email:     result.Email.String,
		Mobile:    result.Mobile.String,
		Nickname:  result.Nickname,
		Avatar:    result.Avatar,
		Extra:     result.Extra,
		LastLogin: result.LastLogin.Unix(),
		Created:   result.Created.Unix(),
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
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ?;", mgr.tableUser.Name)
	args := []interface{}{uid}

	return mgr.findUser(query, args...)
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

	result := &ModelUserAuth{}
	if err = sqlplus.Get(rows, result); err == sql.ErrNoRows {
		return false, nil, nil
	} else if err != nil {
		return false, nil, err
	}

	return mgr.FindUserByUID(result.UID)
}
