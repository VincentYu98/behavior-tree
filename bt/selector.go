package bt

type Selector struct {
	id       int
	label    string
	children []Node
}

func NewSelector(children ...Node) *Selector {
	return &Selector{id: nextNodeID(), label: "Selector", children: children}
}

func (s *Selector) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(s.label)
	defer func() { ctx.traceExit(s.label, status) }()

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
	ctx.traceReset(s.label)
	ctx.clearNodeState(s.id)
	for _, child := range s.children {
		child.Reset(ctx)
	}
}
