// Package usermgr 登录并注册
package usermgr

import "fmt"

// LoginTourist 游客登录
func (mgr *UserMgr) LoginTourist() (user *User, token string, deadline int64, err error) {
	return mgr.LoginTouristWithFrom(fromDefault)
}

// LoginTouristWithFrom 游客登录
func (mgr *UserMgr) LoginTouristWithFrom(from string) (user *User, token string, deadline int64, err error) {
	user, err = mgr.RegisterTourist()
	if err != nil {
		return
	}
	if token, deadline, err = user.LoginWithFrom(from); err != nil {
		return nil, "", 0, err
	}
	return
}

// LoginLAPD 密码登录
func (mgr *UserMgr) LoginLAPD(uid, rawPassword string) (user *User, token string, deadline int64, err error) {
	return mgr.LoginLAPDWithFrom(uid, rawPassword, fromDefault)
}

// LoginLAPDWithFrom 密码登录 带来源
func (mgr *UserMgr) LoginLAPDWithFrom(uid, rawPassword, from string) (user *User, token string, deadline int64, err error) {
	var ok bool
	ok, user, err = mgr.FindUserByUID(uid)
	if err != nil {
		return
	}

	if !ok {
		user, err = mgr.RegisterLAPD(uid, rawPassword)
		if err != nil {
			return
		}
	}

	if token, deadline, err = user.LoginWithFrom(from); err != nil {
		return nil, "", 0, err
	}
	return
}

// LoginMobileApplyCode 手机验证码登录 申请验证码
func (mgr *UserMgr) LoginMobileApplyCode(mobile string) (code string, expire, retry int, err error) {
	return mgr.ApplyCodeAntiReplay(mobile, 0, 0, mobile)
}

// LoginMobile 手机验证码登录
func (mgr *UserMgr) LoginMobile(mobile, code string) (user *User, token string, deadline int64, err error) {
	return mgr.LoginMobileWithFrom(mobile, code, fromDefault)
}

// LoginMobileWithFrom 手机验证码登录 带来源
func (mgr *UserMgr) LoginMobileWithFrom(mobile, code, from string) (user *User, token string, deadline int64, err error) {
	var ok bool
	ok, err = mgr.VerifyCode(code, mobile)
	if err != nil {
		return
	}
	if !ok {
		err = fmt.Errorf("code is invalid")
		return
	}

	ok, user, err = mgr.FindUserByMobile(mobile)
	if err != nil {
		return
	}

	if !ok {
		user, err = mgr.registerMobile(mobile)
		if err != nil {
			return
		}
	}

	if token, deadline, err = user.LoginWithFrom(from); err != nil {
		return nil, "", 0, err
	}
	return
}

// LoginAuth 第三方登录
func (mgr *UserMgr) LoginAuth(authName string, v interface{}) (user *User, token string, deadline int64, err error) {
	return mgr.LoginAuthWithFrom(authName, v, fromDefault)
}

// LoginAuthWithFrom 第三方登录 带来源
func (mgr *UserMgr) LoginAuthWithFrom(authName string, v interface{}, from string) (user *User, token string, deadline int64, err error) {
	var authUID, authExtra string
	authUID, authExtra, err = mgr.VerifyAuth(authName, v)
	if err != nil {
		return
	}

	var ok bool
	ok, user, err = mgr.FindUserByAuth(authName, authUID)
	if err != nil {
		return
	}

	if !ok {
		user, err = mgr.registerAuth(authName, authUID, authExtra)
		if err != nil {
			return
		}
	}

	if token, deadline, err = user.LoginWithFrom(from); err != nil {
		return nil, "", 0, err
	}
	return
}
