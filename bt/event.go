package bt

// Event 一次事件，携带类型和可选数据
type Event struct {
	Type string
	Data any
}

// EventBus 事件总线
// 游戏系统在 Tick 前发射事件，行为树节点在 Tick 中消费
// 每帧结束后调用 Clear() 清理
type EventBus struct {
	fired     []Event
	listeners map[string][]func(Event)
}

func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[string][]func(Event)),
	}
}

// Emit 发射事件（在 tree.Tick() 之前调用）
func (eb *EventBus) Emit(eventType string, data any) {
	evt := Event{Type: eventType, Data: data}
	eb.fired = append(eb.fired, evt)
	for _, handler := range eb.listeners[eventType] {
		handler(evt)
	}
}

// Poll 检查本帧是否有指定类型的事件
func (eb *EventBus) Poll(eventType string) (Event, bool) {
	for _, e := range eb.fired {
		if e.Type == eventType {
			return e, true
		}
	}
	return Event{}, false
}

// Subscribe 注册监听器（可选，用于游戏系统间通信）
func (eb *EventBus) Subscribe(eventType string, handler func(Event)) {
	eb.listeners[eventType] = append(eb.listeners[eventType], handler)
}

// Clear 清理本帧事件，每帧结束时调用
func (eb *EventBus) Clear() {
	eb.fired = eb.fired[:0]
}
