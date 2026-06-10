package bt

// Inverter 取反：Success ↔ Failure，Running 不变
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

func (i *Inverter) Reset() { i.child.Reset() }

// Repeater 重复执行子节点 N 次
// 每轮 Success 后调用 child.Reset()，保证子树从干净状态开始下一轮。
type Repeater struct {
	count   int
	child   Node
	current int
}

func NewRepeater(count int, child Node) *Repeater {
	return &Repeater{count: count, child: child, current: -1}
}

func (r *Repeater) Tick(ctx *Context) Status {
	start := 0
	if r.current >= 0 {
		start = r.current
	}
	for i := start; i < r.count; i++ {
		status := r.child.Tick(ctx)
		switch status {
		case Failure:
			r.current = -1
			return Failure
		case Running:
			r.current = i
			return Running
		default:
			// Success: 本轮完成，重置子树为下一轮做准备
			if i < r.count-1 {
				r.child.Reset()
			}
		}
	}
	r.current = -1
	return Success
}

func (r *Repeater) Reset() {
	r.current = -1
	r.child.Reset()
}

// UntilFail 每帧执行子节点一次
//   - 子节点 Failure → 返回 Success（循环结束）
//   - 子节点 Running → 返回 Running（等子节点完成）
//   - 子节点 Success → 返回 Running（下帧再试，不在同一帧内循环）
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
	return Running
}

func (u *UntilFail) Reset() { u.child.Reset() }

// AlwaysSucceed 无论子节点返回什么都返回 Success（Running 除外）
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

func (a *AlwaysSucceed) Reset() { a.child.Reset() }
