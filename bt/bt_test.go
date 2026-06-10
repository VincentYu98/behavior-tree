package bt

import (
	"encoding/json"
	"testing"
)

func testCtx() *Context {
	return &Context{BB: NewBlackboard(), Bus: NewEventBus(), Delta: 0.016}
}

func statusAction(s Status) *Action {
	return NewAction("test", func(_ *Context) Status { return s })
}

func countAction(s Status) (*Action, *int) {
	count := 0
	a := NewAction("test", func(_ *Context) Status { count++; return s })
	return a, &count
}

// bbStepAction 用黑板跟踪状态，适合多实体测试（闭包变量会跨实体污染）
func bbStepAction(key string, firstStatus, thenStatus Status) *Action {
	return NewAction("step", func(ctx *Context) Status {
		v, _ := Get[int](ctx.BB, key)
		ctx.BB.Set(key, v+1)
		if v == 0 {
			return firstStatus
		}
		return thenStatus
	})
}

// ==================== Sequence ====================

func TestSequence_AllSuccess(t *testing.T) {
	ctx := testCtx()
	seq := NewSequence(statusAction(Success), statusAction(Success))
	if s := seq.Tick(ctx); s != Success {
		t.Fatalf("want Success, got %s", s)
	}
}

func TestSequence_FailShortCircuit(t *testing.T) {
	ctx := testCtx()
	third, cnt3 := countAction(Success)
	seq := NewSequence(statusAction(Success), statusAction(Failure), third)
	if s := seq.Tick(ctx); s != Failure {
		t.Fatalf("want Failure, got %s", s)
	}
	if *cnt3 != 0 {
		t.Fatal("third child should not be ticked")
	}
}

func TestSequence_RunningResume(t *testing.T) {
	ctx := testCtx()
	first, cnt1 := countAction(Success)
	ctx.BB.Set("_step", 0)
	second := bbStepAction("_step", Running, Success)
	seq := NewSequence(first, second)

	if s := seq.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}
	if s := seq.Tick(ctx); s != Success {
		t.Fatalf("tick2: want Success, got %s", s)
	}
	if *cnt1 != 1 {
		t.Fatalf("first should be ticked once (memory), got %d", *cnt1)
	}
}

// ==================== Selector ====================

func TestSelector_AllFail(t *testing.T) {
	ctx := testCtx()
	if s := NewSelector(statusAction(Failure), statusAction(Failure)).Tick(ctx); s != Failure {
		t.Fatalf("want Failure, got %s", s)
	}
}

func TestSelector_RunningResume(t *testing.T) {
	ctx := testCtx()
	first, cnt1 := countAction(Failure)
	ctx.BB.Set("_step", 0)
	second := bbStepAction("_step", Running, Success)
	sel := NewSelector(first, second)

	sel.Tick(ctx)
	sel.Tick(ctx)
	if *cnt1 != 1 {
		t.Fatalf("first should be ticked once (memory), got %d", *cnt1)
	}
}

// ==================== ReactiveSelector ====================

func TestReactiveSelector_Preemption(t *testing.T) {
	ctx := testCtx()
	resetCalled := false
	lowPri := NewActionWithReset("low", func(_ *Context) Status {
		return Running
	}, func(_ *Context) { resetCalled = true })

	ctx.BB.Set("_step", 0)
	hiPri := bbStepAction("_step", Failure, Success)
	rs := NewReactiveSelector(hiPri, lowPri)

	if s := rs.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}
	if s := rs.Tick(ctx); s != Success {
		t.Fatalf("tick2: want Success, got %s", s)
	}
	if !resetCalled {
		t.Fatal("lowPri should be Reset on preemption")
	}
}

// ==================== ReactiveSequence ====================

func TestReactiveSequence_ConditionFails(t *testing.T) {
	ctx := testCtx()
	ctx.BB.Set("ok", true)
	resetCalled := false
	action := NewActionWithReset("act", func(_ *Context) Status {
		return Running
	}, func(_ *Context) { resetCalled = true })

	rs := NewReactiveSequence(NewCondition("ok", "eq", true), action)
	rs.Tick(ctx)

	ctx.BB.Set("ok", false)
	if s := rs.Tick(ctx); s != Failure {
		t.Fatalf("want Failure, got %s", s)
	}
	if !resetCalled {
		t.Fatal("action should be Reset when condition failed")
	}
}

// ==================== Repeater ====================

func TestRepeater_ResetsChildBetweenIterations(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Success
	}, func(_ *Context) { resetCount++ })

	NewRepeater(3, child).Tick(ctx)
	// Reset between iter 0→1 and 1→2, NOT after last
	if resetCount != 2 {
		t.Fatalf("want 2 resets (between iterations), got %d", resetCount)
	}
}

func TestRepeater_PreservesLastIterationResults(t *testing.T) {
	ctx := testCtx()
	child := NewAction("c", func(ctx *Context) Status {
		ctx.BB.Set("result", 42)
		return Success
	})
	seq := NewSequence(
		NewRepeater(2, child),
		NewAction("read", func(ctx *Context) Status {
			v, _ := Get[int](ctx.BB, "result")
			if v != 42 {
				return Failure
			}
			return Success
		}),
	)
	if s := seq.Tick(ctx); s != Success {
		t.Fatal("sibling should see Repeater's last iteration result")
	}
}

// ==================== UntilFail ====================

func TestUntilFail_SingleStepPerTick(t *testing.T) {
	ctx := testCtx()
	tickCount := 0
	child := NewAction("c", func(_ *Context) Status { tickCount++; return Success })
	uf := NewUntilFail(child)
	uf.Tick(ctx)
	if tickCount != 1 {
		t.Fatalf("child should be ticked once per frame, got %d", tickCount)
	}
}

func TestUntilFail_ResetsChildAfterSuccess(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	callCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		callCount++
		if callCount >= 3 {
			return Failure
		}
		return Success
	}, func(_ *Context) { resetCount++ })

	uf := NewUntilFail(child)
	uf.Tick(ctx) // Success → Running + Reset
	uf.Tick(ctx) // Success → Running + Reset
	uf.Tick(ctx) // Failure → Success, no Reset
	if resetCount != 2 {
		t.Fatalf("want 2 resets (after each Success round), got %d", resetCount)
	}
}

// ==================== Inverter / AlwaysSucceed ====================

func TestInverter(t *testing.T) {
	ctx := testCtx()
	if s := NewInverter(statusAction(Success)).Tick(ctx); s != Failure {
		t.Fatalf("got %s", s)
	}
	if s := NewInverter(statusAction(Failure)).Tick(ctx); s != Success {
		t.Fatalf("got %s", s)
	}
}

func TestAlwaysSucceed(t *testing.T) {
	ctx := testCtx()
	if s := NewAlwaysSucceed(statusAction(Failure)).Tick(ctx); s != Success {
		t.Fatalf("got %s", s)
	}
}

// ==================== Interrupt ====================

func TestInterrupt_AbortOnEvent(t *testing.T) {
	ctx := testCtx()
	resetCalled := false
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Running
	}, func(_ *Context) { resetCalled = true })

	intr := NewInterrupt("stun", child)
	intr.Tick(ctx)
	ctx.Bus.Emit("stun", nil)
	if s := intr.Tick(ctx); s != Failure {
		t.Fatalf("want Failure, got %s", s)
	}
	if !resetCalled {
		t.Fatal("child should be reset")
	}
}

// ==================== WaitForEvent ====================

func TestWaitForEvent_WritesToBlackboard(t *testing.T) {
	ctx := testCtx()
	w := NewWaitForEvent("ready", "count")
	w.Tick(ctx) // Running
	ctx.Bus.Emit("ready", 42)
	w.Tick(ctx) // Success
	if v := MustGet[int](ctx.BB, "count"); v != 42 {
		t.Fatalf("want 42, got %d", v)
	}
}

// ==================== Condition ====================

func TestCondition_Operators(t *testing.T) {
	ctx := testCtx()
	ctx.BB.Set("x", 10)
	tests := []struct {
		op   string
		val  any
		want Status
	}{
		{"eq", 10.0, Success}, {"eq", 5.0, Failure},
		{"ne", 5.0, Success}, {"lt", 20.0, Success},
		{"gt", 5.0, Success}, {"le", 10.0, Success},
		{"ge", 10.0, Success}, {"ge", 20.0, Failure},
	}
	for _, tc := range tests {
		if s := NewCondition("x", tc.op, tc.val).Tick(ctx); s != tc.want {
			t.Errorf("x=10 %s %v: want %s, got %s", tc.op, tc.val, tc.want, s)
		}
	}
}

func TestCondition_NilContext(t *testing.T) {
	c := NewCondition("x", "eq", 1)
	if s := c.Tick(nil); s != Failure {
		t.Fatalf("nil ctx: got %s", s)
	}
	if s := c.Tick(&Context{}); s != Failure {
		t.Fatalf("nil BB: got %s", s)
	}
}

// ==================== Loader Validation ====================

func TestLoader_UnknownAction(t *testing.T) {
	_, err := NewLoader().LoadJSON([]byte(`{"type":"action","action":"x"}`))
	if err == nil {
		t.Fatal("should fail")
	}
}

func TestLoader_InvalidOp(t *testing.T) {
	_, err := NewLoader().LoadJSON([]byte(`{"type":"condition","key":"x","op":"bad","value":1}`))
	if err == nil {
		t.Fatal("should fail")
	}
}

func TestLoader_EmptyChildren(t *testing.T) {
	_, err := NewLoader().LoadJSON([]byte(`{"type":"sequence","children":[]}`))
	if err == nil {
		t.Fatal("should fail")
	}
}

func TestLoader_ValidBuild(t *testing.T) {
	l := NewLoader()
	l.RegisterAction("noop", func(_ *Context) Status { return Success })
	cfg := NodeConfig{Type: "sequence", Children: []*NodeConfig{
		{Type: "condition", Key: "hp", Op: "lt", Value: 50.0},
		{Type: "action", Action: "noop"},
	}}
	data, _ := json.Marshal(cfg)
	node, err := l.LoadJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	ctx := testCtx()
	ctx.BB.Set("hp", 30)
	if s := node.Tick(ctx); s != Success {
		t.Fatalf("want Success, got %s", s)
	}
}

// ==================== EventBus ====================

func TestPoll_ReturnsLatestEvent(t *testing.T) {
	bus := NewEventBus()
	bus.Emit("dmg", 10)
	bus.Emit("dmg", 50)
	evt, _ := bus.Poll("dmg")
	if evt.Data.(int) != 50 {
		t.Fatalf("want latest (50), got %v", evt.Data)
	}
}

func TestPollAll_ReturnsAllEvents(t *testing.T) {
	bus := NewEventBus()
	bus.Emit("dmg", 10)
	bus.Emit("dmg", 25)
	if all := bus.PollAll("dmg"); len(all) != 2 {
		t.Fatalf("want 2, got %d", len(all))
	}
}

// ==================== Nil Safety ====================

func TestWaitForEvent_NilBus(t *testing.T) {
	if s := NewWaitForEvent("e", "k").Tick(nil); s != Failure {
		t.Fatalf("got %s", s)
	}
}

func TestInterrupt_NilBus_PassesThrough(t *testing.T) {
	intr := NewInterrupt("e", statusAction(Success))
	if s := intr.Tick(&Context{BB: NewBlackboard()}); s != Success {
		t.Fatalf("got %s", s)
	}
}

// ==================== Numeric Coercion ====================

func TestGet_NumericCoercion(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("hp", 100.0)
	if v, ok := Get[int](bb, "hp"); !ok || v != 100 {
		t.Fatalf("float64→int: got %v %v", v, ok)
	}
	bb.Set("speed", 5)
	if f, ok := Get[float64](bb, "speed"); !ok || f != 5.0 {
		t.Fatalf("int→float64: got %v %v", f, ok)
	}
	bb.Set("flag", true)
	if _, ok := Get[int](bb, "flag"); ok {
		t.Fatal("bool→int should fail")
	}
}

func TestConditionAndGet_ConsistentOnSameKey(t *testing.T) {
	ctx := testCtx()
	ctx.BB.Set("count", 3.0)
	if s := NewCondition("count", "eq", 3.0).Tick(ctx); s != Success {
		t.Fatal("condition should pass")
	}
	if v, ok := Get[int](ctx.BB, "count"); !ok || v != 3 {
		t.Fatalf("Get[int] should also work: got %v %v", v, ok)
	}
}

// ==================== Reset 幂等性 ====================

func TestReactiveSelector_ResetNoDoubleReset(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Running
	}, func(_ *Context) { resetCount++ })

	rs := NewReactiveSelector(statusAction(Failure), child)
	rs.Tick(ctx)
	rs.Reset(ctx)
	if resetCount != 1 {
		t.Fatalf("want exactly 1 reset, got %d", resetCount)
	}
}

func TestReactiveSequence_ResetNoDoubleReset(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Running
	}, func(_ *Context) { resetCount++ })

	rs := NewReactiveSequence(statusAction(Success), child)
	rs.Tick(ctx)
	rs.Reset(ctx)
	if resetCount != 1 {
		t.Fatalf("want exactly 1 reset, got %d", resetCount)
	}
}

// ==================== 终态不误清兄弟数据 ====================

func TestSequence_TerminalDoesNotResetChildren(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Success
	}, func(_ *Context) { resetCount++ })

	seq := NewSequence(child, statusAction(Success))
	seq.Tick(ctx)
	// 终态只清自身 nodeState，不调用子节点 resetFn
	if resetCount != 0 {
		t.Fatalf("terminal should NOT reset children, got %d resets", resetCount)
	}
}

func TestSequence_SubSeqResultVisibleToSibling(t *testing.T) {
	ctx := testCtx()
	// subSeq 写入 BB 结果，sibling 读取
	subSeq := NewSequence(
		NewAction("write", func(ctx *Context) Status {
			ctx.BB.Set("result", 42)
			return Success
		}),
	)
	sibling := NewAction("read", func(ctx *Context) Status {
		v, ok := Get[int](ctx.BB, "result")
		if !ok || v != 42 {
			return Failure
		}
		return Success
	})
	outer := NewSequence(subSeq, sibling)
	if s := outer.Tick(ctx); s != Success {
		t.Fatal("sibling should see subSeq's BB result")
	}
}

func TestReactiveSelector_PreemptionNoDoubleReset(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	lowPri := NewActionWithReset("low", func(_ *Context) Status {
		return Running
	}, func(_ *Context) { resetCount++ })

	ctx.BB.Set("_hi", 0)
	hiPri := bbStepAction("_hi", Failure, Success)
	rs := NewReactiveSelector(hiPri, lowPri)

	rs.Tick(ctx) // hiPri=Fail, lowPri=Running
	rs.Tick(ctx) // hiPri=Success → preempt lowPri
	// lowPri should be Reset exactly once (preemption), not twice
	if resetCount != 1 {
		t.Fatalf("want exactly 1 reset (preemption only), got %d", resetCount)
	}
}

// ==================== Blackboard nil 安全 ====================

func TestBlackboard_NilSafe(t *testing.T) {
	var bb *Blackboard

	v, ok := Get[int](bb, "x")
	if ok || v != 0 {
		t.Fatalf("Get on nil: want (0, false), got (%d, %v)", v, ok)
	}
	bb.Set("x", 1)
	if bb.Has("x") {
		t.Fatal("Has on nil should be false")
	}
	_, ok = bb.GetAny("x")
	if ok {
		t.Fatal("GetAny on nil should be false")
	}
	bb.Delete("x")
}

func TestBlackboard_ZeroValueSafe(t *testing.T) {
	bb := &Blackboard{} // data == nil

	// Set should lazy-init the map, not panic
	bb.Set("x", 42)
	v, ok := Get[int](bb, "x")
	if !ok || v != 42 {
		t.Fatalf("after Set on zero-value BB: want 42, got %v %v", v, ok)
	}

	// Other methods also safe before any Set
	bb2 := &Blackboard{}
	if bb2.Has("y") {
		t.Fatal("Has on zero-value should be false")
	}
	_, ok = bb2.GetAny("y")
	if ok {
		t.Fatal("GetAny on zero-value should be false")
	}
	bb2.Delete("y") // should not panic
}

// ==================== 浮点截断拒绝 ====================

func TestGet_RejectsNonIntegralFloat(t *testing.T) {
	bb := NewBlackboard()

	bb.Set("x", 3.9)
	_, ok := Get[int](bb, "x")
	if ok {
		t.Fatal("3.9 → int should be rejected (non-integral)")
	}

	bb.Set("y", 3.0)
	v, ok := Get[int](bb, "y")
	if !ok || v != 3 {
		t.Fatalf("3.0 → int should work: got %d %v", v, ok)
	}

	// float → float 不受限制
	bb.Set("z", 3.9)
	f, ok := Get[float64](bb, "z")
	if !ok || f != 3.9 {
		t.Fatalf("float64 → float64 should work: got %v %v", f, ok)
	}
}

// ==================== 多实体共享树 ====================

func TestMultiEntity_SharedTree(t *testing.T) {
	// 一棵树，两个实体各持自己的 Context
	tree := NewSequence(
		NewAction("inc", func(ctx *Context) Status {
			v, _ := Get[int](ctx.BB, "count")
			ctx.BB.Set("count", v+1)
			return Success
		}),
		bbStepAction("_step", Running, Success),
	)

	ctx1 := testCtx()
	ctx1.BB.Set("count", 0)
	ctx1.BB.Set("_step", 0)

	ctx2 := testCtx()
	ctx2.BB.Set("count", 0)
	ctx2.BB.Set("_step", 0)

	// Entity 1: inc(count=1), step=Running
	if s := tree.Tick(ctx1); s != Running {
		t.Fatalf("e1 tick1: want Running, got %s", s)
	}

	// Entity 2: 同一棵树，但 Context 独立，应该从 child[0] 开始
	if s := tree.Tick(ctx2); s != Running {
		t.Fatalf("e2 tick1: want Running, got %s", s)
	}
	if v := MustGet[int](ctx2.BB, "count"); v != 1 {
		t.Fatalf("e2 count: want 1 (inc executed), got %d", v)
	}

	// Entity 1 恢复: 从 child[1] 续跑（memory），inc 被跳过
	if s := tree.Tick(ctx1); s != Success {
		t.Fatalf("e1 tick2: want Success, got %s", s)
	}
	if v := MustGet[int](ctx1.BB, "count"); v != 1 {
		t.Fatalf("e1 count: want 1 (inc skipped on resume), got %d", v)
	}
}
