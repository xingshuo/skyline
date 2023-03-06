package log

import (
	"log"
	"os"
)

var gLogger Logger
var gLogLevel LogLevel = LevelDebug

func Init(filename string, level LogLevel) {
	if filename != "" {
		file, err := os.Create(filename)
		if err != nil {
			log.Fatalf("log Init err: %v", err)
		}
		l := log.New(file, "", log.Ldate|log.Ltime)
		gLogger = &FileLogger{
			logger: l,
			file:   file,
		}
	} else {
		gLogger = &StdLogger{}
	}
	gLogLevel = level
}

func Close() {
	gLogger.Close()
}

func SetLogLevel(level LogLevel) {
	gLogLevel = level
}

func Debug(args ...interface{}) {
	if gLogLevel > LevelDebug {
		return
	}
	gLogger.Log(LevelDebug, args...)
}

func Debugf(format string, args ...interface{}) {
	if gLogLevel > LevelDebug {
		return
	}
	gLogger.Logf(LevelDebug, format, args...)
}

func Info(args ...interface{}) {
	if gLogLevel > LevelInfo {
		return
	}
	gLogger.Log(LevelInfo, args...)
}

func Infof(format string, args ...interface{}) {
	if gLogLevel > LevelInfo {
		return
	}
	gLogger.Logf(LevelInfo, format, args...)
}

func Warning(args ...interface{}) {
	if gLogLevel > LevelWarning {
		return
	}
	gLogger.Log(LevelWarning, args...)
}

func Warningf(format string, args ...interface{}) {
	if gLogLevel > LevelWarning {
		return
	}
	gLogger.Logf(LevelWarning, format, args...)
}

func Error(args ...interface{}) {
	if gLogLevel > LevelError {
		return
	}
	gLogger.Log(LevelError, args...)
}

func Errorf(format string, args ...interface{}) {
	if gLogLevel > LevelError {
		return
	}
	gLogger.Logf(LevelError, format, args...)
}

func Fatal(args ...interface{}) {
	gLogger.Log(LevelFatal, args...)
	os.Exit(1)
}

func Fatalf(format string, args ...interface{}) {
	gLogger.Logf(LevelFatal, format, args...)
	os.Exit(1)
}
