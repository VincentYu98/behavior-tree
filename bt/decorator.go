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
// 迭代间 Reset 子树保证每轮干净。最后一轮不调 Reset，保留结果给同帧兄弟。
// re-entry 的 nodeState 由子节点自身终态清理保证（每个节点返回终态时清自己的 ctx.ns）。
// resetFn 仅在打断路径调用（Repeater 失败、父级中断等），不在正常完成后调用。
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
			if i < r.count-1 {
				r.child.Reset(ctx)
			}
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
