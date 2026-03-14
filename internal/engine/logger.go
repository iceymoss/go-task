package engine

import (
	"log"
	"os"
)

// Logger 外部调用者只要实现了这个接口，就可以把他们自己的日志框架（Zap, Logrus, Slog等）无缝接入引擎
type Logger interface {
	Info(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Debug(msg string, keysAndValues ...any)
}

// defaultLogger 是一个极其简陋的兜底实现，使用标准库 log
type defaultLogger struct {
	l *log.Logger
}

func newDefaultLogger() Logger {
	return &defaultLogger{
		l: log.New(os.Stdout, "[go-task/engine] ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (d *defaultLogger) Info(msg string, kvs ...any)  { d.l.Printf("INFO: %s %v", msg, kvs) }
func (d *defaultLogger) Error(msg string, kvs ...any) { d.l.Printf("ERROR: %s %v", msg, kvs) }
func (d *defaultLogger) Warn(msg string, kvs ...any)  { d.l.Printf("WARN: %s %v", msg, kvs) }
func (d *defaultLogger) Debug(msg string, kvs ...any) { d.l.Printf("DEBUG: %s %v", msg, kvs) }
