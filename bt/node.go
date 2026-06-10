package bt

type Status int

const (
	Success Status = iota
	Failure
	Running // 行为还在执行中，下次 Tick 会从断点继续
)

func (s Status) String() string {
	switch s {
	case Success:
		return "Success"
	case Failure:
		return "Failure"
	case Running:
		return "Running"
	default:
		return "Unknown"
	}
}

type Node interface {
	Tick() Status
	// Reset 清除节点的运行时状态（runningIdx 等）
	// 当父节点决定不再继续执行某个 Running 子树时调用
	Reset()
}
