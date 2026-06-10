package bt

import (
	"encoding/json"
	"fmt"
	"os"
)

type NodeConfig struct {
	Type     string        `json:"type"`
	Name     string        `json:"name,omitempty"`
	Action   string        `json:"action,omitempty"`
	Key      string        `json:"key,omitempty"`
	Op       string        `json:"op,omitempty"`
	Value    any           `json:"value,omitempty"`
	Count    int           `json:"count,omitempty"`
	Event    string        `json:"event,omitempty"`
	WriteTo  string        `json:"write_to,omitempty"`
	Child    *NodeConfig   `json:"child,omitempty"`
	Children []*NodeConfig `json:"children,omitempty"`
}

type ActionDef struct {
	Fn    func(ctx *Context) Status
	Reset func()
}

// Loader 从 JSON 构建行为树
// Context 在运行时传入，Loader 只负责构建阶段
type Loader struct {
	actions map[string]ActionDef
}

func NewLoader() *Loader {
	return &Loader{actions: make(map[string]ActionDef)}
}

func (l *Loader) RegisterAction(name string, fn func(ctx *Context) Status) {
	l.actions[name] = ActionDef{Fn: fn}
}

func (l *Loader) RegisterActionWithReset(name string, fn func(ctx *Context) Status, resetFn func()) {
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

var validOps = map[string]bool{
	"eq": true, "ne": true, "lt": true, "gt": true, "le": true, "ge": true,
}

func (l *Loader) validate(cfg *NodeConfig) error {
	switch cfg.Type {
	case "action":
		if cfg.Action == "" {
			return fmt.Errorf("action: missing 'action' field")
		}
		if _, ok := l.actions[cfg.Action]; !ok {
			return fmt.Errorf("action: unknown action %q (not registered)", cfg.Action)
		}
	case "condition":
		if cfg.Key == "" {
			return fmt.Errorf("condition: missing 'key' field")
		}
		if !validOps[cfg.Op] {
			return fmt.Errorf("condition: invalid op %q, valid: eq ne lt gt le ge", cfg.Op)
		}
	case "sequence", "selector", "reactive_selector", "reactive_sequence":
		if len(cfg.Children) == 0 {
			return fmt.Errorf("%s: 'children' is empty", cfg.Type)
		}
	case "inverter", "always_succeed", "until_fail":
		if cfg.Child == nil {
			return fmt.Errorf("%s: missing 'child'", cfg.Type)
		}
	case "repeater":
		if cfg.Count <= 0 {
			return fmt.Errorf("repeater: count must be > 0, got %d", cfg.Count)
		}
		if cfg.Child == nil {
			return fmt.Errorf("repeater: missing 'child'")
		}
	case "interrupt":
		if cfg.Event == "" {
			return fmt.Errorf("interrupt: missing 'event' field")
		}
		if cfg.Child == nil {
			return fmt.Errorf("interrupt: missing 'child'")
		}
	case "wait_event":
		if cfg.Event == "" {
			return fmt.Errorf("wait_event: missing 'event' field")
		}
	default:
		return fmt.Errorf("unknown node type: %q", cfg.Type)
	}
	return nil
}

func (l *Loader) build(cfg *NodeConfig) (Node, error) {
	if err := l.validate(cfg); err != nil {
		name := cfg.Name
		if name != "" {
			return nil, fmt.Errorf("[%s] %w", name, err)
		}
		return nil, err
	}

	switch cfg.Type {
	case "action":
		def := l.actions[cfg.Action]
		name := cfg.Name
		if name == "" {
			name = cfg.Action
		}
		return NewActionWithReset(name, def.Fn, def.Reset), nil

	case "condition":
		return NewCondition(cfg.Key, cfg.Op, cfg.Value), nil

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

	case "reactive_sequence":
		children, err := l.buildChildren(cfg.Children)
		if err != nil {
			return nil, fmt.Errorf("reactive_sequence %q: %w", cfg.Name, err)
		}
		return NewReactiveSequence(children...), nil

	case "inverter":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, err
		}
		return NewInverter(child), nil

	case "repeater":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, err
		}
		return NewRepeater(cfg.Count, child), nil

	case "always_succeed":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, err
		}
		return NewAlwaysSucceed(child), nil

	case "until_fail":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, err
		}
		return NewUntilFail(child), nil

	case "interrupt":
		child, err := l.buildChild(cfg)
		if err != nil {
			return nil, err
		}
		return NewInterrupt(cfg.Event, child), nil

	case "wait_event":
		return NewWaitForEvent(cfg.Event, cfg.WriteTo), nil

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
