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
//
// 字段职责：
//   - BB:     共享数据黑板，节点间通信的唯一通道（跨帧持久）
//   - Bus:    事件总线，帧内一次性信号（Emit → Poll → Clear）
//   - Delta:  帧间隔秒数，供需要时间的节点使用
//   - Logger: 调试输出接口，nil 则静默
//   - ns:     节点运行态（runningIdx 等），框架内部管理，外部不要直接操作
type Context struct {
	BB     *Blackboard
	Bus    *EventBus
	Delta  float64
	Logger Logger
	ns     map[int]any
}

// Log 通过 Logger 输出调试信息。Logger 为 nil 时静默。
func (ctx *Context) Log(format string, args ...any) {
	if ctx != nil && ctx.Logger != nil {
		ctx.Logger.Printf(format, args...)
	}
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

// Node 行为树节点接口。
//
// 生命周期语义：
//
//   Tick(ctx) — 每帧调用，驱动节点执行一步决策。
//     返回 Success/Failure: 终态。节点自清自己的 nodeState(ctx.ns)。
//     返回 Running: 未完成，父节点记住断点，下帧从此处恢复。
//
//   Reset(ctx) — 打断。仅在 Running 被放弃时由父级调用。
//     级联清理 nodeState + 调用 Action.resetFn 清理未完成的业务状态。
//     正常终态路径不触发 Reset。
//
// Action 约定：
//   - fn 在返回 Success/Failure 前负责清理自己的 BB 内部状态
//   - resetFn 仅处理 Running 被中断时的清理（如蓄力计数器归零）
//   - resetFn 不应清理"结果类"BB key（那些需要被兄弟节点消费的数据）
type Node interface {
	Tick(ctx *Context) Status
	Reset(ctx *Context)
}

var nodeSeq atomic.Int64

func nextNodeID() int {
	return int(nodeSeq.Add(1))
}
