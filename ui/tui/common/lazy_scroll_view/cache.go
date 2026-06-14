package lazy_scroll_view

import (
	"sort"
	"strings"
)

// lineIndexEntry maps a contiguous range of display lines to one block.
type lineIndexEntry struct {
	Block  int
	Start  int
	Height int
}

const estimatedDirtyBlockHeight = 32

func (l *LazyScrollView[B]) renderBlock(idx int) {
	if idx < 0 || idx >= len(l.blocks) {
		return
	}
	blockContent := l.blocks[idx].Block.Render()
	if blockContent == "" {
		l.blocks[idx].Lines = nil
		l.blocks[idx].Dirty = false
		return
	}
	l.blocks[idx].Lines = strings.Split(blockContent, "\n")
	l.blocks[idx].Dirty = false
}

func (l *LazyScrollView[B]) renderAllBlocks() {
	for i := range l.blocks {
		l.renderBlock(i)
	}
}

func (l *LazyScrollView[B]) rebuildLineIndex() {
	l.lineIndex = l.lineIndex[:0]
	l.totalLines = 0
	first := true
	for i, m := range l.blocks {
		height := len(m.Lines)
		if height == 0 && m.Dirty {
			height = estimatedDirtyBlockHeight
		}
		if height == 0 {
			continue
		}
		if !first {
			l.totalLines++
		}
		first = false
		l.lineIndex = append(l.lineIndex, lineIndexEntry{
			Block:  i,
			Start:  l.totalLines,
			Height: height,
		})
		l.totalLines += height
	}
}

func (l *LazyScrollView[B]) lineBlockAt(lineIdx int) (blockIdx, localY int, ok bool) {
	if lineIdx < 0 || lineIdx >= l.totalLines || len(l.lineIndex) == 0 {
		return -1, 0, false
	}
	pos := sort.Search(len(l.lineIndex), func(i int) bool {
		return l.lineIndex[i].Start > lineIdx
	}) - 1
	if pos < 0 {
		return -1, 0, false
	}
	entry := l.lineIndex[pos]
	if lineIdx < entry.Start || lineIdx >= entry.Start+entry.Height {
		return -1, 0, false
	}
	return entry.Block, lineIdx - entry.Start, true
}

func (l *LazyScrollView[B]) localBlockY(wrappedLineIdx int) int {
	_, localY, ok := l.lineBlockAt(wrappedLineIdx)
	if !ok {
		return -1
	}
	return localY
}

func (l *LazyScrollView[B]) lineAt(lineIdx int) string {
	blockIdx, localY, ok := l.lineBlockAt(lineIdx)
	if !ok {
		return ""
	}
	if l.blocks[blockIdx].Dirty {
		l.renderBlock(blockIdx)
		l.rebuildLineIndex()
		l.keepBottomOrClamp(false)
		blockIdx, localY, ok = l.lineBlockAt(lineIdx)
		if !ok {
			return ""
		}
	}
	if blockIdx < 0 || blockIdx >= len(l.blocks) {
		return ""
	}
	lines := l.blocks[blockIdx].Lines
	if localY < 0 || localY >= len(lines) {
		return ""
	}
	return lines[localY]
}

func (l *LazyScrollView[B]) ensureLineRangeRendered(start, end int) {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	for {
		rendered := false
		for _, entry := range l.lineIndex {
			if entry.Start >= end {
				break
			}
			if entry.Start+entry.Height <= start {
				continue
			}
			if !l.blocks[entry.Block].Dirty {
				continue
			}
			l.renderBlock(entry.Block)
			rendered = true
		}
		if !rendered {
			return
		}
		l.rebuildLineIndex()
		l.keepBottomOrClamp(false)
		if end > l.totalLines {
			end = l.totalLines
		}
	}
}

func (l *LazyScrollView[B]) markAllBlockCachesDirty() {
	for i := range l.blocks {
		l.blocks[i].Dirty = true
	}
}
