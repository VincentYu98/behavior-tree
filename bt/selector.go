package bt

// Selector 从左到右尝试子节点（|| 语义，Memory 版本）
type Selector struct {
	id       int
	children []Node
}

func NewSelector(children ...Node) *Selector {
	return &Selector{id: nextNodeID(), children: children}
}

func (s *Selector) Tick(ctx *Context) Status {
	start := 0
	if idx, ok := getNodeState[int](ctx, s.id); ok {
		start = idx
	}
	for i := start; i < len(s.children); i++ {
		status := s.children[i].Tick(ctx)
		switch status {
		case Success:
			ctx.clearNodeState(s.id)
			return Success
		case Running:
			ctx.setNodeState(s.id, i)
			return Running
		}
	}
	ctx.clearNodeState(s.id)
	return Failure
}

func (s *Selector) Reset(ctx *Context) {
	ctx.clearNodeState(s.id)
	for _, child := range s.children {
		child.Reset(ctx)
	}
}
