package bt

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

func (w *WaitForEvent) Reset(_ *Context) {}
