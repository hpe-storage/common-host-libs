package main

import (
	"github.com/hpe-storage/common-host-libs/logger"
)

func main() {
	_, lg := logger.InitLogging("test.log", nil, true, true)

	lg.Info("************** Start Workflow 1 **************")
	lg.Info("********** Workflow 1 Line 1 **********")
	s := lg.StartContext()
	lg.Info("**************** Start Workflow 2 *****************")
	lg.Info("********** Workflow 2 Line 1 ******************")
	logger.EndContext(s)
	lg.CloseTracer()

	/*logger.InitLogging("test.log", nil, true, true)
	logger.Info("**********************************************")
	logger.Info("*************** HPE CSI DRIVER ***************")
	logger.Info("**********************************************")
	sp, ctx2 := opentracing.StartSpanFromContext("Testing CSI-Driver", ctx)
	sp.LogEvent("say-hello")
	sp.Finish() . */
}
