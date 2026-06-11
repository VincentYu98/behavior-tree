package bt

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
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

func TestRepeater_ResetsBetweenIterations(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Success
	}, func(_ *Context) { resetCount++ })

	NewRepeater(3, child).Tick(ctx)
	// Reset between iter 0→1 and 1→2, NOT after last (preserve results)
	if resetCount != 2 {
		t.Fatalf("want 2 resets (between iterations), got %d", resetCount)
	}
}

func TestRepeater_SiblingReadsResult(t *testing.T) {
	ctx := testCtx()
	child := NewActionWithReset("c", func(ctx *Context) Status {
		ctx.BB.Set("result", 42) // result for sibling
		return Success
	}, func(ctx *Context) {
		ctx.BB.Set("_internal", 0) // resetFn only clears internal state
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

func TestRepeater_ReentryNodeStateClean(t *testing.T) {
	ctx := testCtx()
	// 子节点是一个有 nodeState 的 Sequence（含多帧 action）
	inner := NewSequence(
		bbStepAction("_step", Running, Success),
		statusAction(Success),
	)
	rep := NewRepeater(1, inner)

	ctx.BB.Set("_step", 0)
	rep.Tick(ctx) // inner child[0]=Running → rep Running
	rep.Tick(ctx) // inner child[0]=Success, child[1]=Success → rep Success

	// re-entry: inner 的 nodeState 已在终态时自清，不需要 resetFn
	ctx.BB.Set("_step", 0)
	if s := rep.Tick(ctx); s != Running {
		t.Fatalf("re-entry: want Running (inner starts fresh from child[0]), got %s", s)
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
	cfg := NodeConfig{Type: "sequence", Name: "test-seq", Children: []*NodeConfig{
		{Type: "condition", Key: "hp", Op: "lt", Value: 50.0},
		{Type: "action", Action: "noop"},
	}}
	data, _ := json.Marshal(cfg)
	tree, err := l.LoadJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if tree.Name() != "test-seq" {
		t.Fatalf("tree name: want test-seq, got %s", tree.Name())
	}
	ctx := testCtx()
	ctx.BB.Set("hp", 30)
	exec := tree.NewExecutor(ctx)
	if s := exec.Tick(); s != Success {
		t.Fatalf("want Success, got %s", s)
	}
	if exec.Ticks() != 1 {
		t.Fatalf("want 1 tick, got %d", exec.Ticks())
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
	if _, ok := Get[int](bb, "x"); ok {
		t.Fatal("3.9 → int should be rejected")
	}

	bb.Set("y", 3.0)
	if v, ok := Get[int](bb, "y"); !ok || v != 3 {
		t.Fatalf("3.0 → int: got %d %v", v, ok)
	}

	bb.Set("z", 3.9)
	if f, ok := Get[float64](bb, "z"); !ok || f != 3.9 {
		t.Fatalf("float64 direct: got %v %v", f, ok)
	}
}

func TestGet_RejectsOutOfRangeFloat(t *testing.T) {
	bb := NewBlackboard()

	// 超出 int32 范围
	bb.Set("big", 1e18)
	if _, ok := Get[int32](bb, "big"); ok {
		t.Fatal("1e18 → int32 should be rejected (out of range)")
	}

	// int64 范围内的大整值应该可以
	bb.Set("ok", float64(1<<40))
	if v, ok := Get[int64](bb, "ok"); !ok || v != 1<<40 {
		t.Fatalf("2^40 → int64: got %d %v", v, ok)
	}

	// NaN
	bb.Set("nan", math.NaN())
	if _, ok := Get[int](bb, "nan"); ok {
		t.Fatal("NaN → int should be rejected")
	}

	// Inf
	bb.Set("inf", math.Inf(1))
	if _, ok := Get[int](bb, "inf"); ok {
		t.Fatal("Inf → int should be rejected")
	}
}

// ==================== Stateful Action re-entry 模式 ====================

func TestStatefulAction_CleansUpBeforeTerminal(t *testing.T) {
	ctx := testCtx()
	// 正确模式：action 在返回 Success 前清理自己的 BB 内部状态
	action := NewAction("channel", func(ctx *Context) Status {
		ticks, _ := Get[int](ctx.BB, "_ch_ticks")
		ticks++
		ctx.BB.Set("_ch_ticks", ticks)
		if ticks < 3 {
			return Running
		}
		ctx.BB.Set("_ch_ticks", 0) // 完成前清理
		return Success
	})

	sel := NewSelector(
		NewSequence(NewCondition("go", "eq", true), action),
		statusAction(Success),
	)

	ctx.BB.Set("go", true)
	ctx.BB.Set("_ch_ticks", 0)

	// 第一轮：3帧蓄力
	sel.Tick(ctx) // Running (ticks=1)
	sel.Tick(ctx) // Running (ticks=2)
	sel.Tick(ctx) // Success (ticks→0)
	if v := MustGet[int](ctx.BB, "_ch_ticks"); v != 0 {
		t.Fatalf("after completion: want _ch_ticks=0, got %d", v)
	}

	// 第二轮 re-entry：action 从干净状态开始
	sel.Tick(ctx) // Running (ticks=1)
	if v := MustGet[int](ctx.BB, "_ch_ticks"); v != 1 {
		t.Fatalf("re-entry: want _ch_ticks=1 (fresh start), got %d", v)
	}
}

// ==================== Tree / Executor ====================

func TestExecutor_Lifecycle(t *testing.T) {
	ctx := testCtx()
	ctx.BB.Set("_step", 0)
	root := bbStepAction("_step", Running, Success)
	tree := NewTree("test", root)
	exec := tree.NewExecutor(ctx)

	if exec.Ticks() != 0 {
		t.Fatalf("initial ticks: want 0, got %d", exec.Ticks())
	}

	// Tick 1: Running
	if s := exec.Tick(); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}
	if !exec.IsRunning() {
		t.Fatal("should be running")
	}
	if exec.Ticks() != 1 {
		t.Fatalf("ticks: want 1, got %d", exec.Ticks())
	}

	// Tick 2: Success
	if s := exec.Tick(); s != Success {
		t.Fatalf("tick2: want Success, got %s", s)
	}
	if exec.IsRunning() {
		t.Fatal("should not be running")
	}
	if exec.LastStatus() != Success {
		t.Fatalf("last status: want Success, got %s", exec.LastStatus())
	}
}

func TestExecutor_Reset(t *testing.T) {
	ctx := testCtx()
	resetCalled := false
	root := NewActionWithReset("a", func(_ *Context) Status {
		return Running
	}, func(_ *Context) { resetCalled = true })

	exec := NewTree("test", root).NewExecutor(ctx)
	exec.Tick() // Running
	exec.Reset()
	if !resetCalled {
		t.Fatal("Reset should cascade to root")
	}
}

func TestExecutor_MultiInstance(t *testing.T) {
	root := NewSequence(
		NewAction("inc", func(ctx *Context) Status {
			v, _ := Get[int](ctx.BB, "n")
			ctx.BB.Set("n", v+1)
			return Success
		}),
		bbStepAction("_s", Running, Success),
	)
	tree := NewTree("shared", root)

	ctx1 := testCtx()
	ctx1.BB.Set("n", 0)
	ctx1.BB.Set("_s", 0)
	ctx2 := testCtx()
	ctx2.BB.Set("n", 0)
	ctx2.BB.Set("_s", 0)

	e1 := tree.NewExecutor(ctx1)
	e2 := tree.NewExecutor(ctx2)

	e1.Tick() // Running
	e2.Tick() // Running (independent)
	e1.Tick() // Success
	e2.Tick() // Success

	if v := MustGet[int](ctx1.BB, "n"); v != 1 {
		t.Fatalf("e1: want n=1, got %d", v)
	}
	if v := MustGet[int](ctx2.BB, "n"); v != 1 {
		t.Fatalf("e2: want n=1, got %d", v)
	}
}

// ==================== Logger ====================

func TestContext_Log_NilSafe(t *testing.T) {
	// nil Logger → no panic
	ctx := testCtx()
	ctx.Log("should not panic %d", 42)

	// nil Context → no panic
	var nilCtx *Context
	nilCtx.Log("should not panic")
}

func TestContext_Log_Captures(t *testing.T) {
	var captured string
	ctx := testCtx()
	ctx.Logger = loggerFunc(func(format string, args ...any) {
		captured = fmt.Sprintf(format, args...)
	})
	ctx.Log("hello %d", 42)
	if captured != "hello 42" {
		t.Fatalf("want 'hello 42', got %q", captured)
	}
}

type loggerFunc func(string, ...any)

func (f loggerFunc) Printf(format string, args ...any) { f(format, args...) }

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

// ==================== Tracer ====================

func tracedCtx() *Context {
	return &Context{BB: NewBlackboard(), Bus: NewEventBus(), Tracer: NewTracer()}
}

func TestTracer_GoldenTrace(t *testing.T) {
	ctx := tracedCtx()
	ctx.BB.Set("go", false)

	// Selector [Condition(go), Action(fallback)]
	tree := NewTree("test", NewSelector(
		NewSequence(
			NewCondition("go", "eq", true),
			NewAction("attack", func(_ *Context) Status { return Success }),
		),
		NewAction("patrol", func(_ *Context) Status { return Success }),
	))
	exec := tree.NewExecutor(ctx)
	exec.Tick()

	// 验证 trace 包含关键路径
	entries := ctx.Tracer.FrameEntries(1)
	exits := filterExits(entries)

	// Condition 失败 → Sequence 失败 → patrol 成功 → Selector 成功
	if len(exits) < 4 {
		t.Fatalf("want >= 4 exit entries, got %d", len(exits))
	}
	assertExit(t, exits[0], "Condition(go eq true)", Failure)
	assertExit(t, exits[1], "Sequence", Failure)
	assertExit(t, exits[2], "Action(patrol)", Success)
	assertExit(t, exits[3], "Selector", Success)
}

func TestTracer_RunningPath(t *testing.T) {
	ctx := tracedCtx()
	ctx.BB.Set("_s", 0)
	tree := NewTree("test", NewSequence(
		statusAction(Success),
		bbStepAction("_s", Running, Success),
	))
	exec := tree.NewExecutor(ctx)
	exec.Tick()

	if len(ctx.Tracer.RunningPath) != 2 {
		t.Fatalf("running path: want 2 nodes, got %v", ctx.Tracer.RunningPath)
	}
	if ctx.Tracer.RunningPath[0] != "Sequence" {
		t.Fatalf("running path[0]: want Sequence, got %s", ctx.Tracer.RunningPath[0])
	}
}

func TestTracer_InterruptTrace(t *testing.T) {
	ctx := tracedCtx()
	child := NewAction("channel", func(_ *Context) Status { return Running })
	intr := NewInterrupt("stun", child)
	tree := NewTree("test", intr)
	exec := tree.NewExecutor(ctx)

	exec.Tick() // Running
	ctx.Bus.Emit("stun", nil)
	exec.Tick() // Interrupted

	// 查找 Interrupt 事件
	found := false
	for _, e := range ctx.Tracer.FrameEntries(2) {
		if e.Type == TraceInterrupt {
			found = true
			if e.Detail != "event=stun" {
				t.Fatalf("interrupt detail: want 'event=stun', got %q", e.Detail)
			}
		}
	}
	if !found {
		t.Fatal("should have TraceInterrupt entry")
	}
}

func TestTracer_ResetTrace(t *testing.T) {
	ctx := tracedCtx()
	child := NewAction("act", func(_ *Context) Status { return Running })
	tree := NewTree("test", child)
	exec := tree.NewExecutor(ctx)
	exec.Tick()
	exec.Reset()

	found := false
	for _, e := range ctx.Tracer.Entries {
		if e.Type == TraceReset && e.Node == "Action(act)" {
			found = true
		}
	}
	if !found {
		t.Fatal("should have TraceReset for Action(act)")
	}
}

func TestTracer_Snapshot(t *testing.T) {
	ctx := tracedCtx()
	ctx.BB.Set("hp", 100)
	ctx.BB.Set("_s", 0)

	tree := NewTree("boss", bbStepAction("_s", Running, Success))
	exec := tree.NewExecutor(ctx)
	exec.Tick()

	snap := exec.Snapshot()
	if snap.Frame != 1 {
		t.Fatalf("frame: want 1, got %d", snap.Frame)
	}
	if snap.Status != Running {
		t.Fatalf("status: want Running, got %s", snap.Status)
	}
	if snap.BB["hp"] != 100 {
		t.Fatalf("BB hp: want 100, got %v", snap.BB["hp"])
	}
}

func TestTracer_DumpText(t *testing.T) {
	ctx := tracedCtx()
	tree := NewTree("test", NewSelector(
		statusAction(Failure),
		statusAction(Success),
	))
	exec := tree.NewExecutor(ctx)
	exec.Tick()

	text := ctx.Tracer.DumpFrameText(1)
	if !strings.Contains(text, "Selector") {
		t.Fatal("dump should contain Selector")
	}
	if !strings.Contains(text, "Success") {
		t.Fatal("dump should contain Success")
	}
}

func TestTracer_DumpJSON(t *testing.T) {
	ctx := tracedCtx()
	tree := NewTree("test", statusAction(Success))
	exec := tree.NewExecutor(ctx)
	exec.Tick()

	data, err := ctx.Tracer.DumpJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("JSON dump should not be empty")
	}
}

func TestTracer_PreemptTrace(t *testing.T) {
	ctx := tracedCtx()
	ctx.BB.Set("_hi", 0)

	tree := NewTree("test", NewReactiveSelector(
		bbStepAction("_hi", Failure, Success),
		NewAction("low", func(_ *Context) Status { return Running }),
	))
	exec := tree.NewExecutor(ctx)
	exec.Tick() // hi=Failure, low=Running
	exec.Tick() // hi=Success, preempt low

	found := false
	for _, e := range ctx.Tracer.FrameEntries(2) {
		if e.Type == TracePreempt {
			found = true
		}
	}
	if !found {
		t.Fatal("should have TracePreempt entry")
	}
}

// helpers
func filterExits(entries []TraceEntry) []TraceEntry {
	var result []TraceEntry
	for _, e := range entries {
		if e.Type == TraceExit {
			result = append(result, e)
		}
	}
	return result
}

func assertExit(t *testing.T, e TraceEntry, node string, status Status) {
	t.Helper()
	if e.Node != node || e.Status != status {
		t.Fatalf("want Exit(%s=%s), got Exit(%s=%s)", node, status, e.Node, e.Status)
	}
}
