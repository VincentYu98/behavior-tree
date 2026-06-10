package bt

// Inverter Success ↔ Failure，Running 不变
type Inverter struct {
	child Node
}

func NewInverter(child Node) *Inverter {
	return &Inverter{child: child}
}

func (i *Inverter) Tick(ctx *Context) Status {
	switch i.child.Tick(ctx) {
	case Success:
		return Failure
	case Failure:
		return Success
	default:
		return Running
	}
}

func (i *Inverter) Reset(ctx *Context) { i.child.Reset(ctx) }

// Repeater 重复执行子节点 N 次。
// 每轮 Success 后立即 Reset 子树（包括最后一轮），保证 re-entry 干净。
// 不依赖父级级联——裸 Repeater 作为根节点也能正确工作。
type Repeater struct {
	id    int
	count int
	child Node
}

func NewRepeater(count int, child Node) *Repeater {
	return &Repeater{id: nextNodeID(), count: count, child: child}
}

func (r *Repeater) Tick(ctx *Context) Status {
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
			r.child.Reset(ctx)
		}
	}
	ctx.clearNodeState(r.id)
	return Success
}

func (r *Repeater) Reset(ctx *Context) {
	ctx.clearNodeState(r.id)
	r.child.Reset(ctx)
}

// UntilFail 每帧执行子节点一次，Success 后 Reset 子树为下帧做准备。
type UntilFail struct {
	child Node
}

func NewUntilFail(child Node) *UntilFail {
	return &UntilFail{child: child}
}

func (u *UntilFail) Tick(ctx *Context) Status {
	status := u.child.Tick(ctx)
	if status == Failure {
		return Success
	}
	if status == Success {
		u.child.Reset(ctx)
	}
	return Running
}

func (u *UntilFail) Reset(ctx *Context) { u.child.Reset(ctx) }

// AlwaysSucceed 吞掉 Failure，Running 穿透
type AlwaysSucceed struct {
	child Node
}

func NewAlwaysSucceed(child Node) *AlwaysSucceed {
	return &AlwaysSucceed{child: child}
}

func (a *AlwaysSucceed) Tick(ctx *Context) Status {
	if a.child.Tick(ctx) == Running {
		return Running
	}
	return Success
}

func (a *AlwaysSucceed) Reset(ctx *Context) { a.child.Reset(ctx) }
