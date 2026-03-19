package engine

import (
	"fmt"
	"log"
	"os"
)

// Logger 它同时支持“结构化键值对”和“传统占位符格式化”两种主流输出方式，
// 外部调用者可通过适配器无缝接入 Zap, Logrus, Slog 等任意框架。
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Fatal(msg string, keysAndValues ...any)

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)

	// With 附加固定的键值对到日志上下文中，返回一个新的 Logger
	With(keysAndValues ...any) Logger
}

type defaultLogger struct {
	l    *log.Logger
	args []any // 存放通过 With 继承下来的公共键值对
}

// NewDefaultLogger 创建默认的控制台日志器
func NewDefaultLogger() Logger {
	return &defaultLogger{
		l: log.New(os.Stdout, "[go-task/engine] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// 辅助方法：合并子 Logger 的公共字段和当前方法传入的字段
func (d *defaultLogger) mergeArgs(kvs ...any) []any {
	if len(d.args) == 0 {
		return kvs
	}
	res := make([]any, 0, len(d.args)+len(kvs))
	res = append(res, d.args...)
	res = append(res, kvs...)
	return res
}

func (d *defaultLogger) Debug(msg string, kvs ...any) {
	d.l.Printf("[DEBUG] %s %v", msg, d.mergeArgs(kvs...))
}
func (d *defaultLogger) Info(msg string, kvs ...any) {
	d.l.Printf("[INFO] %s %v", msg, d.mergeArgs(kvs...))
}
func (d *defaultLogger) Warn(msg string, kvs ...any) {
	d.l.Printf("[WARN] %s %v", msg, d.mergeArgs(kvs...))
}
func (d *defaultLogger) Error(msg string, kvs ...any) {
	d.l.Printf("[ERROR] %s %v", msg, d.mergeArgs(kvs...))
}
func (d *defaultLogger) Fatal(msg string, kvs ...any) {
	d.l.Printf("[FATAL] %s %v", msg, d.mergeArgs(kvs...))
	os.Exit(1)
}

// 格式化输出时，如果该 Logger 通过 With 携带了公共参数，我们将参数打印在末尾
func (d *defaultLogger) formatSuffix() string {
	if len(d.args) > 0 {
		return fmt.Sprintf(" %v", d.args)
	}
	return ""
}

func (d *defaultLogger) Debugf(format string, args ...any) {
	d.l.Printf("[DEBUG] "+format+d.formatSuffix(), args...)
}
func (d *defaultLogger) Infof(format string, args ...any) {
	d.l.Printf("[INFO] "+format+d.formatSuffix(), args...)
}
func (d *defaultLogger) Warnf(format string, args ...any) {
	d.l.Printf("[WARN] "+format+d.formatSuffix(), args...)
}
func (d *defaultLogger) Errorf(format string, args ...any) {
	d.l.Printf("[ERROR] "+format+d.formatSuffix(), args...)
}
func (d *defaultLogger) Fatalf(format string, args ...any) {
	d.l.Printf("[FATAL] "+format+d.formatSuffix(), args...)
	os.Exit(1)
}

func (d *defaultLogger) With(kvs ...any) Logger {
	return &defaultLogger{
		l:    d.l,
		args: d.mergeArgs(kvs...),
	}
}
