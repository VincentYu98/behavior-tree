package bt

import "fmt"

type ReactiveSelector struct {
	id       int
	label    string
	children []Node
}

func NewReactiveSelector(children ...Node) *ReactiveSelector {
	return &ReactiveSelector{id: nextNodeID(), label: "ReactiveSelector", children: children}
}

func (rs *ReactiveSelector) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(rs.label)
	defer func() { ctx.traceExit(rs.label, status) }()

	prevRunning, hasPrev := getNodeState[int](ctx, rs.id)

	for i, child := range rs.children {
		status := child.Tick(ctx)
		switch status {
		case Success:
			if hasPrev && prevRunning != i {
				ctx.tracePreempt(fmt.Sprintf("child[%d]", prevRunning), fmt.Sprintf("child[%d]", i))
				rs.children[prevRunning].Reset(ctx)
			}
			ctx.clearNodeState(rs.id)
			return Success
		case Running:
			if hasPrev && prevRunning != i {
				ctx.tracePreempt(fmt.Sprintf("child[%d]", prevRunning), fmt.Sprintf("child[%d]", i))
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
	ctx.traceReset(rs.label)
	ctx.clearNodeState(rs.id)
	for _, child := range rs.children {
		child.Reset(ctx)
	}
}

type ReactiveSequence struct {
	id       int
	label    string
	children []Node
}

func NewReactiveSequence(children ...Node) *ReactiveSequence {
	return &ReactiveSequence{id: nextNodeID(), label: "ReactiveSequence", children: children}
}

func (rs *ReactiveSequence) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(rs.label)
	defer func() { ctx.traceExit(rs.label, status) }()

	prevRunning, hasPrev := getNodeState[int](ctx, rs.id)

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
	ctx.traceReset(rs.label)
	ctx.clearNodeState(rs.id)
	for _, child := range rs.children {
		child.Reset(ctx)
	}
}
