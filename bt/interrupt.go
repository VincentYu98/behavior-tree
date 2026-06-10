package bt

type Interrupt struct {
	event string
	child Node
}

func NewInterrupt(event string, child Node) *Interrupt {
	return &Interrupt{event: event, child: child}
}

func (i *Interrupt) Tick(ctx *Context) Status {
	if ctx != nil && ctx.Bus != nil {
		if _, fired := ctx.Bus.Poll(i.event); fired {
			i.child.Reset(ctx)
			return Failure
		}
	}
	return i.child.Tick(ctx)
}

func (i *Interrupt) Reset(ctx *Context) {
	i.child.Reset(ctx)
}
