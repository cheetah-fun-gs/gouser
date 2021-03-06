// Package gouser 注册方法 仅注册
package gouser

import (
	"database/sql"
	"fmt"
	"time"

	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
	mlogger "github.com/cheetah-fun-gs/goplus/multier/multilogger"
)

// RegisterLAPD 密码用户注册
func (mgr *UserMgr) RegisterLAPD(uid, rawPassword string) (*User, error) {
	now := time.Now()
	_, nickname, avatar, extra := mgr.generateUID()

	data := &ModelUser{
		UID:       uid,
		Password:  mgr.getPassword(rawPassword),
		Nickname:  nickname,
		Avatar:    avatar,
		Extra:     extra,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	return &User{
		mgr: mgr,
		UserData: &UserData{
			ID:        int(aid),
			UID:       uid,
			Nickname:  nickname,
			Avatar:    avatar,
			LastLogin: now.Unix(),
			Created:   now.Unix(),
		},
	}, nil
}

// RegisterEmailApplyCode 邮件用户注册申请code
func (mgr *UserMgr) RegisterEmailApplyCode(email string) (code string, expire int, err error) {
	return mgr.ApplyCode(0, email)
}

// RegisterEmail 邮件用户注册
func (mgr *UserMgr) RegisterEmail(email, code string) (*User, error) {
	ok, err := mgr.VerifyCode(code, email)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("code is invalid")
	}

	now := time.Now()
	uid, nickname, avatar, extra := mgr.generateUID()
	data := &ModelUser{
		UID:       uid,
		Email:     sql.NullString{Valid: true, String: email},
		Nickname:  nickname,
		Avatar:    avatar,
		Extra:     extra,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	return &User{
		mgr: mgr,
		UserData: &UserData{
			ID:        int(aid),
			UID:       uid,
			Email:     email,
			Nickname:  nickname,
			Avatar:    avatar,
			LastLogin: now.Unix(),
			Created:   now.Unix(),
		},
	}, nil
}

// RegisterMobileApplyCode 手机用户注册申请code
func (mgr *UserMgr) RegisterMobileApplyCode(mobile string) (code string, expire, retry int, err error) {
	return mgr.ApplyCodeAntiReplay(mobile, 0, 0, mobile)
}

// RegisterMobile 手机用户注册
func (mgr *UserMgr) RegisterMobile(mobile, code string) (*User, error) {
	ok, err := mgr.VerifyCode(code, mobile)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("code is invalid")
	}

	return mgr.registerMobile(mobile)
}

func (mgr *UserMgr) registerMobile(mobile string) (*User, error) {
	now := time.Now()
	uid, nickname, avatar, extra := mgr.generateUID()
	data := &ModelUser{
		UID:       uid,
		Mobile:    sql.NullString{Valid: true, String: mobile},
		Nickname:  nickname,
		Avatar:    avatar,
		Extra:     extra,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	return &User{
		mgr: mgr,
		UserData: &UserData{
			ID:        int(aid),
			UID:       uid,
			Mobile:    mobile,
			Nickname:  nickname,
			Avatar:    avatar,
			LastLogin: now.Unix(),
			Created:   now.Unix(),
		},
	}, nil
}

// RegisterTourist 游客注册
func (mgr *UserMgr) RegisterTourist() (*User, error) {
	now := time.Now()
	uid, nickname, avatar, extra := mgr.generateUID()
	data := &ModelUser{
		UID:       uid,
		Nickname:  nickname,
		Avatar:    avatar,
		Extra:     extra,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	aid, err := sqlplus.LastInsertId(mgr.db.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	return &User{
		mgr: mgr,
		UserData: &UserData{
			ID:        int(aid),
			UID:       uid,
			Nickname:  nickname,
			Avatar:    avatar,
			LastLogin: now.Unix(),
			Created:   now.Unix(),
		},
	}, nil
}

// RegisterAuth 第三方认证注册
func (mgr *UserMgr) RegisterAuth(authName string, v interface{}) (*User, error) {
	authUID, authExtra, err := mgr.VerifyAuth(authName, v)
	if err != nil {
		return nil, err
	}

	return mgr.registerAuth(authName, authUID, authExtra)
}

func (mgr *UserMgr) registerAuth(authName, authUID, authExtra string) (*User, error) {
	now := time.Now()
	uid, nickname, avatar, _ := mgr.generateUID()

	tx, err := mgr.db.Begin()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				mlogger.WarnN(mgr.mlogname, "RegisterAuth Rollback %v %v err: %v", authName, authUID, errRollback)
			}
		}
	}()

	data := &ModelUser{
		UID:       uid,
		Nickname:  nickname,
		Avatar:    avatar,
		LastLogin: now,
		Created:   now,
		Updated:   now,
	}

	query, args := sqlplus.GenInsert(mgr.tableUser.Name, data)
	var aid int
	aid, err = sqlplus.LastInsertId(tx.Exec(query, args...))
	if err != nil {
		return nil, err
	}

	authData := &ModelUserAuth{
		UID:       uid,
		AuthName:  authName,
		AuthUID:   authUID,
		AuthExtra: authExtra,
		Created:   now,
		Updated:   now,
	}
	authQuery, authArgs := sqlplus.GenInsert(mgr.tableUserAuth.Name, authData)
	_, err = sqlplus.LastInsertId(tx.Exec(authQuery, authArgs...))
	if err != nil {
		return nil, err
	}

	if errCommit := tx.Commit(); errCommit != nil {
		return nil, errCommit
	}

	return &User{
		mgr: mgr,
		UserData: &UserData{
			ID:        int(aid),
			UID:       uid,
			Nickname:  nickname,
			Avatar:    avatar,
			LastLogin: now.Unix(),
			Created:   now.Unix(),
		},
	}, nil
}
