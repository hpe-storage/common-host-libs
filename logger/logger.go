// Copyright 2020 Hewlett Packard Enterprise Development LP

package logger

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	otLog "github.com/opentracing/opentracing-go/log"
	log "github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go/config"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DefaultLogLevel    = "info"
	DefaultLogFormat   = TextFormat
	DefaultMaxLogFiles = 10
	MaxFilesLimit      = 20
	DefaultMaxLogSize  = 100  // in MB
	MaxLogSizeLimit    = 1024 // in MB
	JSONFormat         = "json"
	TextFormat         = "text"
)

// LogParams to configure logging
type LogParams struct {
	Level      string
	File       string
	MaxFiles   int
	MaxSizeMiB int
	Format     string
}

type Logr struct {
	ctx      context.Context
	logEntry *log.Entry
	cl       io.Closer
}

var (
	logParams LogParams
	initMutex sync.Mutex
)

func (l LogParams) isValidLevel() bool {
	switch l.Level {
	case "trace":
		fallthrough
	case "debug":
		fallthrough
	case "info":
		fallthrough
	case "warn":
		fallthrough
	case "error":
		return true
	default:
		return false
	}
}

func (l LogParams) isValidLogFormat() bool {
	switch l.Format {
	case "json":
		fallthrough
	case "text":
		return true
	default:
		return false
	}
}

func (l LogParams) isValidMaxLogFiles() bool {
	if l.MaxFiles == 0 || l.MaxFiles > MaxFilesLimit {
		return false
	}
	return true
}

func (l LogParams) isValidMaxLogSize() bool {
	if l.MaxSizeMiB == 0 || l.MaxSizeMiB > MaxLogSizeLimit {
		return false
	}
	return true
}

func (l LogParams) GetLevel() string {
	if !l.isValidLevel() {
		return DefaultLogLevel
	}
	return l.Level
}

func (l LogParams) GetFile() string {
	return l.File
}

func (l LogParams) GetMaxFiles() int {
	if !l.isValidMaxLogFiles() {
		return DefaultMaxLogFiles
	}
	return l.MaxFiles
}

func (l LogParams) GetMaxSize() int {
	if !l.isValidMaxLogSize() {
		return DefaultMaxLogSize
	}
	return l.MaxSizeMiB
}

func (l LogParams) GetLogFormat() string {
	if !l.isValidLogFormat() {
		return DefaultLogFormat
	}
	return l.Format
}

func (l LogParams) UseJsonFormatter() bool {
	return l.Format == JSONFormat
}

func (l LogParams) UseTextFormatter() bool {
	return l.Format == TextFormat
}

type Fields = log.Fields

func updateLogParamsFromEnv() {
	level := os.Getenv("LOG_LEVEL")
	if level != "" {
		logParams.Level = level
	}

	logFile := os.Getenv("LOG_FILE")
	if logFile != "" {
		logParams.File = logFile
	}

	maxSize := os.Getenv("LOG_MAX_SIZE")
	if maxSize != "" {
		size, err := strconv.ParseInt(maxSize, 0, 0)
		if err == nil {
			logParams.MaxSizeMiB = int(size)
		}
	}

	maxFiles := os.Getenv("LOG_MAX_FILES")
	if maxFiles != "" {
		fileCount, err := strconv.ParseInt(maxFiles, 0, 0)
		if err == nil {
			logParams.MaxFiles = int(fileCount)
		}
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat != "" {
		logParams.Format = logFormat
	}
}

//Initalizes opentracing tracing
func InitOpentracing(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		ServiceName: service,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}
	//add tracer as a input of NewTracer so that the logspans declared true above will work
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init tracing: %v\n", err))
	}
	return tracer, closer
}

// Initialize logging with given params
func InitLogging(logName string, params *LogParams, alsoLogToStderr bool, initTracing bool) (err error, l *Logr) {
	initMutex.Lock()
	defer initMutex.Unlock()

	// if logParams is not provided, then initialize from defaults and command line parameters
	if params == nil {
		// Initialize defaults.
		logParams.Level = DefaultLogLevel
		logParams.MaxSizeMiB = DefaultMaxLogSize
		logParams.MaxFiles = DefaultMaxLogFiles
		logParams.Format = DefaultLogFormat
	} else {
		logParams = *params
	}

	// verify if log location is provided
	if logName != "" {
		logParams.File = logName
	}

	// check any overrides from env and apply
	updateLogParamsFromEnv()

	// No output except for the hooks
	log.SetOutput(ioutil.Discard)

	//Default Logr
	logEntry := sourced()
	lg := Logr{nil, logEntry, nil}

	if logParams.GetFile() != "" {
		err = AddFileHook()
		if err != nil {
			return err, &lg
		}
	}
	if alsoLogToStderr {
		err = AddConsoleHook()
		if err != nil {
			return err, &lg
		}
	}

	// Set log level
	level, err := log.ParseLevel(logParams.GetLevel())
	if err != nil {
		return err, &lg
	}
	log.SetLevel(level)

	// Remind users where the log file lives
	log.WithFields(log.Fields{
		"logLevel":        log.GetLevel().String(),
		"logFileLocation": logParams.GetFile(),
		"alsoLogToStderr": alsoLogToStderr,
	}).Info("Initialized logging.")

	//initializes tracing capabilites if true
	if initTracing {
		//Initializing the tracer
		tracer, closer := InitOpentracing("CSI-Driver")
		opentracing.SetGlobalTracer(tracer)

		//Span Initialized with default context
		span := tracer.StartSpan("CSI-Driver")
		log.Tracef("Span Context --- Traceid:Spanid:ParentSpanid:Flags  : %v", span.Context())
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		logEntry := sourced()
		l := Logr{ctx, logEntry, closer}

		l.LogToTrace("Info", "Tracing Initialized")
		defer span.Finish()

		return nil, &l
	}

	return nil, &lg
}

func (l *Logr) CloseTracer() {
	l.cl.Close()
}

//Logs given string to tracer
func (l *Logr) LogToTrace(level, msg string) {
	span := opentracing.SpanFromContext(l.ctx)
	//fmt.Print("In LogToTrace")
	if span != nil {
		span.LogFields(otLog.String("event", msg))
	}
	if span == nil {
		fmt.Print("Span is nil")
	}
	span.Finish()
}

//Sets context of called Logr to given context
func (l *Logr) SetContext(context context.Context) {
	l.ctx = context
}

//Starts and returns a span for the inputted Logr
func (l *Logr) StartContext(spanName string) (s opentracing.Span) {
	s = opentracing.SpanFromContext(l.ctx)
	if s == nil || s.BaggageItem(spanName) == "" {
		s = opentracing.StartSpan(spanName)
		s.SetBaggageItem(spanName, "true")
		l.ctx = opentracing.ContextWithSpan(context.Background(), s)
	}
	return s
}

//Ends the inputted span
func EndContext(span opentracing.Span) {
	span.Finish()
}

func AddConsoleHook() error {
	// Write to stdout/stderr
	log.AddHook(NewConsoleHook())
	return nil
}

func AddFileHook() error {
	// Write to the log file
	logFileHook, err := NewFileHook()
	if err != nil {
		return fmt.Errorf("could not initialize logging to file %s: %v", logFileHook.GetLocation(), err)
	}
	log.AddHook(logFileHook)
	return nil
}

// ConsoleHook sends log entries to stdout.
type ConsoleHook struct {
	formatter log.Formatter
}

// NewConsoleHook creates a new log hook for writing to stdout/stderr.
func NewConsoleHook() *ConsoleHook {
	if logParams.UseJsonFormatter() {
		return &ConsoleHook{&log.JSONFormatter{CallerPrettyfier: CustomCallerPrettyfier}}
	}
	return &ConsoleHook{&log.TextFormatter{FullTimestamp: true, CallerPrettyfier: CustomCallerPrettyfier}}
}

func (hook *ConsoleHook) Levels() []log.Level {
	return log.AllLevels
}

func (hook *ConsoleHook) checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func (hook *ConsoleHook) Fire(entry *log.Entry) error {
	// Determine output stream
	var logWriter io.Writer
	switch entry.Level {
	case log.DebugLevel, log.InfoLevel, log.WarnLevel, log.TraceLevel:
		logWriter = os.Stdout
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		logWriter = os.Stderr
	}

	// Write log entry to output stream
	if logParams.UseTextFormatter() {
		//https://github.com/sirupsen/logrus/issues/172
		if runtime.GOOS != "windows" {
			hook.formatter.(*log.TextFormatter).ForceColors = hook.checkIfTerminal(logWriter)
		}
	}

	lineBytes, err := hook.formatter.Format(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	logWriter.Write(lineBytes)
	return nil
}

// FileHook sends log entries to a file.
type FileHook struct {
	formatter log.Formatter
	mutex     *sync.Mutex
	logWriter io.Writer
}

func CustomCallerPrettyfier(f *runtime.Frame) (string, string) {
	s := strings.Split(f.Function, ".")
	funcname := s[len(s)-1]
	_, filename := path.Split(f.File)
	return funcname, filename
}

// NewFileHook creates a new log hook for writing to a file.
func NewFileHook() (hook *FileHook, err error) {

	if logParams.UseJsonFormatter() {
		hook = &FileHook{&log.JSONFormatter{}, &sync.Mutex{}, nil}
	} else {
		hook = &FileHook{&log.TextFormatter{FullTimestamp: true}, &sync.Mutex{}, nil}
	}

	// use lumberjack for log rotation
	hook.logWriter = &lumberjack.Logger{
		Filename:   logParams.GetFile(),
		MaxSize:    logParams.GetMaxSize(),
		MaxBackups: logParams.GetMaxFiles(),
		MaxAge:     30,
		Compress:   true,
	}
	return hook, nil
}

func (hook *FileHook) Levels() []log.Level {
	return log.AllLevels
}

func (hook *FileHook) Fire(entry *log.Entry) error {
	// Get formatted entry
	lineBytes, err := hook.formatter.Format(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read log entry. %v", err)
		return err
	}

	// For Windows only, insert '/r' in front of any tailing '/n'.  Windows text files end
	// lines with CRLF while other platforms just end with LF.
	if runtime.GOOS == "windows" {
		for i := len(lineBytes) - 1; i > 0; i-- {
			if (lineBytes[i] != '\n') || (i > 0 && lineBytes[i-1] == '\r') {
				break
			}
			lineBytes = append(lineBytes[:i], append([]byte{'\r'}, lineBytes[i:]...)...)
		}
	}

	hook.logWriter.Write(lineBytes)
	return nil
}

func (hook *FileHook) GetLocation() string {
	return logParams.GetFile()
}

// GetLevel returns the standard logger level.
func GetLevel() log.Level {
	return log.GetLevel()
}

// IsLevelEnabled checks if the log level of the standard logger is greater than the level param
func IsLevelEnabled(level log.Level) bool {
	return log.IsLevelEnabled(level)
}

// AddHook adds a hook to the standard logger hooks.
func AddHook(hook log.Hook) {
	log.AddHook(hook)
}

// WithError creates an entry from the standard logger and adds an error to it, using the value defined in ErrorKey as key.
func WithError(err error) *log.Entry {
	return log.WithField(log.ErrorKey, err)
}

// WithContext creates an entry from the standard logger and adds a context to it.
func WithContext(ctx context.Context) *log.Entry {
	return log.WithContext(ctx)
}

// WithField creates an entry from the standard logger and adds a field to
// it. If you want multiple fields, use `WithFields`.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithField(key string, value interface{}) *log.Entry {
	return log.WithField(key, value)
}

// WithFields creates an entry from the standard logger and adds multiple
// fields to it. This is simply a helper for `WithField`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithFields(fields Fields) *log.Entry {
	return log.WithFields(fields)
}

// WithTime creats an entry from the standard logger and overrides the time of
// logs generated with it.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithTime(t time.Time) *log.Entry {
	return log.WithTime(t)
}

// HTTPLogger : wrapper for http logging
func HTTPLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panicked := true
		defer func() {
			if panicked {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				sourced().Errorf("HTTPLogger: panic serving %v:\n%s", name, buf)
			}
		}()

		sourced().Infof(
			">>>>> %s %s - %s",
			r.Method,
			r.RequestURI,
			name,
		)

		start := time.Now()
		inner.ServeHTTP(w, r)

		sourced().Infof(
			"<<<<< %s %s - %s %s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)

		panicked = false
	})
}

// IsSensitive checks if the given key exists in the list of bad words (sensitive info)
func IsSensitive(key string) bool {
	// TODO: Add more sensitive words (lower-case) to this list
	badWords := []string{
		"x-auth-token",
		"username",
		"user",
		"password",
		"passwd",
		"secret",
		"token",
		"accesskey",
		"passphrase",
	}
	key = strings.ToLower(key)
	for _, bad := range badWords {
		// Perform case-insensitive and substring match
		if strings.Contains(key, bad) {
			return true
		}
	}
	return false
}

// Scrubber checks if the args list contains any sensitive information like username/password/secret
// If found, then returns masked string list, else returns the original input list unmodified.
func Scrubber(args []string) []string {
	for _, arg := range args {
		if IsSensitive(arg) {
			return []string{"**********"}
		}
	}
	return args
}

// MapScrubber checks if the map contains any sensitive information like username/password/secret
// If found, then masks values for those keys, else copies the original value and returns new map
func MapScrubber(m map[string]string) map[string]string {
	retMap := make(map[string]string)
	for k, v := range m {
		if IsSensitive(k) {
			retMap[k] = "**********"
		} else {
			retMap[k] = v
		}
	}
	return retMap
}

// sourced adds a source field to the logger that contains
// the file name and line where the logging happened.
func sourced() *log.Entry {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	return log.WithField("file", fmt.Sprintf("%s:%d", file, line))
}

func Trace(args ...interface{}) {
	sourced().Trace(args...)
}

// Trace logs a message at level Trace on the standard logger.
func (l *Logr) Trace(args ...interface{}) {
	l.logEntry.Trace(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Trace", str)
}

func Debug(args ...interface{}) {
	sourced().Trace(args...)
}

// Debug logs a message at level Debug on the standard logger.
func (l *Logr) Debug(args ...interface{}) {
	l.logEntry.Debug(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Debug", str)
}

// Print logs a message at level Info on the standard logger.
func Print(args ...interface{}) {
	sourced().Print(args...)
}

func (l *Logr) Print(args ...interface{}) {
	l.logEntry.Print(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Print", str)
}

func Info(args ...interface{}) {
	sourced().Trace(args...)
}

// Info logs a message at level Info on the standard logger.
func (l *Logr) Info(args ...interface{}) {
	l.logEntry.Info(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Info", str)
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
	sourced().Warn(args...)
}

func (l *Logr) Warn(args ...interface{}) {
	l.logEntry.Warn(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Warn", str)
}

// Warning logs a message at level Warn on the standard logger.
func Warning(args ...interface{}) {
	sourced().Warning(args...)
}

func (l *Logr) Warning(args ...interface{}) {
	l.logEntry.Warning(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Warning", str)
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
	sourced().Error(args...)
}

func (l *Logr) Error(args ...interface{}) {
	l.logEntry.Error(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Error", str)
}

// Panic logs a message at level Panic on the standard logger.
func Panic(args ...interface{}) {
	sourced().Panic(args...)
}

func (l *Logr) Panic(args ...interface{}) {
	l.logEntry.Panic(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Panic", str)
}

// Fatal logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatal(args ...interface{}) {
	sourced().Fatal(args...)
}
func (l *Logr) Fatal(args ...interface{}) {
	l.logEntry.Fatal(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Fatal", str)
}

// Tracef logs a message at level Trace on the standard logger.
func Tracef(format string, args ...interface{}) {
	sourced().Tracef(format, args...)
}

func (l *Logr) Tracef(format string, args ...interface{}) {
	l.logEntry.Tracef(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Tracef", str)
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	sourced().Debugf(format, args...)
}

func (l *Logr) Debugf(format string, args ...interface{}) {
	l.logEntry.Debugf(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Debugf", str)
}

// Printf logs a message at level Info on the standard logger.
func Printf(format string, args ...interface{}) {
	sourced().Printf(format, args...)
}

func (l *Logr) Printf(format string, args ...interface{}) {
	l.logEntry.Printf(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Printf", str)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	sourced().Infof(format, args...)
}

func (l *Logr) Infof(format string, args ...interface{}) {
	l.logEntry.Infof(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Infof", str)
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	sourced().Warnf(format, args...)
}

func (l *Logr) Warnf(format string, args ...interface{}) {
	l.logEntry.Warnf(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Warnf", str)
}

// Warningf logs a message at level Warn on the standard logger.
func Warningf(format string, args ...interface{}) {
	sourced().Warningf(format, args...)
}

func (l *Logr) Warningf(format string, args ...interface{}) {
	l.logEntry.Warningf(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Warningf", str)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	sourced().Errorf(format, args...)
}

func (l *Logr) Errorf(format string, args ...interface{}) {
	l.logEntry.Errorf(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Errorf", str)
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
	sourced().Panicf(format, args...)
}

func (l *Logr) Panicf(format string, args ...interface{}) {
	l.logEntry.Panicf(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Panicf", str)
}

// Fatalf logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatalf(format string, args ...interface{}) {
	sourced().Fatalf(format, args...)
}

func (l *Logr) Fatalf(format string, args ...interface{}) {
	l.logEntry.Fatalf(format, args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Fatalf", str)
}

// Traceln logs a message at level Trace on the standard logger.
func Traceln(args ...interface{}) {
	sourced().Traceln(args...)
}

func (l *Logr) Traceln(args ...interface{}) {
	l.logEntry.Traceln(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Traceln", str)
}

// Debugln logs a message at level Debug on the standard logger.
func Debugln(args ...interface{}) {
	sourced().Debugln(args...)
}

func (l *Logr) Debugln(args ...interface{}) {
	l.logEntry.Debugln(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Debugln", str)
}

// Println logs a message at level Info on the standard logger.
func Println(args ...interface{}) {
	sourced().Println(args...)
}

func (l *Logr) Println(args ...interface{}) {
	l.logEntry.Println(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Println", str)
}

// Infoln logs a message at level Info on the standard logger.
func Infoln(args ...interface{}) {
	sourced().Infoln(args...)
}

func (l *Logr) Infoln(args ...interface{}) {
	l.logEntry.Infoln(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Infoln", str)
}

// Warnln logs a message at level Warn on the standard logger.
func Warnln(args ...interface{}) {
	sourced().Warnln(args...)
}

func (l *Logr) Warnln(args ...interface{}) {
	l.logEntry.Warnln(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Warnln", str)
}

// Warningln logs a message at level Warn on the standard logger.
func Warningln(args ...interface{}) {
	sourced().Warningln(args...)
}

func (l *Logr) Warningln(args ...interface{}) {
	l.logEntry.Warningln(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Warningln", str)
}

// Errorln logs a message at level Error on the standard logger.
func Errorln(args ...interface{}) {
	sourced().Errorln(args...)
}

func (l *Logr) Errorln(args ...interface{}) {
	l.logEntry.Errorln(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Errorln", str)
}

// Panicln logs a message at level Panic on the standard logger.
func Panicln(args ...interface{}) {
	sourced().Panicln(args...)
}

func (l *Logr) Panicln(args ...interface{}) {
	l.logEntry.Panicln(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Panicln", str)
}

// Fatalln logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
func Fatalln(args ...interface{}) {
	sourced().Fatalln(args...)
}

func (l *Logr) Fatalln(args ...interface{}) {
	l.logEntry.Fatalln(args...)
	str := fmt.Sprintf("%v", args)
	l.LogToTrace("Fatalln", str)
}
