package bt

// Condition 条件节点，从黑板读值并比较
// 支持的操作符：eq, ne, lt, gt, le, ge
// JSON 加载器用这个节点代替手写的条件 Action
type Condition struct {
	bb    *Blackboard
	key   string
	op    string
	value any
}

func NewCondition(bb *Blackboard, key, op string, value any) *Condition {
	return &Condition{bb: bb, key: key, op: op, value: value}
}

func (c *Condition) Tick() Status {
	raw, ok := c.bb.GetAny(c.key)
	if !ok {
		return Failure
	}

	switch c.op {
	case "eq":
		if numEquals(raw, c.value) {
			return Success
		}
		return Failure
	case "ne":
		if !numEquals(raw, c.value) {
			return Success
		}
		return Failure
	case "lt", "gt", "le", "ge":
		a, b, valid := toFloats(raw, c.value)
		if !valid {
			return Failure
		}
		switch c.op {
		case "lt":
			if a < b {
				return Success
			}
		case "gt":
			if a > b {
				return Success
			}
		case "le":
			if a <= b {
				return Success
			}
		case "ge":
			if a >= b {
				return Success
			}
		}
		return Failure
	}
	return Failure
}

func (c *Condition) Reset() {}

// 数值相等比较，兼容 int/float64 混合（JSON 数字解析为 float64，黑板值通常是 int）
func numEquals(a, b any) bool {
	af, aOk := toFloat(a)
	bf, bOk := toFloat(b)
	if aOk && bOk {
		return af == bf
	}
	return a == b
}

func toFloats(a, b any) (float64, float64, bool) {
	af, aOk := toFloat(a)
	bf, bOk := toFloat(b)
	return af, bf, aOk && bOk
}

func toFloat(v any) (float64, bool) {
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
