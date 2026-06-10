package bt

// Selector 从左到右尝试子节点（|| 语义，Memory 版本）
//
// 子节点返回 Running 后，下次 Tick 从断点恢复，不重新尝试已失败的高优先级分支。
// 如果需要每帧重新评估优先级，使用 ReactiveSelector。
type Selector struct {
	children   []Node
	runningIdx int
}

func NewSelector(children ...Node) *Selector {
	return &Selector{children: children, runningIdx: -1}
}

func (s *Selector) Tick(ctx *Context) Status {
	start := 0
	if s.runningIdx >= 0 {
		start = s.runningIdx
	}
	for i := start; i < len(s.children); i++ {
		status := s.children[i].Tick(ctx)
		switch status {
		case Success:
			s.runningIdx = -1
			return Success
		case Running:
			s.runningIdx = i
			return Running
		}
	}
	s.runningIdx = -1
	return Failure
}

func (s *Selector) Reset() {
	s.runningIdx = -1
	for _, child := range s.children {
		child.Reset()
	}
}
