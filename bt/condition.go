package bt

// Condition 条件节点，从 Context.BB 读值并比较
// 支持: eq, ne, lt, gt, le, ge
type Condition struct {
	key   string
	op    string
	value any
}

func NewCondition(key, op string, value any) *Condition {
	return &Condition{key: key, op: op, value: value}
}

func (c *Condition) Tick(ctx *Context) Status {
	if ctx == nil || ctx.BB == nil {
		return Failure
	}
	raw, ok := ctx.BB.GetAny(c.key)
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

func numEquals(a, b any) bool {
	af, aOk := asFloat64(a)
	bf, bOk := asFloat64(b)
	if aOk && bOk {
		return af == bf
	}
	return a == b
}

func toFloats(a, b any) (float64, float64, bool) {
	af, aOk := asFloat64(a)
	bf, bOk := asFloat64(b)
	return af, bf, aOk && bOk
}
