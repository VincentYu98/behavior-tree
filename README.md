# Behavior Tree

Go 语言实现的行为树框架，面向游戏 AI 场景设计。支持 JSON 配置化加载、Running 断点续跑、事件驱动打断。

附带一个三阶段 Boss AI 完整示例（炎魔将军），17 帧战斗模拟演示所有核心机制。

## 快速开始

```bash
go run .
```

输出 17 帧 Boss 战斗模拟，覆盖巡逻、追击、连击、蓄力、眩晕打断、事件等待、狂暴连击等场景。

## 框架架构

```
游戏主循环（每帧）
  EventBus.Clear()  →  EventBus.Emit(...)  →  Blackboard.Set(...)  →  tree.Tick()
```

### 节点类型

**组合节点** — 控制子节点的执行流向

| 节点 | 语义 | 说明 |
|------|------|------|
| `Sequence` | `&&` | 从左到右依次执行，全部成功才成功，任一失败则失败 |
| `Selector` | `\|\|` | 从左到右依次尝试，任一成功则成功，全部失败才失败 |
| `ReactiveSelector` | 响应式 `\|\|` | 每帧从头重新评估，高优先级分支可抢占低优先级 Running 分支 |

**装饰节点** — 单子节点，变换执行结果

| 节点 | 说明 |
|------|------|
| `Inverter` | Success ↔ Failure 翻转 |
| `Repeater(N)` | 重复执行子节点 N 次 |
| `AlwaysSucceed` | 吞掉 Failure，强制返回 Success |
| `UntilFail` | 循环执行直到子节点返回 Failure |
| `Interrupt(event)` | 监听事件，触发时 Reset 子树并返回 Failure |

**叶子节点** — 执行具体工作

| 节点 | 说明 |
|------|------|
| `Action` | 包装 `func() Status`，可选 `resetFn` 用于打断时清理 |
| `Condition` | 从 Blackboard 读值比较（支持 eq/ne/lt/gt/le/ge） |
| `WaitForEvent(event)` | 返回 Running 直到事件触发，可将事件数据写入黑板 |

### 基础设施

| 组件 | 说明 |
|------|------|
| `Blackboard` | `map[string]any` 共享黑板，泛型 `Get[T]` / `MustGet[T]` 取值 |
| `EventBus` | 帧内事件发布/订阅，`Emit` → 节点 `Poll` → 帧末 `Clear` |
| `Loader` | JSON 加载器，`RegisterAction` 注册行为函数，JSON 中按名引用 |

### Running 断点续跑

节点返回 `Running` 时，`Sequence` / `Selector` / `Repeater` 记住当前执行位置（`runningIdx`）。下一帧 Tick 时直接从断点恢复，不重复执行已完成的子节点。

### Reset 级联

`ReactiveSelector` 抢占或 `Interrupt` 打断时，调用旧分支的 `Reset()`。Reset 从组合节点逐层向下传递，最终到达 `Action.resetFn()`，清理蓄力计数器等运行状态。

## Boss 示例 — 炎魔将军

行为树结构定义在 `boss_event.json`，行为实现在 `main.go` 的 `registerActions`。

```
ReactiveSelector (根 — 每帧重新评估优先级)
│
├── [P0] 眩晕处理 (stunned == true → play_stun, Running 3帧)
├── [P1] 紧急闪避 (should_dodge == true → dodge_roll + counter_attack)
│
└── [P2] 三阶段战斗 Selector
    ├── Phase3 (HP < 20%) — 残血狂暴
    │   ├── 终极技能: cast_ultimate (Running 4帧蓄力, Interrupt 沉默可打断)
    │   ├── 逃跑回血: flee + heal
    │   ├── 狂暴连击: Repeater(5) × berserk_attack
    │   └── 兜底逃跑
    │
    ├── Phase2 (HP < 60%) — 狂暴
    │   ├── 召唤协同: summon → WaitForEvent("minions_ready") → coordinate
    │   ├── 火焰风暴: fire_storm (需 ready + in_range)
    │   ├── 冲锋: charge (detected + 不在范围 → Inverter)
    │   ├── 强化连击: Repeater(4) × enhanced_attack + war_cry
    │   └── 兜底巡逻
    │
    └── Phase1 (HP >= 60%) — 正常战斗
        ├── 火球术: channel_fireball (Running 3帧蓄力, Interrupt 沉默可打断)
        ├── 追击: chase (detected + 不在范围)
        ├── 普通连击: Repeater(3) × attack + backstep
        └── 兜底巡逻
```

### 事件驱动示例

| 机制 | 事件 | 效果 |
|------|------|------|
| Interrupt | `on_silenced` | 沉默打断火球/终极蓄力，技能进入冷却 |
| ReactiveSelector | `stunned=true` | 眩晕抢占任何战斗行为，级联 Reset 子树 |
| WaitForEvent | `minions_ready` | 召唤小怪后等待就位，事件数据写入黑板 |

## 项目结构

```
bt/                     框架层（通用，无业务逻辑）
├── node.go             Status 枚举 + Node 接口
├── action.go           Action 叶子节点
├── condition.go        Condition 叶子节点
├── wait_event.go       WaitForEvent 叶子节点
├── sequence.go         Sequence 组合节点
├── selector.go         Selector 组合节点
├── reactive.go         ReactiveSelector
├── decorator.go        Inverter / Repeater / AlwaysSucceed / UntilFail
├── interrupt.go        Interrupt 装饰节点
├── blackboard.go       Blackboard 共享黑板
├── event.go            EventBus 事件总线
└── loader.go           JSON 加载器 + Action 注册表

boss.json               基础版 Boss 行为树配置（无事件）
boss_event.json         事件驱动版 Boss 行为树配置
main.go                 Boss 行为实现 + 17帧战斗模拟
```

## JSON 配置格式

```jsonc
{
  "type": "selector",              // 节点类型
  "name": "可选标签",               // 调试用
  "children": [                    // 组合节点的子节点
    {"type": "condition", "key": "hp_percent", "op": "lt", "value": 20},
    {"type": "action", "action": "registered_name"},
    {"type": "repeater", "count": 3, "child": {"type": "action", "action": "attack"}},
    {"type": "inverter", "child": {"type": "condition", "key": "in_range", "op": "eq", "value": true}},
    {"type": "interrupt", "event": "on_silenced", "child": {"type": "action", "action": "channel"}},
    {"type": "wait_event", "event": "minions_ready", "write_to": "minion_count"},
    {"type": "reactive_selector", "children": [...]},
    {"type": "always_succeed", "child": {...}},
    {"type": "until_fail", "child": {...}}
  ]
}
```

## License

MIT
