package bt

import "fmt"

// Blackboard 行为树的共享数据黑板
// 节点之间不直接传数据，通过黑板解耦
type Blackboard struct {
	data map[string]any
}

func NewBlackboard() *Blackboard {
	return &Blackboard{data: make(map[string]any)}
}

func (b *Blackboard) Set(key string, value any) {
	b.data[key] = value
}

func (b *Blackboard) Has(key string) bool {
	_, ok := b.data[key]
	return ok
}

func (b *Blackboard) Delete(key string) {
	delete(b.data, key)
}

func (b *Blackboard) GetAny(key string) (any, bool) {
	val, ok := b.data[key]
	return val, ok
}

func Get[T any](b *Blackboard, key string) (T, bool) {
	val, ok := b.data[key]
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := val.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return typed, true
}

func MustGet[T any](b *Blackboard, key string) T {
	val, ok := Get[T](b, key)
	if !ok {
		panic(fmt.Sprintf("blackboard: key %q not found or type mismatch", key))
	}
	return val
}
