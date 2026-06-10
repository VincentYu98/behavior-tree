package main

import (
	"behavior-tree/bt"
	"fmt"
	"log"
	"strings"
)

func registerActions(loader *bt.Loader, bb *bt.Blackboard) {
	comboCount := 0
	currentPhase := 0

	// ========== 阶段切换 ==========
	loader.RegisterAction("enter_phase1", func() bt.Status {
		if currentPhase != 1 {
			currentPhase = 1
			fmt.Println("  ⚔ [阶段] 炎魔将军 - Phase1 正常战斗")
		}
		return bt.Success
	})
	loader.RegisterAction("enter_phase2", func() bt.Status {
		if currentPhase != 2 {
			currentPhase = 2
			fmt.Println("  ⚔ [阶段] 炎魔将军 - Phase2 狂暴!")
			fmt.Println("    \"你们这些蝼蚁...让我认真起来!\"")
		}
		return bt.Success
	})
	loader.RegisterAction("enter_phase3", func() bt.Status {
		if currentPhase != 3 {
			currentPhase = 3
			fmt.Println("  ⚔ [阶段] 炎魔将军 - Phase3 残血狂暴!")
			fmt.Println("    \"同归于尽吧!\"")
		}
		return bt.Success
	})

	// ========== 事件驱动专属：眩晕 / 闪避 ==========
	stunTicks := 0
	loader.RegisterActionWithReset("play_stun", func() bt.Status {
		stunTicks++
		if stunTicks < 3 {
			fmt.Printf("  ★ 炎魔将军被眩晕中... (%d/3)\n", stunTicks)
			return bt.Running
		}
		fmt.Println("  ★ 炎魔将军眩晕结束, 恢复行动!")
		stunTicks = 0
		bb.Set("stunned", false)
		return bt.Success
	}, func() { stunTicks = 0 })

	loader.RegisterAction("dodge_roll", func() bt.Status {
		fmt.Println("  ★ 炎魔将军紧急翻滚闪避!")
		bb.Set("should_dodge", false)
		return bt.Success
	})
	loader.RegisterAction("counter_attack", func() bt.Status {
		fmt.Println("  ★ 炎魔将军反击一刀!")
		return bt.Success
	})

	// ========== Phase1: 正常战斗 ==========
	loader.RegisterAction("patrol", func() bt.Status {
		fmt.Println("  → 炎魔将军在领地内巡逻...")
		return bt.Success
	})
	loader.RegisterAction("chase", func() bt.Status {
		fmt.Println("  → 炎魔将军向玩家冲去!")
		return bt.Success
	})

	// 火球术：蓄力2帧，可被沉默打断
	fireballTicks := 0
	loader.RegisterActionWithReset("channel_fireball", func() bt.Status {
		fireballTicks++
		if fireballTicks < 3 {
			fmt.Printf("  → 炎魔将军蓄力【火球术】... (%d/3)\n", fireballTicks)
			return bt.Running
		}
		fireballTicks = 0
		bb.Set("fireball_ready", false)
		fmt.Println("  → 炎魔将军释放【火球术】! 轰!")
		return bt.Success
	}, func() {
		if fireballTicks > 0 {
			fmt.Println("    !! 火球蓄力被打断!")
			fireballTicks = 0
		}
	})

	loader.RegisterAction("attack", func() bt.Status {
		comboCount++
		fmt.Printf("  → 普通攻击 (第%d击)\n", comboCount)
		return bt.Success
	})
	loader.RegisterAction("backstep", func() bt.Status {
		fmt.Println("  → 后跳拉开距离!")
		comboCount = 0
		return bt.Success
	})

	// ========== Phase2: 狂暴 ==========
	loader.RegisterAction("summon_minions", func() bt.Status {
		fmt.Println("  → 炎魔将军召唤3只火焰小鬼!")
		bb.Set("minions_summoned", true)
		return bt.Success
	})
	loader.RegisterAction("coordinate_attack", func() bt.Status {
		count, _ := bt.Get[int](bb, "minion_count")
		fmt.Printf("  → 炎魔将军与%d只小鬼发起协同攻击!\n", count)
		return bt.Success
	})
	loader.RegisterAction("fire_storm", func() bt.Status {
		fmt.Println("  → 炎魔将军施放【火焰风暴】! 全屏AOE!")
		bb.Set("fire_storm_ready", false)
		return bt.Success
	})
	loader.RegisterAction("charge", func() bt.Status {
		fmt.Println("  → 炎魔将军冲锋突进!")
		bb.Set("player_in_range", true)
		return bt.Success
	})
	loader.RegisterAction("enhanced_attack", func() bt.Status {
		comboCount++
		fmt.Printf("  → 强化攻击 (第%d击) 附带灼烧!\n", comboCount)
		return bt.Success
	})
	loader.RegisterAction("war_cry", func() bt.Status {
		fmt.Println("  → 炎魔将军发出战吼! 降低玩家防御!")
		comboCount = 0
		return bt.Success
	})

	// ========== Phase3: 残血 ==========
	// 终极技能：蓄力3帧，可被沉默打断
	ultimateTicks := 0
	loader.RegisterActionWithReset("cast_ultimate", func() bt.Status {
		ultimateTicks++
		if ultimateTicks < 4 {
			fmt.Printf("  → 炎魔将军蓄力【炎灭天地】... (%d/4) [Running]\n", ultimateTicks)
			return bt.Running
		}
		ultimateTicks = 0
		bb.Set("ultimate_ready", false)
		fmt.Println("  → 炎魔将军释放【炎灭天地】!!! 全屏毁灭!")
		return bt.Success
	}, func() {
		if ultimateTicks > 0 {
			fmt.Println("    !! 终极技能蓄力被打断! 技能进入冷却!")
			ultimateTicks = 0
			bb.Set("ultimate_ready", false)
		}
	})

	loader.RegisterAction("flee", func() bt.Status {
		fmt.Println("  → 炎魔将军向后撤退!")
		return bt.Success
	})
	loader.RegisterAction("heal", func() bt.Status {
		fmt.Println("  → 炎魔将军吞噬火焰恢复生命!")
		bb.Set("heal_ready", false)
		return bt.Success
	})
	loader.RegisterAction("berserk_attack", func() bt.Status {
		comboCount++
		fmt.Printf("  → 狂暴攻击 (第%d击) 伤害翻倍!\n", comboCount)
		return bt.Success
	})
}

type Tick struct {
	desc   string
	events []Event       // 本帧事件
	setup  func(bb *bt.Blackboard) // 设置黑板状态，nil 表示续跑帧
}

type Event struct {
	name string
	data any
}

func main() {
	bb := bt.NewBlackboard()
	bus := bt.NewEventBus()
	loader := bt.NewLoader(bb, bus)
	registerActions(loader, bb)

	tree, err := loader.LoadFile("boss_event.json")
	if err != nil {
		log.Fatalf("加载行为树失败: %v", err)
	}

	ticks := []Tick{
		// ---- Phase 1: 正常战斗 ----
		{
			desc: "玩家进入副本, Boss 巡逻",
			setup: func(bb *bt.Blackboard) {
				bb.Set("hp_percent", 100)
				bb.Set("stunned", false)
				bb.Set("should_dodge", false)
				bb.Set("player_detected", false)
				bb.Set("player_in_range", false)
				bb.Set("fireball_ready", false)
				bb.Set("fire_storm_ready", false)
				bb.Set("heal_ready", false)
				bb.Set("ultimate_ready", false)
				bb.Set("minions_summoned", false)
			},
		},
		{
			desc: "远处发现玩家, 火球就绪, 开始蓄力",
			setup: func(bb *bt.Blackboard) {
				bb.Set("player_detected", true)
				bb.Set("fireball_ready", true)
			},
		},
		{desc: "(续) 火球蓄力中..."},
		{
			desc:   "玩家沉默了 Boss! 火球蓄力被打断!",
			events: []Event{{name: "on_silenced"}},
		},
		{
			desc: "火球冷却, 玩家还在远处, 追击",
			setup: func(bb *bt.Blackboard) {
				bb.Set("fireball_ready", false)
			},
		},
		{
			desc: "玩家进入范围, 连击",
			setup: func(bb *bt.Blackboard) {
				bb.Set("hp_percent", 90)
				bb.Set("player_in_range", true)
			},
		},

		// ---- Phase 2: 狂暴 ----
		{
			desc: "HP 跌破 60%, Phase2! 召唤小怪, 等待小怪就位...",
			setup: func(bb *bt.Blackboard) {
				bb.Set("hp_percent", 55)
				bb.Set("minions_summoned", false)
			},
		},
		{
			desc:   "事件: 小怪就位! 发起协同攻击!",
			events: []Event{{name: "minions_ready", data: 3}},
		},
		{
			desc: "火焰风暴就绪, 释放 AOE",
			setup: func(bb *bt.Blackboard) {
				bb.Set("hp_percent", 50)
				bb.Set("fire_storm_ready", true)
			},
		},
		{
			desc: "强化连击",
			setup: func(bb *bt.Blackboard) {
				bb.Set("hp_percent", 40)
			},
		},

		// ---- Phase 3: 残血 + 事件打断演示 ----
		{
			desc: "HP 跌破 20%, Phase3! 终极技能蓄力!",
			setup: func(bb *bt.Blackboard) {
				bb.Set("hp_percent", 15)
				bb.Set("ultimate_ready", true)
			},
		},
		{desc: "(续) 终极蓄力中..."},
		{
			desc:   "Boss 被眩晕了! ReactiveSelector 抢占, 终极被打断!",
			events: []Event{{name: "on_stunned"}},
			setup: func(bb *bt.Blackboard) {
				bb.Set("stunned", true)
			},
		},
		{desc: "(续) 眩晕中..."},
		{
			desc: "眩晕结束, 回血就绪, 逃跑回血",
			setup: func(bb *bt.Blackboard) {
				bb.Set("heal_ready", true)
			},
		},
		{
			desc: "玩家重击! 紧急闪避+反击!",
			events: []Event{{name: "on_heavy_hit"}},
			setup: func(bb *bt.Blackboard) {
				bb.Set("should_dodge", true)
			},
		},
		{
			desc: "回血冷却, 玩家在范围, 狂暴连击5次",
			setup: func(bb *bt.Blackboard) {
				bb.Set("player_in_range", true)
				bb.Set("heal_ready", false)
			},
		},
	}

	for i, tick := range ticks {
		sep := strings.Repeat("=", 25)
		fmt.Printf("\n%s Tick %d %s\n", sep, i+1, sep)
		fmt.Printf("  [描述] %s\n", tick.desc)

		// 1. 发射事件
		bus.Clear()
		for _, evt := range tick.events {
			fmt.Printf("  [事件] >> %s\n", evt.name)
			bus.Emit(evt.name, evt.data)
		}

		// 2. 更新黑板
		if tick.setup != nil {
			tick.setup(bb)
		}

		// 3. Tick 行为树
		status := tree.Tick()
		fmt.Printf("  [结果] 树返回: %s\n", status)
	}
}
