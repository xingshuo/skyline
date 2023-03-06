package log

import (
	"fmt"
	"log"
	"os"
)

type Logger interface {
	Log(lv LogLevel, args ...interface{})
	Logf(lv LogLevel, format string, args ...interface{})
	Close()
}

type StdLogger struct {
}

func (l *StdLogger) Log(lv LogLevel, args ...interface{}) {
	_ = log.Output(2, fmt.Sprintf("[%s]", lv)+fmt.Sprint(args...)+"\n")
}

func (l *StdLogger) Logf(lv LogLevel, format string, args ...interface{}) {
	_ = log.Output(2, fmt.Sprintf("[%s]", lv)+fmt.Sprintf(format, args...)+"\n")
}

func (l *StdLogger) Close() {
}

type FileLogger struct {
	logger *log.Logger
	file   *os.File
}

func (l *FileLogger) Log(lv LogLevel, args ...interface{}) {
	_ = l.logger.Output(2, fmt.Sprintf("[%s]", lv)+fmt.Sprint(args...)+"\n")
}

func (l *FileLogger) Logf(lv LogLevel, format string, args ...interface{}) {
	_ = l.logger.Output(2, fmt.Sprintf("[%s]", lv)+fmt.Sprintf(format, args...)+"\n")
}

func (l *FileLogger) Close() {
	_ = l.file.Close()
}
