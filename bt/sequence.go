package bt

// Sequence 从左到右依次执行子节点（&& 语义）
//
// 断点续跑：如果某个子节点返回 Running，下次 Tick 直接从该子节点恢复
// 不会重新执行已经成功的子节点（比如蓄力技能时不会重复判断前置条件）
type Sequence struct {
	children   []Node
	runningIdx int // 上次 Running 的子节点下标，-1 表示无断点
}

func NewSequence(children ...Node) *Sequence {
	return &Sequence{children: children, runningIdx: -1}
}

func (s *Sequence) Tick() Status {
	start := 0
	if s.runningIdx >= 0 {
		start = s.runningIdx
	}

	for i := start; i < len(s.children); i++ {
		status := s.children[i].Tick()
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
