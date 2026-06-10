package bt

import (
	"encoding/json"
	"fmt"
	"os"
)

// NodeConfig JSON 中每个节点的配置结构
type NodeConfig struct {
	Type     string        `json:"type"`
	Name     string        `json:"name,omitempty"`
	Action   string        `json:"action,omitempty"`     // action 节点
	Key      string        `json:"key,omitempty"`        // condition 节点
	Op       string        `json:"op,omitempty"`         // condition 节点
	Value    any           `json:"value,omitempty"`      // condition 节点
	Count    int           `json:"count,omitempty"`      // repeater 节点
	Event    string        `json:"event,omitempty"`      // interrupt / wait_event 节点
	WriteTo  string        `json:"write_to,omitempty"`   // wait_event 节点
	Child    *NodeConfig   `json:"child,omitempty"`      // 装饰节点
	Children []*NodeConfig `json:"children,omitempty"`   // 组合节点
}

// ActionDef 注册的行为定义，包含执行函数和可选的重置函数
type ActionDef struct {
	Fn    func() Status
	Reset func()
}

// Loader 从 JSON 构建行为树
type Loader struct {
	bb      *Blackboard
	bus     *EventBus
	actions map[string]ActionDef
}

func NewLoader(bb *Blackboard, bus *EventBus) *Loader {
	return &Loader{bb: bb, bus: bus, actions: make(map[string]ActionDef)}
}

func (l *Loader) RegisterAction(name string, fn func() Status) {
	l.actions[name] = ActionDef{Fn: fn}
}

// RegisterActionWithReset 注册带重置函数的行为
// 重置函数在 Interrupt/ReactiveSelector 打断时被级联调用
func (l *Loader) RegisterActionWithReset(name string, fn func() Status, resetFn func()) {
	l.actions[name] = ActionDef{Fn: fn, Reset: resetFn}
}

func (l *Loader) LoadFile(path string) (Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return l.LoadJSON(data)
}

func (l *Loader) LoadJSON(data []byte) (Node, error) {
	var cfg NodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return l.build(&cfg)
}

func (l *Loader) build(cfg *NodeConfig) (Node, error) {
	switch cfg.Type {
	case "action":
		def, ok := l.actions[cfg.Action]
		if !ok {
			return nil, fmt.Errorf("unknown action: %q", cfg.Action)
		}
		name := cfg.Name
		if name == "" {
			name = cfg.Action
		}
		return NewActionWithReset(name, def.Fn, def.Reset), nil

	case "condition":
		return NewCondition(l.bb, cfg.Key, cfg.Op, cfg.Value), nil

	case "sequence":
		children, err := l.buildChildren(cfg.Children)
		if err != nil {
			return nil, fmt.Errorf("sequence %q: %w", cfg.Name, err)
		}
		return NewSequence(children...), nil

	case "selector":
		children, err := l.buildChildren(cfg.Children)
		if err != nil {
			return nil, fmt.Errorf("selector %q: %w", cfg.Name, err)
		}
		return NewSelector(children...), nil

	case "reactive_selector":
		children, err := l.buildChildren(cfg.Children)
		if err != nil {
			return nil, fmt.Errorf("reactive_selector %q: %w", cfg.Name, err)
		}
		return NewReactiveSelector(children...), nil

	case "inverter":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, fmt.Errorf("inverter: %w", err)
		}
		return NewInverter(child), nil

	case "repeater":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, fmt.Errorf("repeater: %w", err)
		}
		return NewRepeater(cfg.Count, child), nil

	case "always_succeed":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, fmt.Errorf("always_succeed: %w", err)
		}
		return NewAlwaysSucceed(child), nil

	case "until_fail":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, fmt.Errorf("until_fail: %w", err)
		}
		return NewUntilFail(child), nil

	case "interrupt":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, fmt.Errorf("interrupt: %w", err)
		}
		return NewInterrupt(cfg.Event, l.bus, child), nil

	case "wait_event":
		return NewWaitForEvent(cfg.Event, cfg.WriteTo, l.bus, l.bb), nil

	default:
		return nil, fmt.Errorf("unknown node type: %q", cfg.Type)
	}
}

func (l *Loader) buildChildren(configs []*NodeConfig) ([]Node, error) {
	nodes := make([]Node, 0, len(configs))
	for _, cfg := range configs {
		node, err := l.build(cfg)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (l *Loader) buildChild(cfg *NodeConfig) (Node, error) {
	if cfg.Child == nil {
		return nil, fmt.Errorf("missing child")
	}
	return l.build(cfg.Child)
}
