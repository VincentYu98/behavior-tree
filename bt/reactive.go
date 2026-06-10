package bt

// ReactiveSelector 响应式选择器
//
// 每帧从 child[0] 重新评估。高优先级分支变为可用时，Reset 正在 Running 的低优先级分支。
type ReactiveSelector struct {
	children   []Node
	runningIdx int
}

func NewReactiveSelector(children ...Node) *ReactiveSelector {
	return &ReactiveSelector{children: children, runningIdx: -1}
}

func (rs *ReactiveSelector) Tick(ctx *Context) Status {
	for i, child := range rs.children {
		status := child.Tick(ctx)
		switch status {
		case Success:
			if rs.runningIdx >= 0 && rs.runningIdx != i {
				rs.children[rs.runningIdx].Reset()
			}
			rs.runningIdx = -1
			return Success
		case Running:
			if rs.runningIdx >= 0 && rs.runningIdx != i {
				rs.children[rs.runningIdx].Reset()
			}
			rs.runningIdx = i
			return Running
		}
	}
	if rs.runningIdx >= 0 {
		rs.children[rs.runningIdx].Reset()
	}
	rs.runningIdx = -1
	return Failure
}

func (rs *ReactiveSelector) Reset() {
	rs.runningIdx = -1
	for _, child := range rs.children {
		child.Reset()
	}
}

// ReactiveSequence 响应式顺序节点
//
// 每帧从 child[0] 重新评估。前置条件失效时，Reset 正在 Running 的后续子节点。
// 典型用途：Sequence [条件: 玩家在范围, 行为: 攻击]
//   攻击 Running 时如果玩家离开范围，条件 Failure → Reset 攻击 → 整体 Failure。
type ReactiveSequence struct {
	children   []Node
	runningIdx int
}

func NewReactiveSequence(children ...Node) *ReactiveSequence {
	return &ReactiveSequence{children: children, runningIdx: -1}
}

func (rs *ReactiveSequence) Tick(ctx *Context) Status {
	for i, child := range rs.children {
		status := child.Tick(ctx)
		switch status {
		case Failure:
			if rs.runningIdx >= 0 && rs.runningIdx != i {
				rs.children[rs.runningIdx].Reset()
			}
			rs.runningIdx = -1
			return Failure
		case Running:
			if rs.runningIdx >= 0 && rs.runningIdx != i {
				rs.children[rs.runningIdx].Reset()
			}
			rs.runningIdx = i
			return Running
		}
	}
	if rs.runningIdx >= 0 {
		rs.children[rs.runningIdx].Reset()
	}
	rs.runningIdx = -1
	return Success
}

func (rs *ReactiveSequence) Reset() {
	rs.runningIdx = -1
	for _, child := range rs.children {
		child.Reset()
	}
}
