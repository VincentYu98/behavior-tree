package bt

type Sequence struct {
	id       int
	label    string
	children []Node
}

func NewSequence(children ...Node) *Sequence {
	return &Sequence{id: nextNodeID(), label: "Sequence", children: children}
}

func (s *Sequence) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(s.label)
	defer func() { ctx.traceExit(s.label, status) }()

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
	ctx.traceReset(s.label)
	ctx.clearNodeState(s.id)
	for _, child := range s.children {
		child.Reset(ctx)
	}
}
