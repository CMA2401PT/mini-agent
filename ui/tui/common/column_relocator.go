package common

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type ColumnRelocatorWidthFunc func(rootWidth int) (childOffsetX, childOffsetY, childWidth int)

type ColumnRelocator struct {
	Child StreamWidget

	ChildOffsetX, ChildOffsetY int
	ChildWidth, ChildHeight    int

	WidthCompute ColumnRelocatorWidthFunc
}

type ColumnRelocatorHeightFunc func(totalHeight int, columns []ColumnRelocator, elementsHeight []StreamWidgetHeight) []int

type ColumnRelocatorRoot struct {
	Columns []ColumnRelocator

	Width, Height int

	ComputeHeight ColumnRelocatorHeightFunc
	focused       int
}

func NewColumnRelocatorRoot(columns ...ColumnRelocator) *ColumnRelocatorRoot {
	return &ColumnRelocatorRoot{Columns: columns, focused: -1}
}

func (r *ColumnRelocatorRoot) Measure(width int) StreamWidgetHeight {
	measurements, _ := r.measureColumns(width)
	height := 0
	grow := false
	for _, measurement := range measurements {
		height = max(height, measurement.Height)
		grow = grow || measurement.ExpectGrow
	}
	return StreamWidgetHeight{Height: height, ExpectGrow: grow}
}

func (r *ColumnRelocatorRoot) Render() string {
	rows := fitLines(nil, r.Width, r.Height)

	for _, column := range r.Columns {
		if column.Child == nil {
			continue
		}
		block := FitBlock(column.Child.Render(), column.ChildWidth, column.ChildHeight)
		for y, line := range strings.Split(block, "\n") {
			cy := column.ChildOffsetY + y
			if cy < 0 || cy >= len(rows) {
				continue
			}
			if column.ChildOffsetX >= r.Width {
				continue
			}
			left := ansi.Cut(rows[cy], 0, column.ChildOffsetX)
			right := ansi.Cut(rows[cy], column.ChildOffsetX+column.ChildWidth, r.Width)
			rows[cy] = left + FitBlock(line, column.ChildWidth, 1) + right
		}
	}

	return FitBlock(strings.Join(rows, "\n"), r.Width, r.Height)
}

func (r *ColumnRelocatorRoot) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return r.resize(msg.Width, msg.Height)
	case tea.MouseClickMsg:
		return r.handleMouse(msg)
	case tea.MouseMotionMsg:
		return r.dispatchMouse(msg)
	case tea.MouseReleaseMsg:
		return r.dispatchMouse(msg)
	case tea.MouseWheelMsg:
		return r.dispatchFocused(msg)
	case tea.KeyPressMsg:
		return r.dispatchFocused(msg)
	default:
		return r.broadcast(msg)
	}
}

func (r *ColumnRelocatorRoot) Focus() tea.Cmd {
	if r.focused < 0 || r.focused >= len(r.Columns) {
		return nil
	}
	return Focus(r.Columns[r.focused].Child)
}

func (r *ColumnRelocatorRoot) Blur() {
	if r.focused < 0 || r.focused >= len(r.Columns) {
		return
	}
	Blur(r.Columns[r.focused].Child)
}

func (r *ColumnRelocatorRoot) resize(width, height int) (bool, tea.Cmd) {
	r.Width = max(0, width)
	r.Height = max(0, height)

	var cmds []tea.Cmd
	for pass := 0; pass < 3; pass++ {
		measurements, offsets := r.measureColumns(r.Width)
		heights := r.computeHeights(measurements)
		for i := range r.Columns {
			r.Columns[i].ChildOffsetX = offsets[i].x
			r.Columns[i].ChildOffsetY = offsets[i].y
			r.Columns[i].ChildWidth = offsets[i].width
			r.Columns[i].ChildHeight = heights[i]
		}

		changed := false
		for i, column := range r.Columns {
			if column.Child == nil {
				continue
			}
			childChanged, cmd := column.Child.Update(tea.WindowSizeMsg{Width: column.ChildWidth, Height: heights[i]})
			cmds = append(cmds, cmd)
			changed = changed || childChanged
		}
		if !changed {
			return false, tea.Batch(cmds...)
		}
	}
	return false, tea.Batch(cmds...)
}

type relocatorOffset struct {
	x, y, width int
}

func (r *ColumnRelocatorRoot) measureColumns(width int) ([]StreamWidgetHeight, []relocatorOffset) {
	measurements := make([]StreamWidgetHeight, len(r.Columns))
	offsets := make([]relocatorOffset, len(r.Columns))
	for i, column := range r.Columns {
		childX, childY, childWidth := 0, 0, width
		if column.WidthCompute != nil {
			childX, childY, childWidth = column.WidthCompute(width)
		}
		if childX < 0 {
			childX = 0
		}
		if childY < 0 {
			childY = 0
		}
		if childWidth < 0 {
			childWidth = 0
		}
		if childX+childWidth > width {
			childWidth = max(0, width-childX)
		}
		offsets[i] = relocatorOffset{x: childX, y: childY, width: childWidth}
		if column.Child != nil {
			measurements[i] = column.Child.Measure(childWidth)
		}
	}
	return measurements, offsets
}

func (r *ColumnRelocatorRoot) computeHeights(measurements []StreamWidgetHeight) []int {
	if r.ComputeHeight != nil {
		return normalizeHeights(r.ComputeHeight(r.Height, r.Columns, measurements), len(r.Columns), r.Height)
	}
	heights := make([]int, len(measurements))
	for i, measurement := range measurements {
		heights[i] = min(max(0, measurement.Height), max(0, r.Height))
	}
	return heights
}

func (r *ColumnRelocatorRoot) handleMouse(msg tea.MouseClickMsg) (bool, tea.Cmd) {
	idx := r.columnAt(msg.X, msg.Y)
	if idx < 0 {
		return false, nil
	}
	var cmds []tea.Cmd
	if r.focused != idx {
		cmds = append(cmds, r.switchFocus(idx))
	}
	childChanged, childCmd := r.Columns[idx].Child.Update(OffsetMouseMsg(msg, r.Columns[idx].ChildOffsetX, r.Columns[idx].ChildOffsetY))
	cmds = append(cmds, childCmd)
	return r.relayoutAfterChildChange(childChanged, cmds...)
}

func (r *ColumnRelocatorRoot) dispatchMouse(msg tea.Msg) (bool, tea.Cmd) {
	idx := r.columnAt(mouseX(msg), mouseY(msg))
	if idx < 0 {
		return false, nil
	}
	childChanged, cmd := r.Columns[idx].Child.Update(OffsetMouseMsg(msg, r.Columns[idx].ChildOffsetX, r.Columns[idx].ChildOffsetY))
	return r.relayoutAfterChildChange(childChanged, cmd)
}

func (r *ColumnRelocatorRoot) dispatchFocused(msg tea.Msg) (bool, tea.Cmd) {
	if r.focused < 0 || r.focused >= len(r.Columns) {
		return false, nil
	}
	childChanged, cmd := r.Columns[r.focused].Child.Update(msg)
	return r.relayoutAfterChildChange(childChanged, cmd)
}

func (r *ColumnRelocatorRoot) broadcast(msg tea.Msg) (bool, tea.Cmd) {
	var cmds []tea.Cmd
	changed := false
	for _, column := range r.Columns {
		if column.Child == nil {
			continue
		}
		childChanged, cmd := column.Child.Update(msg)
		changed = changed || childChanged
		cmds = append(cmds, cmd)
	}
	return r.relayoutAfterChildChange(changed, cmds...)
}

func (r *ColumnRelocatorRoot) relayoutAfterChildChange(childChanged bool, cmds ...tea.Cmd) (bool, tea.Cmd) {
	if !childChanged || r.Width <= 0 || r.Height <= 0 {
		return childChanged, tea.Batch(cmds...)
	}
	_, layoutCmd := r.resize(r.Width, r.Height)
	return true, tea.Batch(append(cmds, layoutCmd)...)
}

func (r *ColumnRelocatorRoot) columnAt(x, y int) int {
	for i := len(r.Columns) - 1; i >= 0; i-- {
		column := r.Columns[i]
		if column.Child == nil {
			continue
		}
		if x >= column.ChildOffsetX && x < column.ChildOffsetX+column.ChildWidth &&
			y >= column.ChildOffsetY && y < column.ChildOffsetY+column.ChildHeight {
			return i
		}
	}
	return -1
}

func (r *ColumnRelocatorRoot) switchFocus(idx int) tea.Cmd {
	if r.focused == idx {
		return nil
	}
	if r.focused >= 0 && r.focused < len(r.Columns) {
		Blur(r.Columns[r.focused].Child)
	}
	r.focused = idx
	return Focus(r.Columns[idx].Child)
}

func InsetWidth(offset int) ColumnRelocatorWidthFunc {
	return func(rootWidth int) (int, int, int) {
		if rootWidth <= offset {
			return 0, 0, max(0, rootWidth)
		}
		return offset, 0, rootWidth - offset
	}
}
