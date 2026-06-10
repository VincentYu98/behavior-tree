package bt

// Interrupt 事件打断装饰节点
//
// 每次 Tick 先检查 Context.Bus，目标事件已触发则 Reset 子树并返回 Failure。
// 如果 ctx 或 ctx.Bus 为 nil，跳过事件检查，直接 Tick 子节点。
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
			i.child.Reset()
			return Failure
		}
	}
	return i.child.Tick(ctx)
}

func (i *Interrupt) Reset() {
	i.child.Reset()
}
