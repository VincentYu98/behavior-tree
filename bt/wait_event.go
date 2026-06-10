package bt

// WaitForEvent 等待事件节点
//
// 每次 Tick 检查事件总线：
//   - 事件已触发 → 将 Data 写入黑板（可选）→ 返回 Success
//   - 事件未触发 → 返回 Running（下帧继续等）
//
// 典型场景：Boss 召唤小怪后，等 "minions_ready" 事件再发起协同攻击
type WaitForEvent struct {
	event   string
	writeTo string // 事件数据写入黑板的 key，空则不写
	bus     *EventBus
	bb      *Blackboard
}

func NewWaitForEvent(event string, writeTo string, bus *EventBus, bb *Blackboard) *WaitForEvent {
	return &WaitForEvent{event: event, writeTo: writeTo, bus: bus, bb: bb}
}

func (w *WaitForEvent) Tick() Status {
	evt, ok := w.bus.Poll(w.event)
	if !ok {
		return Running
	}
	if w.writeTo != "" && evt.Data != nil {
		w.bb.Set(w.writeTo, evt.Data)
	}
	return Success
}

func (w *WaitForEvent) Reset() {}
