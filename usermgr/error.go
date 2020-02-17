package usermgr

import (
	"fmt"
)

// 常用错误
var (
	ErrorNotFound = fmt.Errorf("not found")
	ErrorLocked   = fmt.Errorf("locked")
)
