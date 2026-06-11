package bt

type Interrupt struct {
	event string
	label string
	child Node
}

func NewInterrupt(event string, child Node) *Interrupt {
	return &Interrupt{event: event, child: child, label: "Interrupt(" + event + ")"}
}

func (i *Interrupt) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(i.label)
	defer func() { ctx.traceExit(i.label, status) }()

	if ctx != nil && ctx.Bus != nil {
		if _, fired := ctx.Bus.Poll(i.event); fired {
			ctx.traceInterrupt(i.label, i.event)
			i.child.Reset(ctx)
			return Failure
		}
	}
	return i.child.Tick(ctx)
}

func (i *Interrupt) Reset(ctx *Context) {
	ctx.traceReset(i.label)
	i.child.Reset(ctx)
}
