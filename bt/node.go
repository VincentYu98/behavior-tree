package bt

import "sync/atomic"

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

// Context 行为树的执行上下文，每个实体各持一份。
// 同一棵树模板可被多个实体复用，各自的 Context 互不干扰。
type Context struct {
	BB    *Blackboard
	Bus   *EventBus
	Delta float64
	ns    map[int]any // node state: 各节点的运行态（runningIdx 等），按 nodeID 索引
}

func (ctx *Context) setNodeState(id int, val any) {
	if ctx == nil {
		return
	}
	if ctx.ns == nil {
		ctx.ns = make(map[int]any)
	}
	ctx.ns[id] = val
}

func (ctx *Context) clearNodeState(id int) {
	if ctx == nil || ctx.ns == nil {
		return
	}
	delete(ctx.ns, id)
}

func getNodeState[T any](ctx *Context, id int) (T, bool) {
	var zero T
	if ctx == nil || ctx.ns == nil {
		return zero, false
	}
	val, ok := ctx.ns[id]
	if !ok {
		return zero, false
	}
	typed, ok := val.(T)
	return typed, ok
}

// Node 接口。Tick 和 Reset 都接收 Context，节点本身无可变状态。
type Node interface {
	Tick(ctx *Context) Status
	Reset(ctx *Context)
}

var nodeSeq atomic.Int64

func nextNodeID() int {
	return int(nodeSeq.Add(1))
}
