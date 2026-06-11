package bt

type WaitForEvent struct {
	event   string
	writeTo string
	label   string
}

func NewWaitForEvent(event, writeTo string) *WaitForEvent {
	return &WaitForEvent{event: event, writeTo: writeTo, label: "WaitForEvent(" + event + ")"}
}

func (w *WaitForEvent) Tick(ctx *Context) (status Status) {
	ctx.traceEnter(w.label)
	defer func() { ctx.traceExit(w.label, status) }()

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
