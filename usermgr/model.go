package usermgr

import (
	"database/sql"
	"time"
)

// sql table
const (
	TableUser = `CREATE TABLE IF NOT EXISTS %v (
		id int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增长ID',
		uid char(22) NOT NULL COMMENT '用户ID',
		password char(22) NOT NULL COMMENT '密码',
		email varchar(45) DEFAULT NULL COMMENT '邮箱',
		mobile varchar(45) DEFAULT NULL COMMENT '手机号',
		nickname varchar(64) NOT NULL COMMENT '昵称',
		avatar varchar(1024) NOT NULL COMMENT '头像',
		extra varchar(1024) NOT NULL COMMENT '扩展信息',
		last_login datetime NOT NULL COMMENT '最后登录时间',
		created datetime NOT NULL COMMENT '创建时间',
		updated datetime NOT NULL COMMENT '更新时间',
		PRIMARY KEY (id),
		UNIQUE KEY uniq_uid (uid),
		UNIQUE KEY uniq_email (email),
		UNIQUE KEY uniq_mobile (mobile),
		KEY idx_nickname (nickname),
		KEY idx_last_login (last_login),
		KEY idx_created (created),
		KEY idx_updated (updated)
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表'`
	TableUserAuth = `CREATE TABLE IF NOT EXISTS %v (
		id int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增长ID',
		uid char(22) NOT NULL COMMENT 'uid',
		auth_name varchar(45) NOT NULL COMMENT '认证名称',
		auth_uid varchar(128) NOT NULL COMMENT '第三方唯一ID',
		auth_extra varchar(1024) NOT NULL COMMENT '第三方信息',
		created datetime NOT NULL COMMENT '创建时间',
		updated datetime NOT NULL COMMENT '更新时间',
		PRIMARY KEY (id),
		UNIQUE KEY uniq_uid_auth_name (uid,auth_name),
		KEY idx_auth_uid (auth_uid),
		KEY idx_created (created),
		KEY idx_updated (updated)
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='第三方认证表'`
	TableUserAccessKey = `CREATE TABLE IF NOT EXISTS %v (
		id int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增长ID',
		access_key char(22) NOT NULL COMMENT '访问密钥',
		uid char(22) NOT NULL COMMENT '用户ID',
		expire_at datetime NOT NULL COMMENT '到期时间',
		comment varchar(200) NOT NULL COMMENT '密钥注释',
		created datetime NOT NULL COMMENT '创建时间',
		updated datetime NOT NULL COMMENT '更新时间',
		PRIMARY KEY (id),
		UNIQUE KEY uniq_access_key (access_key),
		KEY idx_uid (uid),
		KEY idx_created (created),
		KEY idx_updated (updated),
		KEY idx_expire_at (expire_at)
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='访问密钥表'`
)

// ModelUser 用户表
type ModelUser struct {
	ID        int            `json:"id,omitempty"`
	UID       string         `json:"uid,omitempty"`
	Password  string         `json:"password,omitempty"`
	Email     sql.NullString `json:"email,omitempty"`
	Mobile    sql.NullString `json:"mobile,omitempty"`
	Nickname  string         `json:"nickname,omitempty"`
	Avatar    string         `json:"avatar,omitempty"`
	Extra     string         `json:"extra,omitempty"`
	LastLogin time.Time      `json:"last_login,omitempty"`
	Created   time.Time      `json:"created,omitempty"`
	Updated   time.Time      `json:"updated,omitempty"`
}

// ModelUserAuth 用户和第三方认证绑定表
type ModelUserAuth struct {
	ID        int       `json:"id,omitempty"`
	UID       string    `json:"uid,omitempty"` // ModelUser UID
	AuthName  string    `json:"auth_name,omitempty"`
	AuthUID   string    `json:"auth_uid,omitempty"`
	AuthExtra string    `json:"auth_extra,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	Updated   time.Time `json:"updated,omitempty"`
}

// ModelUserAccessKey 访问密钥
type ModelUserAccessKey struct {
	ID        int       `json:"id,omitempty"`
	AccessKey string    `json:"access_key,omitempty"`
	UID       string    `json:"uid,omitempty"` // ModelUser UID
	ExpireAt  time.Time `json:"expire_at,omitempty"`
	Comment   string    `json:"comment,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	Updated   time.Time `json:"updated,omitempty"`
}
