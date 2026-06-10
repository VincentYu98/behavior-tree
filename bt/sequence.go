package bt

// Sequence 从左到右依次执行子节点（&& 语义，Memory 版本）
// 返回终态时 Reset 全部子节点，保证 re-entry 干净。
type Sequence struct {
	id       int
	children []Node
}

func NewSequence(children ...Node) *Sequence {
	return &Sequence{id: nextNodeID(), children: children}
}

func (s *Sequence) Tick(ctx *Context) Status {
	start := 0
	if idx, ok := getNodeState[int](ctx, s.id); ok {
		start = idx
	}
	for i := start; i < len(s.children); i++ {
		status := s.children[i].Tick(ctx)
		switch status {
		case Failure:
			s.Reset(ctx)
			return Failure
		case Running:
			ctx.setNodeState(s.id, i)
			return Running
		}
	}
	s.Reset(ctx)
	return Success
}

func (s *Sequence) Reset(ctx *Context) {
	ctx.clearNodeState(s.id)
	for _, child := range s.children {
		child.Reset(ctx)
	}
}
