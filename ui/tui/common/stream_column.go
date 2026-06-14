package common

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type StreamColumnElement struct {
	Widget              StreamWidget
	WidgetCurrentHeight int
}

type StreamColumnHeightFunc func(totalHeight int, elementsHeight []StreamWidgetHeight) []int

type StreamColumn struct {
	Elements []StreamColumnElement

	Width, Height int

	ComputeHeight StreamColumnHeightFunc
	focused       int
}

func NewStreamColumn(elements ...StreamWidget) *StreamColumn {
	c := &StreamColumn{focused: -1}
	for _, widget := range elements {
		c.Elements = append(c.Elements, StreamColumnElement{Widget: widget})
	}
	return c
}

func (c *StreamColumn) Measure(width int) StreamWidgetHeight {
	total := 0
	grow := false
	for _, element := range c.Elements {
		if element.Widget == nil {
			continue
		}
		measured := element.Widget.Measure(width)
		total += max(0, measured.Height)
		grow = grow || measured.ExpectGrow
	}
	return StreamWidgetHeight{Height: total, ExpectGrow: grow}
}

func (c *StreamColumn) Render() string {
	rows := make([]string, 0, len(c.Elements))
	for _, element := range c.Elements {
		if element.Widget == nil {
			continue
		}
		rows = append(rows, fitBlock(element.Widget.Render(), c.Width, element.WidgetCurrentHeight))
	}
	return fitBlock(strings.Join(rows, "\n"), c.Width, c.Height)
}

func (c *StreamColumn) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return c.resize(msg.Width, msg.Height)
	case tea.MouseClickMsg:
		return c.handleMouse(msg)
	case tea.MouseMotionMsg:
		return c.dispatchMouseToFocusedOrPosition(msg)
	case tea.MouseReleaseMsg:
		return c.dispatchMouseToFocusedOrPosition(msg)
	case tea.MouseWheelMsg:
		return c.dispatchFocused(msg)
	case tea.KeyPressMsg:
		return c.dispatchFocused(msg)
	default:
		return c.broadcast(msg)
	}
}

func (c *StreamColumn) Focus() tea.Cmd {
	if c.focused < 0 || c.focused >= len(c.Elements) {
		return nil
	}
	return focusWidget(c.Elements[c.focused].Widget)
}

func (c *StreamColumn) Blur() {
	if c.focused < 0 || c.focused >= len(c.Elements) {
		return
	}
	blurWidget(c.Elements[c.focused].Widget)
}

func (c *StreamColumn) FocusChild(idx int) tea.Cmd {
	if idx < 0 || idx >= len(c.Elements) {
		return nil
	}
	return c.switchFocus(idx)
}

func (c *StreamColumn) resize(width, height int) (bool, tea.Cmd) {
	c.Width = max(0, width)
	c.Height = max(0, height)

	var cmds []tea.Cmd
	for pass := 0; pass < 3; pass++ {
		measurements := c.measureElements(c.Width)
		heights := c.computeHeights(measurements)
		for i := range c.Elements {
			c.Elements[i].WidgetCurrentHeight = heights[i]
		}

		changed := false
		for i, element := range c.Elements {
			if element.Widget == nil {
				continue
			}
			childChanged, cmd := element.Widget.Update(tea.WindowSizeMsg{Width: c.Width, Height: heights[i]})
			cmds = append(cmds, cmd)
			changed = changed || childChanged
		}
		if !changed {
			return false, tea.Batch(cmds...)
		}
	}
	return false, tea.Batch(cmds...)
}

func (c *StreamColumn) measureElements(width int) []StreamWidgetHeight {
	out := make([]StreamWidgetHeight, len(c.Elements))
	for i, element := range c.Elements {
		if element.Widget == nil {
			continue
		}
		out[i] = element.Widget.Measure(width)
	}
	return out
}

func (c *StreamColumn) computeHeights(measurements []StreamWidgetHeight) []int {
	if c.ComputeHeight != nil {
		return normalizeHeights(c.ComputeHeight(c.Height, measurements), len(c.Elements), c.Height)
	}
	return defaultColumnHeights(c.Height, measurements)
}

func (c *StreamColumn) handleMouse(msg tea.MouseClickMsg) (bool, tea.Cmd) {
	idx, offsetY := c.elementAtY(msg.Y)
	if idx < 0 {
		return false, nil
	}

	var cmds []tea.Cmd
	if c.focused != idx {
		cmds = append(cmds, c.switchFocus(idx))
	}
	childChanged, childCmd := c.Elements[idx].Widget.Update(OffsetMsg(msg, 0, offsetY))
	cmds = append(cmds, childCmd)
	return c.relayoutAfterChildChange(childChanged, cmds...)
}

func (c *StreamColumn) dispatchMouse(msg tea.Msg) (bool, tea.Cmd) {
	y := mouseY(msg)
	idx, offsetY := c.elementAtY(y)
	if idx < 0 {
		return false, nil
	}
	childChanged, cmd := c.Elements[idx].Widget.Update(OffsetMsg(msg, 0, offsetY))
	return c.relayoutAfterChildChange(childChanged, cmd)
}

func (c *StreamColumn) dispatchMouseToFocusedOrPosition(msg tea.Msg) (bool, tea.Cmd) {
	if c.focused >= 0 && c.focused < len(c.Elements) {
		offsetY := c.elementOffsetY(c.focused)
		childChanged, cmd := c.Elements[c.focused].Widget.Update(OffsetMsg(msg, 0, offsetY))
		return c.relayoutAfterChildChange(childChanged, cmd)
	}
	return c.dispatchMouse(msg)
}

func (c *StreamColumn) dispatchFocused(msg tea.Msg) (bool, tea.Cmd) {
	if c.focused < 0 || c.focused >= len(c.Elements) {
		return false, nil
	}
	childChanged, cmd := c.Elements[c.focused].Widget.Update(msg)
	return c.relayoutAfterChildChange(childChanged, cmd)
}

func (c *StreamColumn) broadcast(msg tea.Msg) (bool, tea.Cmd) {
	var cmds []tea.Cmd
	changed := false
	for _, element := range c.Elements {
		if element.Widget == nil {
			continue
		}
		childChanged, cmd := element.Widget.Update(msg)
		changed = changed || childChanged
		cmds = append(cmds, cmd)
	}
	return c.relayoutAfterChildChange(changed, cmds...)
}

func (c *StreamColumn) relayoutAfterChildChange(childChanged bool, cmds ...tea.Cmd) (bool, tea.Cmd) {
	if !childChanged || c.Width <= 0 || c.Height <= 0 {
		return childChanged, tea.Batch(cmds...)
	}
	_, layoutCmd := c.resize(c.Width, c.Height)
	return true, tea.Batch(append(cmds, layoutCmd)...)
}

func (c *StreamColumn) elementAtY(y int) (idx, offsetY int) {
	if y < 0 {
		return -1, 0
	}
	offset := 0
	for i, element := range c.Elements {
		next := offset + element.WidgetCurrentHeight
		if y < next {
			return i, offset
		}
		offset = next
	}
	return -1, 0
}

func (c *StreamColumn) elementOffsetY(idx int) int {
	if idx < 0 || idx >= len(c.Elements) {
		return 0
	}
	offset := 0
	for i := 0; i < idx; i++ {
		offset += c.Elements[i].WidgetCurrentHeight
	}
	return offset
}

func (c *StreamColumn) switchFocus(idx int) tea.Cmd {
	if c.focused == idx {
		return nil
	}
	if c.focused >= 0 && c.focused < len(c.Elements) {
		blurWidget(c.Elements[c.focused].Widget)
	}
	c.focused = idx
	return focusWidget(c.Elements[idx].Widget)
}

func defaultColumnHeights(totalHeight int, measurements []StreamWidgetHeight) []int {
	heights := make([]int, len(measurements))
	remaining := max(0, totalHeight)
	var growers []int
	for i, measurement := range measurements {
		height := max(0, measurement.Height)
		if height > remaining {
			height = remaining
		}
		heights[i] = height
		remaining -= height
		if measurement.ExpectGrow {
			growers = append(growers, i)
		}
	}
	if remaining <= 0 || len(growers) == 0 {
		return heights
	}
	share := remaining / len(growers)
	extra := remaining % len(growers)
	for _, idx := range growers {
		heights[idx] += share
		if extra > 0 {
			heights[idx]++
			extra--
		}
	}
	return heights
}

func normalizeHeights(heights []int, count, totalHeight int) []int {
	out := make([]int, count)
	remaining := max(0, totalHeight)
	for i := range out {
		if i < len(heights) {
			out[i] = max(0, heights[i])
		}
		if out[i] > remaining {
			out[i] = remaining
		}
		remaining -= out[i]
	}
	return out
}
