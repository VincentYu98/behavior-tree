package bt

// Interrupt 事件打断装饰节点
//
// 每次 Tick 先检查事件总线，如果目标事件已触发：
//   1. 调用 child.Reset() 中止子树（级联重置所有状态）
//   2. 返回 Failure（让父节点选择其他分支）
//
// 典型场景：Boss 蓄力技能时被沉默，立即打断蓄力
// 与 ReactiveSelector 的区别：
//   - ReactiveSelector 靠黑板条件每帧重新评估优先级
//   - Interrupt 靠具体事件打断，适合"沉默/缴械/打断"等一次性触发
type Interrupt struct {
	event string
	child Node
	bus   *EventBus
}

func NewInterrupt(event string, bus *EventBus, child Node) *Interrupt {
	return &Interrupt{event: event, bus: bus, child: child}
}

func (i *Interrupt) Tick() Status {
	if _, fired := i.bus.Poll(i.event); fired {
		i.child.Reset()
		return Failure
	}
	return i.child.Tick()
}

func (i *Interrupt) Reset() {
	i.child.Reset()
}
