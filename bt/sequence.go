package bt

// Sequence 从左到右依次执行子节点（&& 语义，Memory 版本）
//
// 子节点返回 Running 后，下次 Tick 从断点恢复，不重新执行已成功的子节点。
// 如果需要每帧重新评估前置条件，使用 ReactiveSequence。
type Sequence struct {
	children   []Node
	runningIdx int
}

func NewSequence(children ...Node) *Sequence {
	return &Sequence{children: children, runningIdx: -1}
}

func (s *Sequence) Tick(ctx *Context) Status {
	start := 0
	if s.runningIdx >= 0 {
		start = s.runningIdx
	}
	for i := start; i < len(s.children); i++ {
		status := s.children[i].Tick(ctx)
		switch status {
		case Failure:
			s.runningIdx = -1
			return Failure
		case Running:
			s.runningIdx = i
			return Running
		}
	}
	s.runningIdx = -1
	return Success
}

func (s *Sequence) Reset() {
	s.runningIdx = -1
	for _, child := range s.children {
		child.Reset()
	}
}
