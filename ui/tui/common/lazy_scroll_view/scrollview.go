// Package lazy_scroll_view implements a generic, scrollable, vertically-laid-out
// list of blocks as a Bubble Tea component.  It manages:
//   - A per-block, per-line render cache with lazy re-render (see cache.go).
//   - Viewport scrolling with a scrollbar.
//   - Animation-tick dispatch for blocks that report IsAnimating.
package lazy_scroll_view

import (
	"strings"

	"mini_agent/ui/tui/common"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ScrollBlock interface {
	SetWidth(width int)
	Update(tea.Msg) (bool, tea.Cmd)
	IsAnimating() bool
	Render() string
}

type BlockMeta[B ScrollBlock] struct {
	Block B
	Lines []string
	Dirty bool
}

type LazyScrollView[B ScrollBlock] struct {
	Viewport        viewport.Model
	Width           int
	newBlock        func(width int) B
	blocks          []BlockMeta[B]
	lineIndex       []lineIndexEntry
	animatingBlocks map[int]bool
	totalLines      int
	yOffset         int
}

func NewLazyScrollView[B ScrollBlock](newBlock func(width int) B) *LazyScrollView[B] {
	return &LazyScrollView[B]{newBlock: newBlock}
}

func (l *LazyScrollView[B]) EnsureLen(n int) {
	for len(l.blocks) < n {
		l.blocks = append(l.blocks, BlockMeta[B]{Block: l.newBlock(l.Width)})
	}
}

func (l *LazyScrollView[B]) TruncateBlocks(n int) {
	for i := n; i < len(l.blocks); i++ {
		l.setBlockAnimating(i, false)
	}
	l.blocks = l.blocks[:n]
}

func (l *LazyScrollView[B]) BlockCount() int {
	return len(l.blocks)
}

func (l *LazyScrollView[B]) Block(idx int) B {
	if idx < 0 || idx >= len(l.blocks) {
		var zero B
		return zero
	}
	return l.blocks[idx].Block
}

func (l *LazyScrollView[B]) BlockMeta(idx int) *BlockMeta[B] {
	if idx < 0 || idx >= len(l.blocks) {
		return nil
	}
	return &l.blocks[idx]
}

func (l *LazyScrollView[B]) BlockAt(wrappedLineIdx int) int {
	blockIdx, _, ok := l.lineBlockAt(wrappedLineIdx)
	if !ok {
		return -1
	}
	return blockIdx
}

func (l *LazyScrollView[B]) SetWidth(width int) {
	wasAtBottom := l.atBottom()
	if l.Width == width {
		l.keepBottomOrClamp(wasAtBottom)
		return
	}
	l.Width = width
	for i := range l.blocks {
		l.blocks[i].Block.SetWidth(width)
	}
	l.markAllBlockCachesDirty()
	l.rebuildLineIndex()
	l.keepBottomOrClamp(wasAtBottom)
}

func (l *LazyScrollView[B]) InvalidateBlock(idx int) {
	if idx >= 0 && idx < len(l.blocks) {
		l.blocks[idx].Dirty = true
	}
}

func (l *LazyScrollView[B]) IsBlockDirty(idx int) bool {
	if idx < 0 || idx >= len(l.blocks) {
		return false
	}
	return l.blocks[idx].Dirty
}

func (l *LazyScrollView[B]) RefreshBlocks(indexes ...int) {
	wasAtBottom := l.atBottom()
	seen := map[int]bool{}
	for _, idx := range indexes {
		if seen[idx] {
			continue
		}
		seen[idx] = true
		l.renderBlock(idx)
	}
	l.rebuildLineIndex()
	l.keepBottomOrClamp(wasAtBottom)
}

func (l *LazyScrollView[B]) RecomputeContent() {
	wasAtBottom := l.atBottom()
	l.renderAllBlocks()
	l.rebuildLineIndex()
	l.keepBottomOrClamp(wasAtBottom)
}

func (l *LazyScrollView[B]) Render() string {
	h := l.LineHeight()
	if h <= 0 {
		return ""
	}
	cw := l.ContentWidth()
	if cw <= 0 {
		return ""
	}
	l.ensureLineRangeRendered(l.yOffset, l.yOffset+h)

	total := l.totalLines
	yoff := l.yOffset
	thumbStart, thumbSize := scrollbarThumb(h, yoff, total)
	blank := strings.Repeat(" ", cw)

	rows := make([]string, h)
	barCol := make([]string, h)
	for r := 0; r < h; r++ {
		idx := yoff + r
		line := blank
		if idx >= 0 && idx < total {
			line = l.lineAt(idx)
		}
		rows[r] = line
		barCol[r] = scrollbarCell(r, total, h, thumbStart, thumbSize)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(rows, "\n"), strings.Join(barCol, "\n"))
}

func (l *LazyScrollView[B]) LineHeight() int {
	return l.Viewport.Height()
}

func (l *LazyScrollView[B]) ContentWidth() int {
	return l.Viewport.Width()
}

func (l *LazyScrollView[B]) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		wasAtBottom := l.atBottom()
		contentW := msg.Width - 1
		if contentW < 1 {
			contentW = 1
		}
		l.Viewport.SetWidth(contentW)
		l.Viewport.SetHeight(msg.Height)
		l.SetWidth(contentW)
		l.keepBottomOrClamp(wasAtBottom)

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			l.ScrollUp(3)
		case tea.MouseWheelDown:
			l.ScrollDown(3)
		}

	case tea.MouseReleaseMsg:
		return l.SendBlockMsg(msg.Y, msg, msg.X)

	case tea.BackgroundColorMsg:
		l.RecomputeContent()

	case common.AnimationTickMsg:
		var changed []int
		var cmds []tea.Cmd
		for i := range l.animatingBlocks {
			if i < 0 || i >= len(l.blocks) || l.IsBlockPlaceholder(i) {
				l.setBlockAnimating(i, false)
				continue
			}
			blockChanged, blockCmd := l.blocks[i].Block.Update(msg)
			if blockChanged {
				changed = append(changed, i)
			}
			cmds = append(cmds, blockCmd)
			l.updateBlockAnimationState(i)
		}
		if len(changed) > 0 {
			l.RefreshBlocks(changed...)
		}
		return tea.Batch(cmds...)
	}
	return nil
}

func (l *LazyScrollView[B]) SendBlockMsg(viewportY int, msg tea.Msg, x int) tea.Cmd {
	wrappedLineIdx := l.yOffset + viewportY
	l.ensureLineRangeRendered(wrappedLineIdx, wrappedLineIdx+1)
	blockIdx := l.BlockAt(wrappedLineIdx)
	if blockIdx < 0 {
		return nil
	}
	localY := l.localBlockY(wrappedLineIdx)
	b := l.Block(blockIdx)
	changed, cmd := b.Update(localBlockMsg(msg, x, localY))
	if changed {
		l.RefreshBlocks(blockIdx)
	}
	return cmd
}

func localBlockMsg(msg tea.Msg, x, y int) tea.Msg {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		msg.X = x
		msg.Y = y
		return msg
	case tea.MouseMotionMsg:
		msg.X = x
		msg.Y = y
		return msg
	case tea.MouseReleaseMsg:
		msg.X = x
		msg.Y = y
		return msg
	default:
		return msg
	}
}

func (l *LazyScrollView[B]) IsBlockPlaceholder(idx int) bool {
	if idx < 0 || idx >= len(l.blocks) {
		return true
	}
	var zero B
	return any(l.blocks[idx].Block) == any(zero)
}

func (l *LazyScrollView[B]) updateBlockAnimationState(idx int) {
	if idx < 0 || idx >= len(l.blocks) || l.IsBlockPlaceholder(idx) {
		l.setBlockAnimating(idx, false)
		return
	}
	l.setBlockAnimating(idx, l.blocks[idx].Block.IsAnimating())
}

func (l *LazyScrollView[B]) UpdateBlockAnimationState(idx int) {
	l.updateBlockAnimationState(idx)
}

func (l *LazyScrollView[B]) setBlockAnimating(idx int, active bool) {
	if idx < 0 {
		return
	}
	if l.animatingBlocks == nil {
		l.animatingBlocks = map[int]bool{}
	}
	if active {
		l.animatingBlocks[idx] = true
		return
	}
	delete(l.animatingBlocks, idx)
}

func (l *LazyScrollView[B]) maxYOffset() int {
	return max(0, l.totalLines-l.LineHeight())
}

func (l *LazyScrollView[B]) atBottom() bool {
	return l.yOffset >= l.maxYOffset()
}

func (l *LazyScrollView[B]) keepBottomOrClamp(wasAtBottom bool) {
	if wasAtBottom {
		l.gotoBottom()
		return
	}
	if l.yOffset > l.maxYOffset() {
		l.yOffset = l.maxYOffset()
	}
	if l.yOffset < 0 {
		l.yOffset = 0
	}
}

func (l *LazyScrollView[B]) gotoBottom() {
	l.yOffset = l.maxYOffset()
}

func (l *LazyScrollView[B]) ScrollDown(n int) {
	l.yOffset = min(l.yOffset+n, l.maxYOffset())
}

func (l *LazyScrollView[B]) ScrollUp(n int) {
	l.yOffset = max(0, l.yOffset-n)
}

func (l *LazyScrollView[B]) PageDown() {
	l.ScrollDown(l.LineHeight())
}

func (l *LazyScrollView[B]) PageUp() {
	l.ScrollUp(l.LineHeight())
}

func (l *LazyScrollView[B]) AtBottom() bool {
	return l.atBottom()
}

func (l *LazyScrollView[B]) GotoBottom() {
	l.gotoBottom()
}

func scrollbarThumb(height, yoff, total int) (start, size int) {
	if total <= height {
		return 0, 0
	}
	size = height * height / total
	if size < 1 {
		size = 1
	}
	maxYoff := total - height
	start = yoff * (height - size) / maxYoff
	if start > height-size {
		start = height - size
	}
	return start, size
}

func scrollbarCell(row, total, height, thumbStart, thumbSize int) string {
	if total <= height {
		return " "
	}
	if row >= thumbStart && row < thumbStart+thumbSize {
		return lipgloss.NewStyle().Foreground(common.ActiveTheme().ScrollbarThumb).Render("█")
	}
	return lipgloss.NewStyle().Foreground(common.ActiveTheme().ScrollbarTrack).Render("┃")
}
