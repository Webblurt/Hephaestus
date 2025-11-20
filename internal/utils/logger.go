package utils

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	SUCCESS
)

var levelNames = map[LogLevel]string{
	TRACE:   "TRACE",
	DEBUG:   "DEBUG",
	INFO:    "INFO",
	WARN:    "WARN",
	ERROR:   "ERROR",
	FATAL:   "FATAL",
	SUCCESS: "SUCCESS",
}

var levelColors = map[LogLevel]string{
	TRACE:   "\033[36m", // cyan
	DEBUG:   "\033[32m", // green
	INFO:    "\033[34m", // blue
	WARN:    "\033[33m", // yellow
	ERROR:   "\033[31m", // red
	FATAL:   "\033[31m", // red
	SUCCESS: "\033[35m", // magenta
}

const resetColor = "\033[0m"

type Logger struct {
	level LogLevel
	*log.Logger
}

func NewLogger(level string) *Logger {
	return &Logger{
		level:  parseLevel(level),
		Logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}
}

func parseLevel(s string) LogLevel {
	switch strings.ToLower(s) {
	case "trace":
		return TRACE
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	case "success":
		return SUCCESS
	default:
		return INFO
	}
}

func (l *Logger) canLog(level LogLevel) bool {
	return level >= l.level
}

func formatArgs(v ...interface{}) (string, []interface{}) {
	if len(v) == 0 {
		return "", nil
	}

	if s, ok := v[0].(string); ok && strings.Contains(s, "%") {
		return s, v[1:]
	}

	format := strings.Repeat("%v ", len(v))
	return strings.TrimSpace(format), v
}

func (l *Logger) logf(level LogLevel, v ...interface{}) {
	if !l.canLog(level) {
		return
	}

	format, args := formatArgs(v...)
	color := levelColors[level]
	prefix := fmt.Sprintf("%s[%s]%s ", color, levelNames[level], resetColor)

	_, file, line, ok := runtime.Caller(2)
	if ok {
		short := file[strings.LastIndex(file, "/")+1:]
		prefix += fmt.Sprintf("%s:%d: ", short, line)
	}

	message := fmt.Sprintf(format, args...)
	l.Output(3, prefix+message)

	if level == FATAL {
		os.Exit(1)
	}
}

func (l *Logger) Trace(v ...interface{})   { l.logf(TRACE, v...) }
func (l *Logger) Debug(v ...interface{})   { l.logf(DEBUG, v...) }
func (l *Logger) Info(v ...interface{})    { l.logf(INFO, v...) }
func (l *Logger) Warn(v ...interface{})    { l.logf(WARN, v...) }
func (l *Logger) Error(v ...interface{})   { l.logf(ERROR, v...) }
func (l *Logger) Fatal(v ...interface{})   { l.logf(FATAL, v...) }
func (l *Logger) Success(v ...interface{}) { l.logf(SUCCESS, v...) }
