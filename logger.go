package gouser

import (
	"github.com/cheetah-fun-gs/goplus/logger"
	mlogger "github.com/cheetah-fun-gs/goplus/multier/multilogger"
)

// Logger 全局日志
var (
	MLoggerName = "default" // 使用日志管理工具
)

func init() {
	if _, err := mlogger.RetrieveN(MLoggerName); err != nil {
		mlogger.Register(MLoggerName, logger.New())
	}
}

// SetLogger ...
func SetLogger(name string) {
	MLoggerName = name
}
