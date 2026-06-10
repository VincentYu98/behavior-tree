package bt

import "fmt"

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

// Get 泛型取值，直接类型匹配失败时尝试数值类型互转（int↔float64 等）。
// 这保证 Condition 的数值比较和业务层的 Get[int] 对同一个 key 行为一致。
func Get[T any](b *Blackboard, key string) (T, bool) {
	val, ok := b.data[key]
	if !ok {
		var zero T
		return zero, false
	}
	if typed, ok := val.(T); ok {
		return typed, true
	}
	return coerceNumeric[T](val)
}

func MustGet[T any](b *Blackboard, key string) T {
	val, ok := Get[T](b, key)
	if !ok {
		panic(fmt.Sprintf("blackboard: key %q not found or type mismatch", key))
	}
	return val
}

// coerceNumeric 尝试将 val 转为目标数值类型 T。
// 只在 int 系列和 float 系列之间互转，不处理 bool/string。
func coerceNumeric[T any](val any) (T, bool) {
	var zero T
	f, ok := asFloat64(val)
	if !ok {
		return zero, false
	}
	switch p := any(&zero).(type) {
	case *int:
		*p = int(f)
		return zero, true
	case *int32:
		*p = int32(f)
		return zero, true
	case *int64:
		*p = int64(f)
		return zero, true
	case *float32:
		*p = float32(f)
		return zero, true
	case *float64:
		*p = f
		return zero, true
	}
	return zero, false
}

// asFloat64 将数值类型统一转为 float64，供 Get 和 Condition 共用。
func asFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}
