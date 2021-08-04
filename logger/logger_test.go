// Copyright 2019 Hewlett Packard Enterprise Development LP
package logger

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func getLogFile() string {
	// get temp location for logging
	logDir := os.TempDir()
	logName := "test.log"
	return logDir + logName
}

func logAllLevels(testName string) {
	log.Tracef("%s:%s", testName, log.TraceLevel.String())
	log.Debugf("%s:%s", testName, log.DebugLevel.String())
	log.Infof("%s:%s", testName, log.InfoLevel.String())
	log.Errorf("%s:%s", testName, log.ErrorLevel.String())
	log.Warnf("%s:%s", testName, log.WarnLevel.String())
}

func testContains(t *testing.T, logFile string, testName string, level string, shouldContain bool) {
	b, err := ioutil.ReadFile(logFile)
	assert.Equal(t, err, nil)

	switch level {
	case log.TraceLevel.String():
		assert.Equal(t, shouldContain, strings.Contains(string(b), fmt.Sprintf("%s:%s", testName, log.TraceLevel.String())))
		if !shouldContain {
			break
		}
		fallthrough
	case log.DebugLevel.String():
		assert.Equal(t, shouldContain, strings.Contains(string(b), fmt.Sprintf("%s:%s", testName, log.DebugLevel.String())))
		if !shouldContain {
			break
		}
		fallthrough
	case log.InfoLevel.String():
		assert.Equal(t, shouldContain, strings.Contains(string(b), fmt.Sprintf("%s:%s", testName, log.InfoLevel.String())))
		if !shouldContain {
			break
		}
		fallthrough
	case log.WarnLevel.String():
		assert.Equal(t, shouldContain, strings.Contains(string(b), fmt.Sprintf("%s:%s", testName, log.WarnLevel.String())))
		if !shouldContain {
			break
		}
		fallthrough
	case log.ErrorLevel.String():
		assert.Equal(t, shouldContain, strings.Contains(string(b), fmt.Sprintf("%s:%s", testName, log.ErrorLevel.String())))
	}
}

func TestInitLogging(t *testing.T) {
	logFile := getLogFile()

	// cleanup log file before test
	os.RemoveAll(logFile)

	// Test1: test overrides with params to log to only stdout
	InitLogging("", nil, true, false)

	// verify logging with override to stdout only
	testName := "test_param_override_stdout_only"
	logAllLevels(testName)
	// test nothing is logged to file or file not created
	_, err := os.Stat(logFile)
	assert.Equal(t, true, os.IsNotExist(err))

	// Test 2: initialize logger with nil params to verify default levels
	InitLogging(logFile, nil, false, false)

	// verify default info level setting with no params
	assert.Equal(t, DefaultLogLevel, log.GetLevel().String())

	// verify logging with info level and below
	testName = "test_default_info_level"
	logAllLevels(testName)
	testContains(t, logFile, testName, "info", true)
	testContains(t, logFile, testName, "warn", true)
	testContains(t, logFile, testName, "error", true)
	testContains(t, logFile, testName, "trace", false)
	testContains(t, logFile, testName, "debug", false)

	// Test3: initialize logger with override of trace level
	InitLogging(logFile, &LogParams{Level: "trace"}, false, false)

	// verify trace level setting with param override
	assert.Equal(t, log.TraceLevel.String(), log.GetLevel().String())

	// verify logging with trace level and below
	testName = "test_param_override_trace_level"
	logAllLevels(testName)
	testContains(t, logFile, testName, "info", true)
	testContains(t, logFile, testName, "warn", true)
	testContains(t, logFile, testName, "error", true)
	testContains(t, logFile, testName, "trace", true)
	testContains(t, logFile, testName, "debug", true)

	// Test4: initialize logger with env vars for info level
	os.Setenv("LOG_LEVEL", "debug")
	InitLogging(logFile, nil, false, false)
	// verify logging with debug level and below
	testName = "test_env_debug_level"
	logAllLevels(testName)
	testContains(t, logFile, testName, "info", true)
	testContains(t, logFile, testName, "warn", true)
	testContains(t, logFile, testName, "error", true)
	testContains(t, logFile, testName, "debug", true)
	testContains(t, logFile, testName, "trace", false)

	// Test5: initialize logger with invalid log format through env
	os.Setenv("LOG_FORMAT", "yaml")
	InitLogging(logFile, nil, false, false)

	// verify log format is set to default value of text
	assert.Equal(t, logParams.GetLogFormat(), DefaultLogFormat)

	// Test6: initialize logger with invalid log files limit through config
	InitLogging(logFile, &LogParams{MaxFiles: 1000}, false, false)

	// verify log files is set to default value of 10
	assert.Equal(t, logParams.GetMaxFiles(), DefaultMaxLogFiles)

	// Test7: test overrides with env variables even when params is not nil
	os.Setenv("LOG_LEVEL", "info")
	InitLogging(logFile, &LogParams{Level: "trace"}, false, false)

	// verify logging with only info level and below with override from env
	testName = "test_env_override_info_level"
	logAllLevels(testName)
	testContains(t, logFile, testName, "info", true)
	testContains(t, logFile, testName, "warn", true)
	testContains(t, logFile, testName, "error", true)
	testContains(t, logFile, testName, "debug", false)
	testContains(t, logFile, testName, "trace", false)

	// cleanup log file after test
	os.RemoveAll(logFile)
}

func TestInitJaeger(t *testing.T) {
	_, lg := InitLogging("test.log", nil, true, true)

	lg.Info("************** Start Workflow 1 **************")
	lg.Info("********** Workflow 1 Line 1 **********")
	s := lg.StartContext("Workflow 2")
	lg.Info("**************** Start Workflow 2 *****************")
	lg.Info("********** Workflow 2 Line 1 ******************")
	EndContext(s)
	sp := lg.StartContext("Workflow 2")
	lg.Info("**************** Start Workflow 2 *****************")
	lg.Info("********** Workflow 2 Line 1 ******************")
	EndContext(sp)
	sp2 := lg.StartContext("Workflow 3")
	lg.Info("**************** Start Workflow 3 *****************")
	lg.Info("********** Workflow 3 Line 1 ******************")
	EndContext(sp2)
	lg.CloseTracer()

}
