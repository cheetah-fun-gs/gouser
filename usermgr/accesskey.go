package usermgr

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"

	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
)

type accessKeyMgr struct {
	db                 *sql.DB
	tableUserAccessKey *modelTable // 访问密钥表
}

// Get 回源方法
func (akm *accessKeyMgr) Get(dest interface{}, args ...interface{}) (bool, error) {
	uid := args[0].(string)
	aid := args[1].(int)
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ? AND id = ?;", akm.tableUserAccessKey.Name)
	queryArgs := []interface{}{uid, aid}
	rows, err := akm.db.Query(query, queryArgs...)
	if err != nil {
		return false, err
	}

	result := &ModelUserAccessKey{}
	if err = sqlplus.Get(rows, result); err != nil {
		return false, err
	}

	if result.ExpireAt.Valid && result.ExpireAt.Time.Before(time.Now()) {
		return false, fmt.Errorf("accessKey expired")
	}
	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(result.AccessKey))
	return true, nil
}

// Set ...
func (akm *accessKeyMgr) Set(data interface{}, args ...interface{}) error {
	return nil
}

// Del ...
func (akm *accessKeyMgr) Del(args ...interface{}) error {
	return nil
}
