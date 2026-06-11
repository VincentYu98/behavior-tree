# Behavior Tree

Go 语言行为树框架，面向游戏 AI 设计。

支持 JSON 配置化、Running 断点续跑、事件驱动打断、多实体树模板复用。附带三阶段 Boss AI 完整示例。

## 快速开始

```bash
go run .
```

输出 17 帧 Boss 战斗模拟，覆盖巡逻、追击、连击、蓄力、沉默打断、眩晕抢占、事件等待、阶段切换、狂暴连击。

```bash
go test ./bt/ -v
```

运行 45 个单元测试，覆盖所有节点语义、多实体隔离、nil 安全、数值转换、Executor 生命周期。

## 使用方式

```go
// 1. 注册行为
loader := bt.NewLoader()
loader.RegisterAction("patrol", func(ctx *bt.Context) bt.Status {
    ctx.Log("巡逻中...\n")
    return bt.Success
})

// 2. 加载树（JSON 配置 → 不可变模板）
tree, err := loader.LoadFile("boss.json")

// 3. 为每个实体创建执行器
ctx := &bt.Context{
    BB:     bt.NewBlackboard(),
    Bus:    bt.NewEventBus(),
    Delta:  0.016,
    Logger: bt.FmtLogger{},
}
exec := tree.NewExecutor(ctx)

// 4. 游戏主循环
for {
    ctx.Bus.Clear()
    ctx.Bus.Emit("on_hit", damage)
    ctx.BB.Set("hp_percent", hp)
    ctx.Delta = dt

    status := exec.Tick()
    // status: Success / Failure / Running
}
```

同一棵 `Tree` 可被多个 `Executor` 共享。节点本身无可变状态，所有运行态存于各自的 `Context` 中。

## 架构总览

```
应用层 (main.go)
  └── registerActions + 游戏循环
      │
入口层 (tree.go)
  ├── Tree           不可变树模板，Loader.LoadFile 构建
  └── Executor       每实体执行器，Tick / Reset / 统计
      │
节点层
  ├── 组合: Sequence / Selector / ReactiveSelector / ReactiveSequence
  ├── 装饰: Inverter / Repeater / AlwaysSucceed / UntilFail / Interrupt
  └── 叶子: Action / Condition / WaitForEvent
      │
基础设施层
  ├── Context        执行上下文 (BB + Bus + Delta + Logger + nodeState)
  ├── Blackboard     共享数据黑板 (nil-safe, 泛型 Get, 数值互转)
  ├── EventBus       帧内事件总线 (Poll 最新 / PollAll 全部)
  ├── Logger         调试输出接口 (FmtLogger / NilLogger)
  └── Loader         JSON 构建 + 加载期校验
```

## 节点类型

### 组合节点

| 节点 | 语义 | 说明 |
|------|------|------|
| `Sequence` | `&&` (Memory) | 从左到右执行，全部成功才成功。Running 后从断点恢复 |
| `Selector` | `\|\|` (Memory) | 从左到右尝试，任一成功即成功。Running 后从断点恢复 |
| `ReactiveSelector` | 响应式 `\|\|` | 每帧从头评估，高优先级可抢占 Running 分支（级联 Reset） |
| `ReactiveSequence` | 响应式 `&&` | 每帧从头评估，前置条件失效时中止后续节点 |

### 装饰节点

| 节点 | 说明 |
|------|------|
| `Inverter` | Success ↔ Failure 翻转，Running 穿透 |
| `Repeater(N)` | 重复 N 次，迭代间 Reset 子树 |
| `AlwaysSucceed` | 吞掉 Failure，Running 穿透 |
| `UntilFail` | 每帧单步执行，Success 后 Reset 子树为下帧准备，Failure 时结束 |
| `Interrupt(event)` | 监听事件，触发时 Reset 子树并返回 Failure |

### 叶子节点

| 节点 | 说明 |
|------|------|
| `Action` | 包装 `func(*Context) Status`，可选 `resetFn func(*Context)` 用于打断清理 |
| `Condition` | 从 BB 读值比较，支持 `eq` `ne` `lt` `gt` `le` `ge` |
| `WaitForEvent` | 返回 Running 直到事件触发，可将事件 Data 写入 BB |

## 核心机制

### Running 断点续跑

```
帧1: Sequence → A:Success → B:Running → 记住断点, 返回 Running
帧2: Sequence → 跳过 A → B:Success → C:Success → 返回 Success
```

节点运行态（runningIdx）存于 `Context` 内部，节点对象无可变字段。

### Reset 语义

| 场景 | 触发者 | 调用 resetFn？ | 说明 |
|------|--------|--------------|------|
| Running 被放弃 | ReactiveSelector 抢占 / Interrupt 事件 / 显式 Reset | 是 | 中断清理，级联到子树 |
| 正常终态返回 | 无 | 否 | 每个节点自清自己的 nodeState |
| Repeater 迭代间 | Repeater | 是 | 为下一轮提供干净起点 |

**Action 约定**：fn 在返回 Success/Failure 前清理自己的 BB 内部状态。resetFn 仅处理 Running 被中断时的清理，不应清理"结果类" BB key。

### 事件系统

```go
// 发射（帧初）
ctx.Bus.Emit("on_hit", damage)

// 消费（节点 Tick 中）
evt, ok := ctx.Bus.Poll("on_hit")     // 最新一条（latest wins）
all := ctx.Bus.PollAll("on_hit")       // 本帧全部

// 清理（帧末）
ctx.Bus.Clear()
```

### Blackboard 数值互转

`Get[T]` 在严格类型断言失败时尝试数值互转，和 Condition 的比较行为一致：

```go
bb.Set("hp", 100.0)          // JSON 反序列化为 float64
v, ok := bt.Get[int](bb, "hp") // ok=true, v=100（整值浮点→int）

bb.Set("x", 3.9)
_, ok = bt.Get[int](bb, "x")   // ok=false（非整值，拒绝截断）

bb.Set("big", 1e18)
_, ok = bt.Get[int32](bb, "big") // ok=false（超出 int32 范围）
```

## Boss 示例 — 炎魔将军

配置：`boss_event.json` | 行为：`main.go` registerActions

```
ReactiveSelector (根 — 每帧重新评估优先级)
│
├── [P0] 眩晕处理
│   stunned == true → play_stun (Running 3帧, 结束后清除 stunned)
│
├── [P1] 紧急闪避
│   should_dodge == true → dodge_roll + counter_attack
│
└── [P2] 三阶段战斗 (ReactiveSelector — HP 变化时自动切阶段)
    │
    ├── Phase3: HP < 20% — 残血狂暴
    │   ├── 终极: cast_ultimate (Running 4帧蓄力, Interrupt 沉默可打断)
    │   ├── 逃跑回血: flee + heal
    │   ├── 狂暴连击: Repeater(5) x berserk_attack
    │   └── 兜底逃跑
    │
    ├── Phase2: HP < 60% — 狂暴
    │   ├── 召唤协同: summon → WaitForEvent("minions_ready") → coordinate
    │   ├── 火焰风暴: fire_storm
    │   ├── 冲锋: charge (Inverter: 不在范围才冲)
    │   ├── 强化连击: Repeater(4) x enhanced_attack + war_cry
    │   └── 兜底巡逻
    │
    └── Phase1: HP >= 60% — 正常战斗
        ├── 火球术: channel_fireball (Running 3帧蓄力, Interrupt 沉默可打断)
        ├── 追击: chase
        ├── 普通连击: Repeater(3) x attack + backstep
        └── 兜底巡逻
```

### 17 帧战斗时间线

| 帧 | 场景 | Boss 行为 | 关键机制 |
|---|------|----------|---------|
| 1 | 无人 | 巡逻 | Phase1 兜底 |
| 2-3 | 发现玩家, 火球就绪 | 蓄力火球 | Running 断点续跑 |
| 4 | 玩家沉默 | 火球被打断 → 追击 | **Interrupt** |
| 5 | 火球冷却 | 追击 | |
| 6 | 玩家在范围 | 连击 x3 + 后跳 | Repeater |
| 7 | HP < 60% | Phase2! 召唤小怪, 等待就位 | **WaitForEvent** Running |
| 8 | 小怪就位事件 | 协同攻击 | WaitForEvent → BB 写入 |
| 9 | 火焰风暴就绪 | AOE | |
| 10 | 继续战斗 | 强化连击 x4 + 战吼 | Repeater |
| 11-12 | HP < 20% | Phase3! 终极蓄力 | Running 断点续跑 |
| 13 | 被眩晕 | 终极被打断! 眩晕中 | **ReactiveSelector 抢占** |
| 14 | 继续眩晕 | 眩晕中 | Running |
| 15 | 眩晕结束 | 逃跑回血 | |
| 16 | 玩家重击 | 闪避 + 反击 | ReactiveSelector P1 分支 |
| 17 | 回血冷却 | 狂暴连击 x5 | Repeater |

## JSON 配置格式

```jsonc
{
  "type": "selector",                // 节点类型
  "name": "标签(可选)",               // 调试 / Tree.Name()
  "children": [                      // 组合节点的子节点列表
    // 条件
    {"type": "condition", "key": "hp", "op": "lt", "value": 20},

    // 行为（引用注册的 action 名）
    {"type": "action", "action": "attack"},

    // 装饰
    {"type": "repeater", "count": 3, "child": {...}},
    {"type": "inverter", "child": {...}},
    {"type": "always_succeed", "child": {...}},
    {"type": "until_fail", "child": {...}},
    {"type": "interrupt", "event": "on_silenced", "child": {...}},

    // 事件等待
    {"type": "wait_event", "event": "minions_ready", "write_to": "count"},

    // 响应式组合
    {"type": "reactive_selector", "children": [...]},
    {"type": "reactive_sequence", "children": [...]}
  ]
}
```

加载时自动校验：必填字段、合法操作符、action 引用存在性、children 非空、count > 0。

## 项目结构

```
bt/
├── node.go             Node 接口 + Context + Status（含生命周期语义文档）
├── tree.go             Tree (不可变模板) + Executor (每实体执行器)
├── logger.go           Logger 接口 + FmtLogger / NilLogger
├── action.go           Action 叶子节点 (fn + resetFn)
├── condition.go        Condition 叶子节点 (BB 比较, 6 种操作符)
├── wait_event.go       WaitForEvent 叶子节点
├── sequence.go         Sequence (Memory &&)
├── selector.go         Selector (Memory ||)
├── reactive.go         ReactiveSelector + ReactiveSequence
├── decorator.go        Inverter / Repeater / AlwaysSucceed / UntilFail
├── interrupt.go        Interrupt 装饰节点
├── blackboard.go       Blackboard (nil-safe, 零值安全, 泛型数值互转, 范围校验)
├── event.go            EventBus (Poll 最新 / PollAll 全部)
├── loader.go           JSON → *Tree (加载期校验)
└── bt_test.go          45 个单元测试

boss_event.json         三阶段 Boss AI 行为树配置
main.go                 Boss 行为实现 + 17帧战斗模拟 (使用 Tree/Executor/Logger)
```

## 测试覆盖

| 类别 | 测试数 | 覆盖内容 |
|------|--------|---------|
| 组合节点语义 | 7 | Sequence/Selector 成功/失败/Running 恢复, Reactive 抢占/条件失效 |
| 装饰节点语义 | 7 | Repeater 迭代间 Reset/兄弟可见性/re-entry, UntilFail 单步/Reset, Inverter, AlwaysSucceed |
| 事件驱动 | 4 | Interrupt 打断, WaitForEvent 写 BB, Poll 最新, PollAll 全部 |
| Condition | 2 | 全操作符, nil Context |
| Blackboard | 5 | nil 指针, 零值, 数值互转, 非整值拒绝, 范围溢出 |
| Loader 校验 | 4 | 未知 action, 非法 op, 空 children, 构建+执行 |
| Reset 幂等 | 3 | ReactiveSelector/Sequence 不双重 Reset, 终态不触发 resetFn |
| 多实体 | 2 | 共享树隔离 (直接 Node + Executor) |
| Tree/Executor | 3 | 生命周期, Reset 级联, 多实例统计 |
| Logger | 2 | nil 安全, 输出捕获 |
| Action 模式 | 2 | 完成前清理 re-entry, 结果对兄弟可见 |

## License

MIT
