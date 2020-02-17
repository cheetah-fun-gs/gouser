// Package usermgr 查找用户
package usermgr

import (
	"database/sql"
	"fmt"
	"time"

	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
	mlogger "github.com/cheetah-fun-gs/goplus/multier/multilogger"
	"github.com/cheetah-fun-gs/gouser"
)

func (mgr *UserMgr) findAuths(tx *sql.Tx, uid string) ([]*UserAuth, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ?;", mgr.tableUserAuth.Name)
	args := []interface{}{uid}

	rows, err := tx.Query(query, args...)
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

func (mgr *UserMgr) findAccessKeys(tx *sql.Tx, uid string) ([]*UserAccessKey, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ? AND (expire_at is NULL OR expire_at > ?);", mgr.tableUserAccessKey.Name)
	args := []interface{}{uid, time.Now()}

	rows, err := tx.Query(query, args...)
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

// FindAnyUser 根据用户名/邮箱/手机号 查找用户
func (mgr *UserMgr) FindAnyUser(any string) (*User, error) {
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ? OR email = ? OR mobile = ?;", mgr.tableUser.Name)
	args := []interface{}{any, any, any}

	var tx *sql.Tx
	var err error
	defer func() {
		if tx != nil {
			if err != nil {
				if errRollback := tx.Rollback(); errRollback != nil {
					mlogger.WarnN(gouser.MLoggerName, "FindUser Rollback err: %v", errRollback)
				}
			} else {
				if errCommit := tx.Commit(); errCommit != nil {
					mlogger.WarnN(gouser.MLoggerName, "FindUser Commit err: %v", errCommit)
				}
			}
		}
	}()

	var rows *sql.Rows
	if len(mgr.authMgrs) == 0 && !mgr.config.IsEnableAccessKey {
		// 不支持accesskey 和 第三方认证. 直接执行
		rows, err = mgr.db.Query(query, args...)
		if err != nil {
			return nil, err
		}
	} else {
		// 使用事务 虽然都是只读, 但是利用事务隔离 保障数据的原子性
		tx, err = mgr.db.Begin()
		if err != nil {
			return nil, err
		}

		rows, err = tx.Query(query, args...)
		if err != nil {
			return nil, err
		}
	}

	result := &ModelUser{}
	if err = sqlplus.Get(rows, result); err != nil {
		return nil, err
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

	if len(mgr.authMgrs) == 0 && !mgr.config.IsEnableAccessKey {
		return user, nil
	}

	auths := []*UserAuth{}
	accessKeys := []*UserAccessKey{}

	if len(mgr.authMgrs) > 0 {
		auths, err = mgr.findAuths(tx, user.UID)
		if err != nil {
			return nil, err
		}
	}
	if mgr.config.IsEnableAccessKey {
		accessKeys, err = mgr.findAccessKeys(tx, user.UID)
		if err != nil {
			return nil, err
		}
	}

	user.Auths = auths
	user.AccessKeys = accessKeys
	return user, nil
}
