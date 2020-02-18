package usermgr

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"

	sqlplus "github.com/cheetah-fun-gs/goplus/dao/sql"
)

type accessKeyCacher struct {
	db                 *sql.DB
	tableUserAccessKey *modelTable // 访问密钥表
}

// Get 回源方法
func (akc *accessKeyCacher) Get(dest interface{}, args ...interface{}) (bool, error) {
	uid := args[0].(string)
	aid := args[1].(int)
	query := fmt.Sprintf("SELECT * FROM %v WHERE uid = ? AND id = ?;", akc.tableUserAccessKey.Name)
	queryArgs := []interface{}{uid, aid}
	rows, err := akc.db.Query(query, queryArgs...)
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

// Set 仅管理缓存, 外部管理源
func (akc *accessKeyCacher) Set(data interface{}, args ...interface{}) error {
	return nil
}

// Del 仅管理缓存, 外部管理源
func (akc *accessKeyCacher) Del(args ...interface{}) error {
	return nil
}
