package bt

type Action struct {
	Name    string
	fn      func(ctx *Context) Status
	resetFn func(ctx *Context)
	label   string
}

func NewAction(name string, fn func(ctx *Context) Status) *Action {
	return &Action{Name: name, fn: fn, label: "Action(" + name + ")"}
}

func NewActionWithReset(name string, fn func(ctx *Context) Status, resetFn func(ctx *Context)) *Action {
	return &Action{Name: name, fn: fn, resetFn: resetFn, label: "Action(" + name + ")"}
}

func (a *Action) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(a.label)
	defer func() { ctx.traceExit(a.label, status) }()
	status = a.fn(ctx)
	return
}

func (a *Action) Reset(ctx *Context) {
	ctx.traceReset(a.label)
	if a.resetFn != nil {
		a.resetFn(ctx)
	}
}
