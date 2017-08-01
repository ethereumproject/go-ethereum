package p2p

import (
	"github.com/ethereumproject/go-ethereum/logger"
	"sync"
)

var mlog *logger.Logger
var mlogOnce sync.Once

func initMLogging() {
	mlog = logger.NewLogger("p2p")
	mlog.Infoln("[mlog] ON")
}
