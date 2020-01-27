package authmgr

// AuthMgr 第三方认证
type AuthMgr interface {
	GetName() string                                     // 认证名称
	Verify(v interface{}) (uid, extra string, err error) // 验证是否通过
}
