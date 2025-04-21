package logger

import (
	"fmt"
	"io"
	baselog "log"
	"os"
	"testing_system/common/config"
)

const (
	LogLevelTrace = 0
	LogLevelDebug = 1
	LogLevelInfo  = 2
	LogLevelWarn  = 3
	LogLevelError = 4
)

var (
	logLevel = LogLevelInfo
	log      = baselog.New(os.Stdout, "", baselog.Ldate|baselog.Ltime)
	logErr   = baselog.New(os.Stderr, "", baselog.Ldate|baselog.Ltime)
)

func GetLevel() int {
	return logLevel
}

func Trace(format string, values ...interface{}) {
	logPrint(LogLevelTrace, format, values...)
}

func Debug(format string, values ...any) {
	logPrint(LogLevelDebug, format, values...)
}

func Info(format string, values ...any) {
	logPrint(LogLevelInfo, format, values...)
}

func Warn(format string, values ...any) {
	logPrint(LogLevelWarn, format, values...)
}

func Error(format string, values ...any) error {
	logPrint(LogLevelError, format, values...)
	logErr.Printf(logErrPrefix(0)+format, values...)
	return fmt.Errorf(format, values...)
}

func Panic(format string, values ...any) {
	logPrint(LogLevelError, format, values...)
	logErr.Printf(logErrPrefix(0)+format, values...)
	panic(fmt.Errorf(format, values...))
}

type logWriter struct {
	level  int
	prefix string
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	logPrint(w.level, "%s %s", w.prefix, string(p))
	return len(p), nil
}

func CreateWriter(level int, prefix string) io.Writer {
	return &logWriter{
		level:  level,
		prefix: prefix,
	}
}

// PanicLevel will output to log the code line which is reachable by level call depth. May be applied in some library code
func PanicLevel(level int, format string, values ...any) {
	logPrint(LogLevelError, format, values...)
	logErr.Printf(logErrPrefix(level)+format, values...)
	panic(fmt.Errorf(format, values...))
}

func InitLogger(config *config.Config) {
	logLevel = LogLevelInfo
	if config.LogLevel != nil {
		logLevel = *config.LogLevel
	}

	var logFile, logErrFile *os.File

	if config.LogPath == nil {
		logFile = os.Stdout
		logErrFile = os.Stderr
	} else {
		var err error
		logFile, err = os.OpenFile(*config.LogPath+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
		if err != nil {
			panic(err)
		}

		logErrFile, err = os.OpenFile(*config.LogPath+".err", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
		if err != nil {
			panic(err)
		}
	}

	flags := baselog.Ldate | baselog.Ltime
	log = baselog.New(logFile, "", flags)
	logErr = baselog.New(logErrFile, "", flags)

	Info("Logger is successfully initialized")
}
