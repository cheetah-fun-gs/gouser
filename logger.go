package gouser

import (
	"github.com/cheetah-fun-gs/goplus/logger"
	mlogger "github.com/cheetah-fun-gs/goplus/multier/multilogger"
)

func init() {
	n := "default"
	if _, err := mlogger.RetrieveN(n); err != nil {
		mlogger.Register(n, logger.New())
	}
}
