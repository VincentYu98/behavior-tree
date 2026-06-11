package bt

import "fmt"

type Inverter struct {
	label string
	child Node
}

func NewInverter(child Node) *Inverter {
	return &Inverter{label: "Inverter", child: child}
}

func (i *Inverter) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(i.label)
	defer func() { ctx.traceExit(i.label, status) }()
	switch i.child.Tick(ctx) {
	case Success:
		return Failure
	case Failure:
		return Success
	default:
		return Running
	}
}

func (i *Inverter) Reset(ctx *Context) {
	ctx.traceReset(i.label)
	i.child.Reset(ctx)
}

type Repeater struct {
	id    int
	label string
	count int
	child Node
}

func NewRepeater(count int, child Node) *Repeater {
	return &Repeater{
		id: nextNodeID(), count: count, child: child,
		label: fmt.Sprintf("Repeater(%d)", count),
	}
}

func (r *Repeater) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(r.label)
	defer func() { ctx.traceExit(r.label, status) }()

	start := 0
	if idx, ok := getNodeState[int](ctx, r.id); ok {
		start = idx
	}
	for i := start; i < r.count; i++ {
		status := r.child.Tick(ctx)
		switch status {
		case Failure:
			ctx.clearNodeState(r.id)
			return Failure
		case Running:
			ctx.setNodeState(r.id, i)
			return Running
		default:
			if i < r.count-1 {
				r.child.Reset(ctx)
			}
		}
	}
	ctx.clearNodeState(r.id)
	return Success
}

func (r *Repeater) Reset(ctx *Context) {
	ctx.traceReset(r.label)
	ctx.clearNodeState(r.id)
	r.child.Reset(ctx)
}

type UntilFail struct {
	label string
	child Node
}

func NewUntilFail(child Node) *UntilFail {
	return &UntilFail{label: "UntilFail", child: child}
}

func (u *UntilFail) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(u.label)
	defer func() { ctx.traceExit(u.label, status) }()
	s := u.child.Tick(ctx)
	if s == Failure {
		return Success
	}
	if s == Success {
		u.child.Reset(ctx)
	}
	return Running
}

func (u *UntilFail) Reset(ctx *Context) {
	ctx.traceReset(u.label)
	u.child.Reset(ctx)
}

type AlwaysSucceed struct {
	label string
	child Node
}

func NewAlwaysSucceed(child Node) *AlwaysSucceed {
	return &AlwaysSucceed{label: "AlwaysSucceed", child: child}
}

func (a *AlwaysSucceed) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(a.label)
	defer func() { ctx.traceExit(a.label, status) }()
	if a.child.Tick(ctx) == Running {
		return Running
	}
	return Success
}

func (a *AlwaysSucceed) Reset(ctx *Context) {
	ctx.traceReset(a.label)
	a.child.Reset(ctx)
}
