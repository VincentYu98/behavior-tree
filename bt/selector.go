package bt

// Selector 从左到右尝试子节点（|| 语义）
//
// 断点续跑：如果某个子节点返回 Running，下次 Tick 直接从该子节点恢复
// 不会重新尝试已经失败的高优先级分支
type Selector struct {
	children   []Node
	runningIdx int
}

func NewSelector(children ...Node) *Selector {
	return &Selector{children: children, runningIdx: -1}
}

func (s *Selector) Tick() Status {
	start := 0
	if s.runningIdx >= 0 {
		start = s.runningIdx
	}

	for i := start; i < len(s.children); i++ {
		status := s.children[i].Tick()
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
