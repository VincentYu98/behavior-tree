package bt

// Tree 行为树定义，包装根节点。
// 一棵 Tree 是不可变的模板，可被多个 Executor 共享。
type Tree struct {
	name string
	root Node
}

func NewTree(name string, root Node) *Tree {
	return &Tree{name: name, root: root}
}

func (t *Tree) Name() string { return t.name }
func (t *Tree) Root() Node   { return t.root }

// NewExecutor 为一个实体创建执行器。
// 每个实体各持一个 Executor + Context，共享同一棵 Tree。
func (t *Tree) NewExecutor(ctx *Context) *Executor {
	return &Executor{tree: t, ctx: ctx}
}

// Executor 行为树的每实体执行器。
//
// 生命周期：
//   exec := tree.NewExecutor(ctx)
//   for each frame:
//       ctx.Bus.Clear()
//       ctx.Bus.Emit(...)
//       ctx.BB.Set(...)
//       ctx.Delta = dt
//       status := exec.Tick()
//
// 语义约束：
//   - Tick 返回 Running 表示行为未完成，下帧继续
//   - Tick 返回 Success/Failure 后，nodeState 已自动清理，可直接再次 Tick
//   - 调用 Reset 会中断当前 Running 子树（级联 resetFn）
//   - 同一 Tree 可同时被多个 Executor tick（各自 Context 隔离）
type Executor struct {
	tree       *Tree
	ctx        *Context
	lastStatus Status
	ticks      int
}

// Tick 驱动行为树做一次决策。
func (e *Executor) Tick() Status {
	e.ticks++
	if e.ctx != nil && e.ctx.Tracer != nil {
		e.ctx.Tracer.BeginFrame(e.ticks)
	}
	e.lastStatus = e.tree.root.Tick(e.ctx)
	return e.lastStatus
}

// Snapshot 生成当前状态快照（需启用 Tracer）。
func (e *Executor) Snapshot() Snapshot {
	if e.ctx != nil && e.ctx.Tracer != nil {
		return e.ctx.Tracer.TakeSnapshot(e.ctx, e.lastStatus)
	}
	return Snapshot{Status: e.lastStatus}
}

// Reset 中断当前 Running 子树，级联调用所有节点的 resetFn。
// 用于：实体死亡、场景切换、手动重置等外部触发。
func (e *Executor) Reset() {
	e.tree.root.Reset(e.ctx)
	e.lastStatus = Failure
}

func (e *Executor) LastStatus() Status { return e.lastStatus }
func (e *Executor) Ticks() int         { return e.ticks }
func (e *Executor) IsRunning() bool    { return e.lastStatus == Running }
func (e *Executor) Context() *Context  { return e.ctx }
func (e *Executor) Tree() *Tree        { return e.tree }
