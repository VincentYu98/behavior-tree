package bt

import (
	"fmt"
	"math"
)

type Blackboard struct {
	data map[string]any
}

func NewBlackboard() *Blackboard {
	return &Blackboard{data: make(map[string]any)}
}

func (b *Blackboard) Set(key string, value any) {
	if b == nil {
		return
	}
	b.data[key] = value
}

func (b *Blackboard) Has(key string) bool {
	if b == nil {
		return false
	}
	_, ok := b.data[key]
	return ok
}

func (b *Blackboard) Delete(key string) {
	if b == nil {
		return
	}
	delete(b.data, key)
}

func (b *Blackboard) GetAny(key string) (any, bool) {
	if b == nil {
		return nil, false
	}
	val, ok := b.data[key]
	return val, ok
}

// Get 泛型取值。直接类型匹配失败时尝试数值互转。
// float→int 只接受整值浮点（3.0→3），非整值（3.9）返回 false 而非静默截断。
func Get[T any](b *Blackboard, key string) (T, bool) {
	var zero T
	if b == nil {
		return zero, false
	}
	val, ok := b.data[key]
	if !ok {
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

// coerceNumeric 在数值类型之间互转。
// float→int 仅接受整值（无小数部分），拒绝 NaN/Inf/非整值浮点。
func coerceNumeric[T any](val any) (T, bool) {
	var zero T
	f, ok := asFloat64(val)
	if !ok {
		return zero, false
	}
	switch p := any(&zero).(type) {
	case *int:
		if !isIntegral(f) {
			return zero, false
		}
		*p = int(f)
		return zero, true
	case *int32:
		if !isIntegral(f) {
			return zero, false
		}
		*p = int32(f)
		return zero, true
	case *int64:
		if !isIntegral(f) {
			return zero, false
		}
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

func isIntegral(f float64) bool {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return false
	}
	return f == math.Trunc(f)
}

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
