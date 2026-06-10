package bt

// ReactiveSelector 每帧从 child[0] 重新评估，高优先级可抢占 Running 分支。
type ReactiveSelector struct {
	id       int
	children []Node
}

func NewReactiveSelector(children ...Node) *ReactiveSelector {
	return &ReactiveSelector{id: nextNodeID(), children: children}
}

func (rs *ReactiveSelector) Tick(ctx *Context) Status {
	prevRunning, _ := getNodeState[int](ctx, rs.id)
	hasPrev := false
	if _, ok := getNodeState[int](ctx, rs.id); ok {
		hasPrev = true
	}

	for i, child := range rs.children {
		status := child.Tick(ctx)
		switch status {
		case Success:
			if hasPrev && prevRunning != i {
				rs.children[prevRunning].Reset(ctx)
			}
			ctx.clearNodeState(rs.id)
			return Success
		case Running:
			if hasPrev && prevRunning != i {
				rs.children[prevRunning].Reset(ctx)
			}
			ctx.setNodeState(rs.id, i)
			return Running
		}
	}
	if hasPrev {
		rs.children[prevRunning].Reset(ctx)
	}
	ctx.clearNodeState(rs.id)
	return Failure
}

func (rs *ReactiveSelector) Reset(ctx *Context) {
	ctx.clearNodeState(rs.id)
	for _, child := range rs.children {
		child.Reset(ctx)
	}
}

// ReactiveSequence 每帧从 child[0] 重新评估，前置条件失效时中止后续节点。
type ReactiveSequence struct {
	id       int
	children []Node
}

func NewReactiveSequence(children ...Node) *ReactiveSequence {
	return &ReactiveSequence{id: nextNodeID(), children: children}
}

func (rs *ReactiveSequence) Tick(ctx *Context) Status {
	prevRunning, _ := getNodeState[int](ctx, rs.id)
	hasPrev := false
	if _, ok := getNodeState[int](ctx, rs.id); ok {
		hasPrev = true
	}

	for i, child := range rs.children {
		status := child.Tick(ctx)
		switch status {
		case Failure:
			if hasPrev && prevRunning != i {
				rs.children[prevRunning].Reset(ctx)
			}
			ctx.clearNodeState(rs.id)
			return Failure
		case Running:
			if hasPrev && prevRunning != i {
				rs.children[prevRunning].Reset(ctx)
			}
			ctx.setNodeState(rs.id, i)
			return Running
		}
	}
	if hasPrev {
		rs.children[prevRunning].Reset(ctx)
	}
	ctx.clearNodeState(rs.id)
	return Success
}

func (rs *ReactiveSequence) Reset(ctx *Context) {
	ctx.clearNodeState(rs.id)
	for _, child := range rs.children {
		child.Reset(ctx)
	}
}
