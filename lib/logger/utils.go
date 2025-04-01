package logger

import (
	"fmt"
	"runtime"
)

func levelString(level int) string {
	switch level {
	case LogLevelTrace:
		return "[TRACE]"
	case LogLevelDebug:
		return "[DEBUG]"
	case LogLevelInfo:
		return "[INFO]"
	case LogLevelWarn:
		return "[WARN]"
	case LogLevelError:
		return "[ERROR]"
	default:
		return ""
	}
}

func logPrint(level int, format string, value ...any) {
	if logLevel <= level {
		log.Printf(levelString(level)+" "+format, value...)
	}
}

func logErrPrefix(level int) string {
	_, file, line, ok := runtime.Caller(level + 2)
	if ok {
		return fmt.Sprintf("%s:%d ", file, line)
	} else {
		return ""
	}
}
