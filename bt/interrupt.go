package bt

// Interrupt 事件打断装饰节点
//
// 每次 Tick 先检查 Context.Bus，目标事件已触发则 Reset 子树并返回 Failure。
type Interrupt struct {
	event string
	child Node
}

func NewInterrupt(event string, child Node) *Interrupt {
	return &Interrupt{event: event, child: child}
}

func (i *Interrupt) Tick(ctx *Context) Status {
	if _, fired := ctx.Bus.Poll(i.event); fired {
		i.child.Reset()
		return Failure
	}
	return i.child.Tick(ctx)
}

func (i *Interrupt) Reset() {
	i.child.Reset()
}
