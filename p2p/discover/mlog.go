package discover

import (
	"github.com/ethereumproject/go-ethereum/logger"
	"sync"
)

var mlog *logger.Logger
var mlogOnce sync.Once

func initMLogging() {
	mlog = logger.NewLogger("discover")
	mlog.Infoln("[mlog] ON")
}
