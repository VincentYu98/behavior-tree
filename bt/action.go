package bt

type Action struct {
	Name    string
	fn      func(ctx *Context) Status
	resetFn func()
}

func NewAction(name string, fn func(ctx *Context) Status) *Action {
	return &Action{Name: name, fn: fn}
}

func NewActionWithReset(name string, fn func(ctx *Context) Status, resetFn func()) *Action {
	return &Action{Name: name, fn: fn, resetFn: resetFn}
}

func (a *Action) Tick(ctx *Context) Status { return a.fn(ctx) }

func (a *Action) Reset() {
	if a.resetFn != nil {
		a.resetFn()
	}
}
