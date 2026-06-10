package bt

type Status int

const (
	Success Status = iota
	Failure
	Running
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

// Context 行为树的执行上下文，每帧由游戏主循环创建并传入
// 取代了之前通过闭包捕获 Blackboard/EventBus 的方式
type Context struct {
	BB    *Blackboard
	Bus   *EventBus
	Delta float64 // 帧间隔时间（秒）
}

type Node interface {
	Tick(ctx *Context) Status
	Reset()
}
