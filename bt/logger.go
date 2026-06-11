package bt

import "fmt"

// Logger 标准调试输出接口。
// Action 通过 ctx.Log() 输出，替代散落的 fmt.Println。
// 设为 nil 则静默。
type Logger interface {
	Printf(format string, args ...any)
}

// FmtLogger 输出到 stdout。
type FmtLogger struct{}

func (FmtLogger) Printf(format string, args ...any) {
	fmt.Printf(format, args...)
}

// NilLogger 静默，丢弃所有输出。
type NilLogger struct{}

func (NilLogger) Printf(string, ...any) {}
