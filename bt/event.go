package bt

type Event struct {
	Type string
	Data any
}

// EventBus 事件总线
// 生命周期：Emit → 节点 Poll/PollAll → 帧末 Clear
type EventBus struct {
	fired     []Event
	listeners map[string][]func(Event)
}

func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[string][]func(Event)),
	}
}

func (eb *EventBus) Emit(eventType string, data any) {
	evt := Event{Type: eventType, Data: data}
	eb.fired = append(eb.fired, evt)
	for _, handler := range eb.listeners[eventType] {
		handler(evt)
	}
}

// Poll 返回本帧该类型的最后一条事件（latest wins）。
// 同帧多次 Emit 同类型事件时，只取最新的。
func (eb *EventBus) Poll(eventType string) (Event, bool) {
	for i := len(eb.fired) - 1; i >= 0; i-- {
		if eb.fired[i].Type == eventType {
			return eb.fired[i], true
		}
	}
	return Event{}, false
}

// PollAll 返回本帧该类型的所有事件，按 Emit 顺序排列。
// 用于需要聚合的场景（累计伤害、多目标命中等）。
func (eb *EventBus) PollAll(eventType string) []Event {
	var result []Event
	for _, e := range eb.fired {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
}

func (eb *EventBus) Subscribe(eventType string, handler func(Event)) {
	eb.listeners[eventType] = append(eb.listeners[eventType], handler)
}

func (eb *EventBus) Clear() {
	eb.fired = eb.fired[:0]
}
