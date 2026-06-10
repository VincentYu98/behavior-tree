package bt

// WaitForEvent 等待事件节点
//
// 每帧检查 Context.Bus，事件触发则将 Data 写入 Context.BB（可选）并返回 Success。
// 如果 ctx 或 ctx.Bus 为 nil，返回 Failure（永远无法收到事件）。
type WaitForEvent struct {
	event   string
	writeTo string
}

func NewWaitForEvent(event, writeTo string) *WaitForEvent {
	return &WaitForEvent{event: event, writeTo: writeTo}
}

func (w *WaitForEvent) Tick(ctx *Context) Status {
	if ctx == nil || ctx.Bus == nil {
		return Failure
	}
	evt, ok := ctx.Bus.Poll(w.event)
	if !ok {
		return Running
	}
	if w.writeTo != "" && evt.Data != nil && ctx.BB != nil {
		ctx.BB.Set(w.writeTo, evt.Data)
	}
	return Success
}

func (w *WaitForEvent) Reset() {}
