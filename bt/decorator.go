package bt

// ============================================================
// 装饰节点：单子节点，对结果做变换
// ============================================================

// Inverter 取反：Success ↔ Failure，Running 不变
type Inverter struct {
	child Node
}

func NewInverter(child Node) *Inverter {
	return &Inverter{child: child}
}

func (i *Inverter) Tick() Status {
	switch i.child.Tick() {
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
// 支持断点续跑：子节点返回 Running 时记住当前轮次，下次从该轮恢复
type Repeater struct {
	count   int
	child   Node
	current int // 当前迭代轮次，-1 表示无断点
}

func NewRepeater(count int, child Node) *Repeater {
	return &Repeater{count: count, child: child, current: -1}
}

func (r *Repeater) Tick() Status {
	start := 0
	if r.current >= 0 {
		start = r.current
	}
	for i := start; i < r.count; i++ {
		status := r.child.Tick()
		switch status {
		case Failure:
			r.current = -1
			return Failure
		case Running:
			r.current = i
			return Running
		}
	}
	r.current = -1
	return Success
}

func (r *Repeater) Reset() {
	r.current = -1
	r.child.Reset()
}

// UntilFail 持续执行子节点直到它返回 Failure，然后返回 Success
type UntilFail struct {
	child Node
}

func NewUntilFail(child Node) *UntilFail {
	return &UntilFail{child: child}
}

func (u *UntilFail) Tick() Status {
	for {
		switch u.child.Tick() {
		case Failure:
			return Success
		case Running:
			return Running
		}
	}
}

func (u *UntilFail) Reset() { u.child.Reset() }

// AlwaysSucceed 无论子节点结果如何都返回 Success（Running 除外）
type AlwaysSucceed struct {
	child Node
}

func NewAlwaysSucceed(child Node) *AlwaysSucceed {
	return &AlwaysSucceed{child: child}
}

func (a *AlwaysSucceed) Tick() Status {
	if a.child.Tick() == Running {
		return Running
	}
	return Success
}

func (a *AlwaysSucceed) Reset() { a.child.Reset() }
