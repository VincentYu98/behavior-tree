package bt

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TraceEvent 类型
type TraceEvent int

const (
	TraceEnter     TraceEvent = iota // 节点开始 Tick
	TraceExit                        // 节点 Tick 结束
	TraceReset                       // 节点被 Reset（打断）
	TracePreempt                     // ReactiveSelector 抢占
	TraceInterrupt                   // Interrupt 事件触发打断
)

func (e TraceEvent) String() string {
	switch e {
	case TraceEnter:
		return "Enter"
	case TraceExit:
		return "Exit"
	case TraceReset:
		return "Reset"
	case TracePreempt:
		return "Preempt"
	case TraceInterrupt:
		return "Interrupt"
	default:
		return "Unknown"
	}
}

// TraceEntry 一条 trace 记录
type TraceEntry struct {
	Frame  int        `json:"frame"`
	Type   TraceEvent `json:"type"`
	Node   string     `json:"node"`
	Status Status     `json:"status,omitempty"`
	Depth  int        `json:"depth"`
	Detail string     `json:"detail,omitempty"`
}

// Snapshot 某一帧的状态快照
type Snapshot struct {
	Frame       int            `json:"frame"`
	Status      Status         `json:"status"`
	RunningPath []string       `json:"running_path,omitempty"`
	BB          map[string]any `json:"blackboard,omitempty"`
}

// Tracer 行为树执行追踪器，挂在 Context 上，nil 则不追踪。
type Tracer struct {
	Entries     []TraceEntry
	RunningPath []string // 最近一帧的 Running 路径（从根到最深 Running 节点）

	frame int
	depth int
	stack []string
}

func NewTracer() *Tracer {
	return &Tracer{}
}

// BeginFrame 标记新一帧开始，由 Executor.Tick 调用。
func (t *Tracer) BeginFrame(tick int) {
	t.frame = tick
	t.depth = 0
	t.stack = t.stack[:0]
	t.RunningPath = nil
}

func (t *Tracer) enter(label string) {
	t.Entries = append(t.Entries, TraceEntry{
		Frame: t.frame, Type: TraceEnter, Node: label, Depth: t.depth,
	})
	t.stack = append(t.stack, label)
	t.depth++
}

func (t *Tracer) exit(label string, status Status) {
	t.depth--
	t.Entries = append(t.Entries, TraceEntry{
		Frame: t.frame, Type: TraceExit, Node: label, Status: status, Depth: t.depth,
	})
	// 记录最深 Running 路径（第一个 Running exit 包含完整调用栈）
	if status == Running && t.RunningPath == nil {
		path := make([]string, len(t.stack))
		copy(path, t.stack)
		t.RunningPath = path
	}
	t.stack = t.stack[:len(t.stack)-1]
}

func (t *Tracer) reset(label string) {
	t.Entries = append(t.Entries, TraceEntry{
		Frame: t.frame, Type: TraceReset, Node: label, Depth: t.depth,
	})
}

func (t *Tracer) preempt(from, to string) {
	t.Entries = append(t.Entries, TraceEntry{
		Frame: t.frame, Type: TracePreempt, Depth: t.depth,
		Detail: from + " → " + to,
	})
}

func (t *Tracer) interrupt(node, event string) {
	t.Entries = append(t.Entries, TraceEntry{
		Frame: t.frame, Type: TraceInterrupt, Node: node, Depth: t.depth,
		Detail: "event=" + event,
	})
}

// FrameEntries 返回指定帧的所有 trace 记录。
func (t *Tracer) FrameEntries(frame int) []TraceEntry {
	var result []TraceEntry
	for _, e := range t.Entries {
		if e.Frame == frame {
			result = append(result, e)
		}
	}
	return result
}

// TakeSnapshot 生成当前状态快照。
func (t *Tracer) TakeSnapshot(ctx *Context, status Status) Snapshot {
	snap := Snapshot{
		Frame:       t.frame,
		Status:      status,
		RunningPath: t.RunningPath,
	}
	if ctx != nil && ctx.BB != nil {
		snap.BB = ctx.BB.Dump()
	}
	return snap
}

// DumpFrameText 输出单帧执行路径的可读文本。只展示 Exit 事件（含结果）。
func (t *Tracer) DumpFrameText(frame int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== Frame %d ===\n", frame)
	for _, e := range t.Entries {
		if e.Frame != frame {
			continue
		}
		indent := strings.Repeat("  ", e.Depth)
		switch e.Type {
		case TraceExit:
			fmt.Fprintf(&b, "%s%s → %s\n", indent, e.Node, e.Status)
		case TraceReset:
			fmt.Fprintf(&b, "%s[Reset] %s\n", indent, e.Node)
		case TracePreempt:
			fmt.Fprintf(&b, "%s[Preempt] %s\n", indent, e.Detail)
		case TraceInterrupt:
			fmt.Fprintf(&b, "%s[Interrupt] %s (%s)\n", indent, e.Node, e.Detail)
		}
	}
	if t.RunningPath != nil {
		fmt.Fprintf(&b, "Running: %s\n", strings.Join(t.RunningPath, " > "))
	}
	return b.String()
}

// DumpText 输出所有帧的可读文本。
func (t *Tracer) DumpText() string {
	frames := make(map[int]bool)
	for _, e := range t.Entries {
		frames[e.Frame] = true
	}
	var b strings.Builder
	for f := 1; f <= len(frames); f++ {
		if !frames[f] {
			continue
		}
		b.WriteString(t.DumpFrameText(f))
		b.WriteByte('\n')
	}
	return b.String()
}

// DumpJSON 导出全部 trace 为 JSON。
func (t *Tracer) DumpJSON() ([]byte, error) {
	return json.MarshalIndent(t.Entries, "", "  ")
}

// Clear 清空所有 trace 记录。
func (t *Tracer) Clear() {
	t.Entries = t.Entries[:0]
	t.RunningPath = nil
	t.frame = 0
	t.depth = 0
	t.stack = t.stack[:0]
}
