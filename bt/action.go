package bt

// Action 叶子节点，包装一个具体的函数
// 函数可以返回 Running 表示"还没做完，下帧继续"
type Action struct {
	Name    string
	fn      func() Status
	resetFn func() // 可选：被父节点 Reset 时的清理函数（比如重置蓄力计数器）
}

func NewAction(name string, fn func() Status) *Action {
	return &Action{Name: name, fn: fn}
}

func NewActionWithReset(name string, fn func() Status, resetFn func()) *Action {
	return &Action{Name: name, fn: fn, resetFn: resetFn}
}

func (a *Action) Tick() Status { return a.fn() }

func (a *Action) Reset() {
	if a.resetFn != nil {
		a.resetFn()
	}
}
