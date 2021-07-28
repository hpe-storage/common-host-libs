package main

import (
	"github.com/hpe-storage/common-host-libs/logger"
)

func main() {
	_, lg := logger.InitLogging("test.log", nil, true, true)
	lg.Info("**********************************************")
	lg.Info("*************** HPE CSI DRIVER ***************")
	lg.Info("**********************************************")

	/*logger.InitLogging("test.log", nil, true, true)
	logger.Info("**********************************************")
	logger.Info("*************** HPE CSI DRIVER ***************")
	logger.Info("**********************************************")
	sp, ctx2 := opentracing.StartSpanFromContext("Testing CSI-Driver", ctx)
	sp.LogEvent("say-hello")
	sp.Finish() . */
}
