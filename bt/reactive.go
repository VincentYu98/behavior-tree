package bt

// ReactiveSelector 响应式选择器
//
// 和普通 Selector 的区别：每次 Tick 都从第一个子节点重新评估
// 如果更高优先级的分支变为可用，会抢占（Reset）正在运行的低优先级分支
//
// 典型场景：
//   子节点按优先级排列 [眩晕处理, 紧急闪避, 正常战斗]
//   Boss 正在战斗(子节点2)，突然被眩晕 → 子节点0 命中
//   → Reset 子节点2（中止战斗），切换到子节点0（播放眩晕动画）
type ReactiveSelector struct {
	children   []Node
	runningIdx int // 上一帧哪个子节点在 Running
}

func NewReactiveSelector(children ...Node) *ReactiveSelector {
	return &ReactiveSelector{children: children, runningIdx: -1}
}

func (rs *ReactiveSelector) Tick() Status {
	for i, child := range rs.children {
		status := child.Tick()
		switch status {
		case Success:
			// 如果之前有别的子节点在 Running，重置它
			if rs.runningIdx >= 0 && rs.runningIdx != i {
				rs.children[rs.runningIdx].Reset()
			}
			rs.runningIdx = -1
			return Success
		case Running:
			if rs.runningIdx >= 0 && rs.runningIdx != i {
				rs.children[rs.runningIdx].Reset()
			}
			rs.runningIdx = i
			return Running
		}
		// Failure: 继续尝试下一个
	}
	if rs.runningIdx >= 0 {
		rs.children[rs.runningIdx].Reset()
	}
	rs.runningIdx = -1
	return Failure
}

func (rs *ReactiveSelector) Reset() {
	if rs.runningIdx >= 0 {
		rs.children[rs.runningIdx].Reset()
	}
	rs.runningIdx = -1
	for _, child := range rs.children {
		child.Reset()
	}
}
