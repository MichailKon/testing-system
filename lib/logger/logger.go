package logger

import (
	"fmt"
	base_log "log"
	"os"
	"testing_system/lib/config"
)

const (
	LOG_LEVEL_TRACE = 0
	LOG_LEVEL_DEBUG = 1
	LOG_LEVEL_INFO  = 2
	LOG_LEVEL_WARN  = 3
	LOG_LEVEL_ERROR = 4
)

var (
	logLevel = LOG_LEVEL_DEBUG
	log      = base_log.New(os.Stdout, "", base_log.Ldate|base_log.Ltime)
	logErr   = base_log.New(os.Stderr, "", base_log.Ldate|base_log.Ltime)
)

func Trace(format string, values ...interface{}) {
	logPrint(LOG_LEVEL_TRACE, format, values...)
}

func Debug(format string, values ...any) {
	logPrint(LOG_LEVEL_DEBUG, format, values...)
}

func Info(format string, values ...any) {
	logPrint(LOG_LEVEL_INFO, format, values...)
}

func Warn(format string, values ...any) {
	logPrint(LOG_LEVEL_WARN, format, values...)
}

func Error(format string, values ...any) error {
	logPrint(LOG_LEVEL_ERROR, format, values...)
	logErr.Printf(logErrPrefix()+format, values...)
	return fmt.Errorf(format, values...)
}

func Panic(format string, value ...any) {
	logPrint(LOG_LEVEL_ERROR, format, value...)
	logErr.Panicf(format, value...)
}

func InitLogger(config *config.Config) {
	logLevel = LOG_LEVEL_INFO
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

	flags := base_log.Ldate | base_log.Ltime
	log = base_log.New(logFile, "", flags)
	logErr = base_log.New(logErrFile, "", flags)

	Info("Logger is successfully initialized")
}
