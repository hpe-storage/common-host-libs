package main

import (
	"github.com/hpe-storage/common-host-libs/logger"
)

func main() {
	logger.InitLogging("test.log", nil, true, true)
	logger.Info("**********************************************")
	logger.Info("*************** HPE CSI DRIVER ***************")
	logger.Info("**********************************************")

	/*logger.InitLogging("test.log", nil, true, true)
	logger.Info("**********************************************")
	logger.Info("*************** HPE CSI DRIVER ***************")
	logger.Info("**********************************************")
	sp, ctx2 := opentracing.StartSpanFromContext("Testing CSI-Driver", ctx)
	sp.LogEvent("say-hello")
	sp.Finish() . */
}
