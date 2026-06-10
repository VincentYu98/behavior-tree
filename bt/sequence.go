package bt

// Sequence 从左到右依次执行子节点（&& 语义，Memory 版本）
// 终态时清自己的 nodeState。子节点在各自返回终态时已清理自身 nodeState。
// resetFn 仅在 Running 被打断时由父级 Reset 级联调用。
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
			ctx.clearNodeState(s.id)
			return Failure
		case Running:
			ctx.setNodeState(s.id, i)
			return Running
		}
	}
	ctx.clearNodeState(s.id)
	return Success
}

func (s *Sequence) Reset(ctx *Context) {
	ctx.clearNodeState(s.id)
	for _, child := range s.children {
		child.Reset(ctx)
	}
}
