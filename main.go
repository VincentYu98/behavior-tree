package main

import (
	"behavior-tree/bt"
	"fmt"
	"log"
	"strings"
)

func registerActions(loader *bt.Loader) {

	// ========== 阶段切换 ==========
	loader.RegisterAction("enter_phase1", func(ctx *bt.Context) bt.Status {
		phase, _ := bt.Get[int](ctx.BB, "_phase")
		if phase != 1 {
			ctx.BB.Set("_phase", 1)
			fmt.Println("  ⚔ [阶段] 炎魔将军 - Phase1 正常战斗")
		}
		return bt.Success
	})
	loader.RegisterAction("enter_phase2", func(ctx *bt.Context) bt.Status {
		phase, _ := bt.Get[int](ctx.BB, "_phase")
		if phase != 2 {
			ctx.BB.Set("_phase", 2)
			fmt.Println("  ⚔ [阶段] 炎魔将军 - Phase2 狂暴!")
			fmt.Println("    \"你们这些蝼蚁...让我认真起来!\"")
		}
		return bt.Success
	})
	loader.RegisterAction("enter_phase3", func(ctx *bt.Context) bt.Status {
		phase, _ := bt.Get[int](ctx.BB, "_phase")
		if phase != 3 {
			ctx.BB.Set("_phase", 3)
			fmt.Println("  ⚔ [阶段] 炎魔将军 - Phase3 残血狂暴!")
			fmt.Println("    \"同归于尽吧!\"")
		}
		return bt.Success
	})

	// ========== 事件驱动：眩晕 / 闪避 ==========
	loader.RegisterActionWithReset("play_stun", func(ctx *bt.Context) bt.Status {
		ticks, _ := bt.Get[int](ctx.BB, "_stun_ticks")
		ticks++
		ctx.BB.Set("_stun_ticks", ticks)
		if ticks < 3 {
			fmt.Printf("  ★ 炎魔将军被眩晕中... (%d/3)\n", ticks)
			return bt.Running
		}
		fmt.Println("  ★ 炎魔将军眩晕结束, 恢复行动!")
		ctx.BB.Set("_stun_ticks", 0)
		ctx.BB.Set("stunned", false)
		return bt.Success
	}, func(ctx *bt.Context) {
		ctx.BB.Set("_stun_ticks", 0)
	})

	loader.RegisterAction("dodge_roll", func(ctx *bt.Context) bt.Status {
		fmt.Println("  ★ 炎魔将军紧急翻滚闪避!")
		ctx.BB.Set("should_dodge", false)
		return bt.Success
	})
	loader.RegisterAction("counter_attack", func(_ *bt.Context) bt.Status {
		fmt.Println("  ★ 炎魔将军反击一刀!")
		return bt.Success
	})

	// ========== Phase1 ==========
	loader.RegisterAction("patrol", func(_ *bt.Context) bt.Status {
		fmt.Println("  → 炎魔将军在领地内巡逻...")
		return bt.Success
	})
	loader.RegisterAction("chase", func(_ *bt.Context) bt.Status {
		fmt.Println("  → 炎魔将军向玩家冲去!")
		return bt.Success
	})

	loader.RegisterActionWithReset("channel_fireball", func(ctx *bt.Context) bt.Status {
		ticks, _ := bt.Get[int](ctx.BB, "_fb_ticks")
		ticks++
		ctx.BB.Set("_fb_ticks", ticks)
		if ticks < 3 {
			fmt.Printf("  → 炎魔将军蓄力【火球术】... (%d/3)\n", ticks)
			return bt.Running
		}
		ctx.BB.Set("_fb_ticks", 0)
		ctx.BB.Set("fireball_ready", false)
		fmt.Println("  → 炎魔将军释放【火球术】! 轰!")
		return bt.Success
	}, func(ctx *bt.Context) {
		ticks, _ := bt.Get[int](ctx.BB, "_fb_ticks")
		if ticks > 0 {
			fmt.Println("    !! 火球蓄力被打断!")
			ctx.BB.Set("_fb_ticks", 0)
		}
	})

	loader.RegisterAction("attack", func(ctx *bt.Context) bt.Status {
		combo, _ := bt.Get[int](ctx.BB, "_combo")
		combo++
		ctx.BB.Set("_combo", combo)
		fmt.Printf("  → 普通攻击 (第%d击)\n", combo)
		return bt.Success
	})
	loader.RegisterAction("backstep", func(ctx *bt.Context) bt.Status {
		fmt.Println("  → 后跳拉开距离!")
		ctx.BB.Set("_combo", 0)
		return bt.Success
	})

	// ========== Phase2 ==========
	loader.RegisterAction("summon_minions", func(ctx *bt.Context) bt.Status {
		fmt.Println("  → 炎魔将军召唤3只火焰小鬼!")
		ctx.BB.Set("minions_summoned", true)
		return bt.Success
	})
	loader.RegisterAction("coordinate_attack", func(ctx *bt.Context) bt.Status {
		count, _ := bt.Get[int](ctx.BB, "minion_count")
		fmt.Printf("  → 炎魔将军与%d只小鬼发起协同攻击!\n", count)
		return bt.Success
	})
	loader.RegisterAction("fire_storm", func(ctx *bt.Context) bt.Status {
		fmt.Println("  → 炎魔将军施放【火焰风暴】! 全屏AOE!")
		ctx.BB.Set("fire_storm_ready", false)
		return bt.Success
	})
	loader.RegisterAction("charge", func(ctx *bt.Context) bt.Status {
		fmt.Println("  → 炎魔将军冲锋突进!")
		ctx.BB.Set("player_in_range", true)
		return bt.Success
	})
	loader.RegisterAction("enhanced_attack", func(ctx *bt.Context) bt.Status {
		combo, _ := bt.Get[int](ctx.BB, "_combo")
		combo++
		ctx.BB.Set("_combo", combo)
		fmt.Printf("  → 强化攻击 (第%d击) 附带灼烧!\n", combo)
		return bt.Success
	})
	loader.RegisterAction("war_cry", func(ctx *bt.Context) bt.Status {
		fmt.Println("  → 炎魔将军发出战吼! 降低玩家防御!")
		ctx.BB.Set("_combo", 0)
		return bt.Success
	})

	// ========== Phase3 ==========
	loader.RegisterActionWithReset("cast_ultimate", func(ctx *bt.Context) bt.Status {
		ticks, _ := bt.Get[int](ctx.BB, "_ult_ticks")
		ticks++
		ctx.BB.Set("_ult_ticks", ticks)
		if ticks < 4 {
			fmt.Printf("  → 炎魔将军蓄力【炎灭天地】... (%d/4) [Running]\n", ticks)
			return bt.Running
		}
		ctx.BB.Set("_ult_ticks", 0)
		ctx.BB.Set("ultimate_ready", false)
		fmt.Println("  → 炎魔将军释放【炎灭天地】!!! 全屏毁灭!")
		return bt.Success
	}, func(ctx *bt.Context) {
		ticks, _ := bt.Get[int](ctx.BB, "_ult_ticks")
		if ticks > 0 {
			fmt.Println("    !! 终极技能蓄力被打断! 技能进入冷却!")
			ctx.BB.Set("_ult_ticks", 0)
			ctx.BB.Set("ultimate_ready", false)
		}
	})

	loader.RegisterAction("flee", func(_ *bt.Context) bt.Status {
		fmt.Println("  → 炎魔将军向后撤退!")
		return bt.Success
	})
	loader.RegisterAction("heal", func(ctx *bt.Context) bt.Status {
		fmt.Println("  → 炎魔将军吞噬火焰恢复生命!")
		ctx.BB.Set("heal_ready", false)
		return bt.Success
	})
	loader.RegisterAction("berserk_attack", func(ctx *bt.Context) bt.Status {
		combo, _ := bt.Get[int](ctx.BB, "_combo")
		combo++
		ctx.BB.Set("_combo", combo)
		fmt.Printf("  → 狂暴攻击 (第%d击) 伤害翻倍!\n", combo)
		return bt.Success
	})
}

type FrameEvent struct {
	name string
	data any
}

type Tick struct {
	desc   string
	events []FrameEvent
	setup  func(bb *bt.Blackboard)
}

func main() {
	bb := bt.NewBlackboard()
	bus := bt.NewEventBus()
	ctx := &bt.Context{BB: bb, Bus: bus, Delta: 0.016}

	loader := bt.NewLoader()
	registerActions(loader)

	tree, err := loader.LoadFile("boss_event.json")
	if err != nil {
		log.Fatalf("加载行为树失败: %v", err)
	}

	ticks := []Tick{
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
				bb.Set("_combo", 0)
				bb.Set("_phase", 0)
				bb.Set("_fb_ticks", 0)
				bb.Set("_ult_ticks", 0)
				bb.Set("_stun_ticks", 0)
			},
		},
		{desc: "远处发现玩家, 火球就绪, 开始蓄力", setup: func(bb *bt.Blackboard) { bb.Set("player_detected", true); bb.Set("fireball_ready", true) }},
		{desc: "(续) 火球蓄力中..."},
		{desc: "玩家沉默了 Boss! 火球蓄力被打断!", events: []FrameEvent{{name: "on_silenced"}}},
		{desc: "火球冷却, 追击", setup: func(bb *bt.Blackboard) { bb.Set("fireball_ready", false) }},
		{desc: "玩家进入范围, 连击", setup: func(bb *bt.Blackboard) { bb.Set("hp_percent", 90); bb.Set("player_in_range", true) }},
		{desc: "HP<60%, Phase2, 召唤小怪", setup: func(bb *bt.Blackboard) { bb.Set("hp_percent", 55); bb.Set("minions_summoned", false) }},
		{desc: "事件: 小怪就位!", events: []FrameEvent{{name: "minions_ready", data: 3}}},
		{desc: "火焰风暴", setup: func(bb *bt.Blackboard) { bb.Set("hp_percent", 50); bb.Set("fire_storm_ready", true) }},
		{desc: "强化连击", setup: func(bb *bt.Blackboard) { bb.Set("hp_percent", 40) }},
		{desc: "HP<20%, Phase3, 终极蓄力!", setup: func(bb *bt.Blackboard) { bb.Set("hp_percent", 15); bb.Set("ultimate_ready", true) }},
		{desc: "(续) 终极蓄力中..."},
		{desc: "Boss被眩晕! 终极被打断!", events: []FrameEvent{{name: "on_stunned"}}, setup: func(bb *bt.Blackboard) { bb.Set("stunned", true) }},
		{desc: "(续) 眩晕中..."},
		{desc: "眩晕结束, 逃跑回血", setup: func(bb *bt.Blackboard) { bb.Set("heal_ready", true) }},
		{desc: "玩家重击! 闪避+反击!", events: []FrameEvent{{name: "on_heavy_hit"}}, setup: func(bb *bt.Blackboard) { bb.Set("should_dodge", true) }},
		{desc: "狂暴连击5次", setup: func(bb *bt.Blackboard) { bb.Set("player_in_range", true); bb.Set("heal_ready", false) }},
	}

	for i, tick := range ticks {
		sep := strings.Repeat("=", 25)
		fmt.Printf("\n%s Tick %d %s\n", sep, i+1, sep)
		fmt.Printf("  [描述] %s\n", tick.desc)

		bus.Clear()
		for _, evt := range tick.events {
			fmt.Printf("  [事件] >> %s\n", evt.name)
			bus.Emit(evt.name, evt.data)
		}
		if tick.setup != nil {
			tick.setup(bb)
		}
		status := tree.Tick(ctx)
		fmt.Printf("  [结果] 树返回: %s\n", status)
	}
}
