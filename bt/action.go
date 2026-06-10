package bt

type Action struct {
	Name    string
	fn      func(ctx *Context) Status
	resetFn func(ctx *Context)
}

func NewAction(name string, fn func(ctx *Context) Status) *Action {
	return &Action{Name: name, fn: fn}
}

func NewActionWithReset(name string, fn func(ctx *Context) Status, resetFn func(ctx *Context)) *Action {
	return &Action{Name: name, fn: fn, resetFn: resetFn}
}

func (a *Action) Tick(ctx *Context) Status { return a.fn(ctx) }

func (a *Action) Reset(ctx *Context) {
	if a.resetFn != nil {
		a.resetFn(ctx)
	}
}
