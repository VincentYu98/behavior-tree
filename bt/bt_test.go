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

// sequenceAction 按顺序依次返回给定的状态，到末尾后一直返回最后一个
func sequenceAction(statuses ...Status) *Action {
	i := 0
	return NewAction("test", func(_ *Context) Status {
		s := statuses[i]
		if i < len(statuses)-1 {
			i++
		}
		return s
	})
}

func countAction(s Status) (*Action, *int) {
	count := 0
	a := NewAction("test", func(_ *Context) Status { count++; return s })
	return a, &count
}

// ==================== Sequence ====================

func TestSequence_AllSuccess(t *testing.T) {
	ctx := testCtx()
	seq := NewSequence(statusAction(Success), statusAction(Success), statusAction(Success))
	if s := seq.Tick(ctx); s != Success {
		t.Fatalf("want Success, got %s", s)
	}
}

func TestSequence_FailAtSecond(t *testing.T) {
	ctx := testCtx()
	_, cnt := countAction(Success)
	third, cnt3 := countAction(Success)
	seq := NewSequence(statusAction(Success), statusAction(Failure), third)
	if s := seq.Tick(ctx); s != Failure {
		t.Fatalf("want Failure, got %s", s)
	}
	_ = cnt
	if *cnt3 != 0 {
		t.Fatalf("third child should not have been ticked, got %d", *cnt3)
	}
}

func TestSequence_RunningResume(t *testing.T) {
	ctx := testCtx()
	first, cnt1 := countAction(Success)
	second := sequenceAction(Running, Success)
	seq := NewSequence(first, second)

	if s := seq.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}
	if *cnt1 != 1 {
		t.Fatalf("first should be ticked once, got %d", *cnt1)
	}

	// Memory: resumes from second, skips first
	if s := seq.Tick(ctx); s != Success {
		t.Fatalf("tick2: want Success, got %s", s)
	}
	if *cnt1 != 1 {
		t.Fatalf("first should still be 1 (memory), got %d", *cnt1)
	}
}

// ==================== Selector ====================

func TestSelector_AllFail(t *testing.T) {
	ctx := testCtx()
	sel := NewSelector(statusAction(Failure), statusAction(Failure))
	if s := sel.Tick(ctx); s != Failure {
		t.Fatalf("want Failure, got %s", s)
	}
}

func TestSelector_SuccessAtSecond(t *testing.T) {
	ctx := testCtx()
	sel := NewSelector(statusAction(Failure), statusAction(Success))
	if s := sel.Tick(ctx); s != Success {
		t.Fatalf("want Success, got %s", s)
	}
}

func TestSelector_RunningResume(t *testing.T) {
	ctx := testCtx()
	first, cnt1 := countAction(Failure)
	second := sequenceAction(Running, Success)
	sel := NewSelector(first, second)

	if s := sel.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}
	if s := sel.Tick(ctx); s != Success {
		t.Fatalf("tick2: want Success, got %s", s)
	}
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
	}, func() {
		resetCalled = true
	})

	// child[0]=Failure first, then Success
	hiPri := sequenceAction(Failure, Success)
	rs := NewReactiveSelector(hiPri, lowPri)

	// Tick 1: hiPri=Failure, lowPri=Running → Running
	if s := rs.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}

	// Tick 2: hiPri=Success → preempt, reset lowPri
	if s := rs.Tick(ctx); s != Success {
		t.Fatalf("tick2: want Success, got %s", s)
	}
	if !resetCalled {
		t.Fatal("lowPri should have been Reset on preemption")
	}
}

// ==================== ReactiveSequence ====================

func TestReactiveSequence_ConditionFails(t *testing.T) {
	ctx := testCtx()
	ctx.BB.Set("ok", true)
	resetCalled := false
	action := NewActionWithReset("act", func(_ *Context) Status {
		return Running
	}, func() { resetCalled = true })

	cond := NewCondition("ok", "eq", true)
	rs := NewReactiveSequence(cond, action)

	// Tick 1: cond=Success, action=Running
	if s := rs.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}

	// Condition becomes false
	ctx.BB.Set("ok", false)
	// Tick 2: cond=Failure → abort action
	if s := rs.Tick(ctx); s != Failure {
		t.Fatalf("tick2: want Failure, got %s", s)
	}
	if !resetCalled {
		t.Fatal("action should have been Reset when condition failed")
	}
}

// ==================== Repeater ====================

func TestRepeater_ResetsChildBetweenIterations(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Success
	}, func() { resetCount++ })

	rep := NewRepeater(3, child)
	if s := rep.Tick(ctx); s != Success {
		t.Fatalf("want Success, got %s", s)
	}
	// 3 iterations, each followed by Reset (including last, to leave child clean)
	if resetCount != 3 {
		t.Fatalf("want 3 resets (every iteration), got %d", resetCount)
	}
}

func TestRepeater_RunningResume(t *testing.T) {
	ctx := testCtx()
	child := sequenceAction(Success, Running, Success)
	rep := NewRepeater(3, child)

	// Tick 1: iter0=Success, reset, iter1=Running
	if s := rep.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}
	// Tick 2: resume iter1=Success, reset, iter2=Success → done
	if s := rep.Tick(ctx); s != Success {
		t.Fatalf("tick2: want Success, got %s", s)
	}
}

// ==================== UntilFail ====================

func TestUntilFail_SingleStepPerTick(t *testing.T) {
	ctx := testCtx()
	tickCount := 0
	child := NewAction("c", func(_ *Context) Status {
		tickCount++
		return Success
	})

	uf := NewUntilFail(child)
	// Should return Running (not loop forever), child ticked once
	if s := uf.Tick(ctx); s != Running {
		t.Fatalf("want Running, got %s", s)
	}
	if tickCount != 1 {
		t.Fatalf("child should be ticked once per frame, got %d", tickCount)
	}
}

func TestUntilFail_StopsOnFailure(t *testing.T) {
	ctx := testCtx()
	child := sequenceAction(Success, Success, Failure)
	uf := NewUntilFail(child)

	if s := uf.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}
	if s := uf.Tick(ctx); s != Running {
		t.Fatalf("tick2: want Running, got %s", s)
	}
	if s := uf.Tick(ctx); s != Success {
		t.Fatalf("tick3: want Success (child failed), got %s", s)
	}
}

// ==================== Inverter ====================

func TestInverter(t *testing.T) {
	ctx := testCtx()
	if s := NewInverter(statusAction(Success)).Tick(ctx); s != Failure {
		t.Fatalf("Success→Failure: got %s", s)
	}
	if s := NewInverter(statusAction(Failure)).Tick(ctx); s != Success {
		t.Fatalf("Failure→Success: got %s", s)
	}
	if s := NewInverter(statusAction(Running)).Tick(ctx); s != Running {
		t.Fatalf("Running→Running: got %s", s)
	}
}

// ==================== Interrupt ====================

func TestInterrupt_AbortOnEvent(t *testing.T) {
	ctx := testCtx()
	resetCalled := false
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Running
	}, func() { resetCalled = true })

	intr := NewInterrupt("stun", child)

	// No event: tick child
	if s := intr.Tick(ctx); s != Running {
		t.Fatalf("no event: want Running, got %s", s)
	}

	// Event fires: abort
	ctx.Bus.Emit("stun", nil)
	if s := intr.Tick(ctx); s != Failure {
		t.Fatalf("event: want Failure, got %s", s)
	}
	if !resetCalled {
		t.Fatal("child should be reset on interrupt")
	}
}

// ==================== WaitForEvent ====================

func TestWaitForEvent_WritesToBlackboard(t *testing.T) {
	ctx := testCtx()
	w := NewWaitForEvent("ready", "count")

	if s := w.Tick(ctx); s != Running {
		t.Fatalf("no event: want Running, got %s", s)
	}

	ctx.Bus.Emit("ready", 42)
	if s := w.Tick(ctx); s != Success {
		t.Fatalf("event: want Success, got %s", s)
	}
	val := MustGet[int](ctx.BB, "count")
	if val != 42 {
		t.Fatalf("BB count: want 42, got %d", val)
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
		{"eq", 10.0, Success},  // int vs float64
		{"eq", 5.0, Failure},
		{"ne", 5.0, Success},
		{"lt", 20.0, Success},
		{"lt", 5.0, Failure},
		{"gt", 5.0, Success},
		{"le", 10.0, Success},
		{"ge", 10.0, Success},
		{"ge", 20.0, Failure},
	}
	for _, tc := range tests {
		c := NewCondition("x", tc.op, tc.val)
		if s := c.Tick(ctx); s != tc.want {
			t.Errorf("x=10 %s %v: want %s, got %s", tc.op, tc.val, tc.want, s)
		}
	}
}

func TestCondition_MissingKey(t *testing.T) {
	ctx := testCtx()
	c := NewCondition("nonexistent", "eq", true)
	if s := c.Tick(ctx); s != Failure {
		t.Fatalf("missing key: want Failure, got %s", s)
	}
}

// ==================== Loader Validation ====================

func TestLoader_UnknownAction(t *testing.T) {
	loader := NewLoader()
	data := `{"type":"action","action":"nonexistent"}`
	_, err := loader.LoadJSON([]byte(data))
	if err == nil {
		t.Fatal("should fail on unknown action")
	}
}

func TestLoader_InvalidOp(t *testing.T) {
	loader := NewLoader()
	data := `{"type":"condition","key":"x","op":"invalid","value":1}`
	_, err := loader.LoadJSON([]byte(data))
	if err == nil {
		t.Fatal("should fail on invalid op")
	}
}

func TestLoader_EmptyChildren(t *testing.T) {
	loader := NewLoader()
	data := `{"type":"sequence","children":[]}`
	_, err := loader.LoadJSON([]byte(data))
	if err == nil {
		t.Fatal("should fail on empty children")
	}
}

func TestLoader_RepeaterZeroCount(t *testing.T) {
	loader := NewLoader()
	data := `{"type":"repeater","count":0,"child":{"type":"action","action":"a"}}`
	loader.RegisterAction("a", func(_ *Context) Status { return Success })
	_, err := loader.LoadJSON([]byte(data))
	if err == nil {
		t.Fatal("should fail on zero count")
	}
}

func TestLoader_MissingEvent(t *testing.T) {
	loader := NewLoader()
	data := `{"type":"interrupt","child":{"type":"action","action":"a"}}`
	loader.RegisterAction("a", func(_ *Context) Status { return Success })
	_, err := loader.LoadJSON([]byte(data))
	if err == nil {
		t.Fatal("should fail on missing event")
	}
}

func TestLoader_MissingChild(t *testing.T) {
	loader := NewLoader()
	data := `{"type":"inverter"}`
	_, err := loader.LoadJSON([]byte(data))
	if err == nil {
		t.Fatal("should fail on missing child")
	}
}

func TestLoader_ValidBuild(t *testing.T) {
	loader := NewLoader()
	loader.RegisterAction("noop", func(_ *Context) Status { return Success })

	cfg := NodeConfig{
		Type: "sequence",
		Children: []*NodeConfig{
			{Type: "condition", Key: "hp", Op: "lt", Value: 50.0},
			{Type: "action", Action: "noop"},
		},
	}
	data, _ := json.Marshal(cfg)
	node, err := loader.LoadJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := testCtx()
	ctx.BB.Set("hp", 30)
	if s := node.Tick(ctx); s != Success {
		t.Fatalf("want Success, got %s", s)
	}
}

// ==================== AlwaysSucceed ====================

func TestAlwaysSucceed(t *testing.T) {
	ctx := testCtx()
	if s := NewAlwaysSucceed(statusAction(Failure)).Tick(ctx); s != Success {
		t.Fatalf("Failure→Success: got %s", s)
	}
	if s := NewAlwaysSucceed(statusAction(Running)).Tick(ctx); s != Running {
		t.Fatalf("Running→Running: got %s", s)
	}
}

// ==================== Reset 幂等性回归测试 ====================

// 验证 ReactiveSelector.Reset() 不会对 running 子节点 Reset 两次
func TestReactiveSelector_ResetNoDoubleReset(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Running
	}, func() { resetCount++ })

	rs := NewReactiveSelector(statusAction(Failure), child)
	// child becomes running
	rs.Tick(ctx)
	if resetCount != 0 {
		t.Fatalf("before Reset: want 0 resets, got %d", resetCount)
	}

	rs.Reset()
	if resetCount != 1 {
		t.Fatalf("after Reset: want exactly 1 reset, got %d", resetCount)
	}
}

// 验证 ReactiveSequence.Reset() 不会对 running 子节点 Reset 两次
func TestReactiveSequence_ResetNoDoubleReset(t *testing.T) {
	ctx := testCtx()
	resetCount := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		return Running
	}, func() { resetCount++ })

	rs := NewReactiveSequence(statusAction(Success), child)
	rs.Tick(ctx)

	rs.Reset()
	if resetCount != 1 {
		t.Fatalf("after Reset: want exactly 1 reset, got %d", resetCount)
	}
}

// 验证 Repeater 完成后子树是干净的，再次进入不带脏状态
func TestRepeater_CleanAfterCompletion(t *testing.T) {
	ctx := testCtx()
	round := 0
	child := NewActionWithReset("c", func(_ *Context) Status {
		round++
		return Success
	}, func() { round = 0 })

	rep := NewRepeater(2, child)

	// 第一次运行：2轮，round 被 reset 到 0（最后一轮也 Reset）
	rep.Tick(ctx)
	if round != 0 {
		t.Fatalf("after first run: want round=0 (clean), got %d", round)
	}

	// 第二次运行：应该从 round=0 开始，不是从上次残留状态
	rep.Tick(ctx)
	if round != 0 {
		t.Fatalf("after second run: want round=0 (clean), got %d", round)
	}
}

// 验证 UntilFail 在子节点 Success 后 Reset 子树，下一帧干净启动
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
	}, func() { resetCount++ })

	uf := NewUntilFail(child)

	// Tick 1: child Success → UntilFail returns Running, should Reset child
	if s := uf.Tick(ctx); s != Running {
		t.Fatalf("tick1: want Running, got %s", s)
	}
	if resetCount != 1 {
		t.Fatalf("tick1: want 1 reset after Success round, got %d", resetCount)
	}

	// Tick 2: child Success → Running, another Reset
	if s := uf.Tick(ctx); s != Running {
		t.Fatalf("tick2: want Running, got %s", s)
	}
	if resetCount != 2 {
		t.Fatalf("tick2: want 2 resets, got %d", resetCount)
	}

	// Tick 3: child Failure → UntilFail returns Success, no extra Reset
	if s := uf.Tick(ctx); s != Success {
		t.Fatalf("tick3: want Success, got %s", s)
	}
	if resetCount != 2 {
		t.Fatalf("tick3: want still 2 resets (no reset on Failure exit), got %d", resetCount)
	}
}

// ==================== EventBus: Poll 返回最新、PollAll 返回全部 ====================

func TestPoll_ReturnsLatestEvent(t *testing.T) {
	bus := NewEventBus()
	bus.Emit("dmg", 10)
	bus.Emit("dmg", 25)
	bus.Emit("dmg", 50)

	evt, ok := bus.Poll("dmg")
	if !ok {
		t.Fatal("should find event")
	}
	if evt.Data.(int) != 50 {
		t.Fatalf("Poll should return latest: want 50, got %v", evt.Data)
	}
}

func TestPollAll_ReturnsAllEvents(t *testing.T) {
	bus := NewEventBus()
	bus.Emit("dmg", 10)
	bus.Emit("heal", 5)
	bus.Emit("dmg", 25)

	all := bus.PollAll("dmg")
	if len(all) != 2 {
		t.Fatalf("want 2 dmg events, got %d", len(all))
	}
	if all[0].Data.(int) != 10 || all[1].Data.(int) != 25 {
		t.Fatalf("events out of order: %v", all)
	}
}

// ==================== Nil Context 防御 ====================

func TestCondition_NilContext(t *testing.T) {
	c := NewCondition("x", "eq", 1)
	if s := c.Tick(nil); s != Failure {
		t.Fatalf("nil ctx: want Failure, got %s", s)
	}
	ctx := &Context{} // BB is nil
	if s := c.Tick(ctx); s != Failure {
		t.Fatalf("nil BB: want Failure, got %s", s)
	}
}

func TestWaitForEvent_NilBus(t *testing.T) {
	w := NewWaitForEvent("evt", "key")
	if s := w.Tick(nil); s != Failure {
		t.Fatalf("nil ctx: want Failure, got %s", s)
	}
	ctx := &Context{BB: NewBlackboard()} // Bus is nil
	if s := w.Tick(ctx); s != Failure {
		t.Fatalf("nil Bus: want Failure, got %s", s)
	}
}

func TestInterrupt_NilBus_PassesThrough(t *testing.T) {
	child := statusAction(Success)
	intr := NewInterrupt("evt", child)

	// nil Bus: can't check events, should pass through to child
	ctx := &Context{BB: NewBlackboard()}
	if s := intr.Tick(ctx); s != Success {
		t.Fatalf("nil Bus: want child's Success, got %s", s)
	}
}

// ==================== Blackboard 数值互转 ====================

func TestGet_NumericCoercion(t *testing.T) {
	bb := NewBlackboard()

	// float64 → int (JSON 数字场景)
	bb.Set("hp", 100.0)
	v, ok := Get[int](bb, "hp")
	if !ok || v != 100 {
		t.Fatalf("float64→int: want 100, got %v ok=%v", v, ok)
	}

	// int → float64
	bb.Set("speed", 5)
	f, ok := Get[float64](bb, "speed")
	if !ok || f != 5.0 {
		t.Fatalf("int→float64: want 5.0, got %v ok=%v", f, ok)
	}

	// bool 不做数值转换
	bb.Set("flag", true)
	_, ok = Get[int](bb, "flag")
	if ok {
		t.Fatal("bool→int should fail")
	}

	// string 不做数值转换
	bb.Set("name", "boss")
	_, ok = Get[int](bb, "name")
	if ok {
		t.Fatal("string→int should fail")
	}
}

func TestMustGet_CoercionWorks(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("x", 42.0)

	// MustGet[int] 对 float64 值不再 panic
	v := MustGet[int](bb, "x")
	if v != 42 {
		t.Fatalf("MustGet[int] on float64: want 42, got %d", v)
	}
}

func TestConditionAndGet_ConsistentOnSameKey(t *testing.T) {
	ctx := testCtx()
	// 模拟事件 payload 写入 float64 (JSON 反序列化的典型情况)
	ctx.BB.Set("count", 3.0)

	// Condition 判断成功
	c := NewCondition("count", "eq", 3.0)
	if s := c.Tick(ctx); s != Success {
		t.Fatal("Condition eq 3.0 should succeed")
	}

	// Get[int] 也能读到值（之前会失败）
	v, ok := Get[int](ctx.BB, "count")
	if !ok || v != 3 {
		t.Fatalf("Get[int] on float64 3.0: want 3, got %v ok=%v", v, ok)
	}
}
