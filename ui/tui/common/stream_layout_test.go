package common

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

type probeWidget struct {
	measure    StreamWidgetHeight
	render     string
	lastSize   tea.WindowSizeMsg
	lastMouseX int
	lastMouseY int
	keyCount   int
	focused    bool
}

func (w *probeWidget) Measure(width int) StreamWidgetHeight {
	return w.measure
}

func (w *probeWidget) Render() string {
	return w.render
}

func (w *probeWidget) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		w.lastSize = m
	case tea.MouseClickMsg:
		w.lastMouseX = m.X
		w.lastMouseY = m.Y
	case tea.MouseReleaseMsg:
		w.lastMouseX = m.X
		w.lastMouseY = m.Y
	case tea.KeyPressMsg:
		w.keyCount++
	}
	return false, nil
}

func (w *probeWidget) Focus() tea.Cmd {
	w.focused = true
	return nil
}

func (w *probeWidget) Blur() {
	w.focused = false
}

func TestStreamColumnAllocatesFixedAndGrowHeights(t *testing.T) {
	top := &probeWidget{measure: StreamWidgetHeight{Height: 1, ExpectGrow: true}, render: "T"}
	bottom := &probeWidget{measure: StreamWidgetHeight{Height: 2}, render: "B"}
	column := NewStreamColumn(top, bottom)

	column.Update(tea.WindowSizeMsg{Width: 4, Height: 6})

	if top.lastSize.Height != 4 || bottom.lastSize.Height != 2 {
		t.Fatalf("heights = %d/%d, want 4/2", top.lastSize.Height, bottom.lastSize.Height)
	}
	if top.lastSize.Width != 4 || bottom.lastSize.Width != 4 {
		t.Fatalf("widths = %d/%d, want 4/4", top.lastSize.Width, bottom.lastSize.Width)
	}
}

func TestStreamColumnFocusAndLocalMouseCoordinates(t *testing.T) {
	top := &probeWidget{measure: StreamWidgetHeight{Height: 2}, render: "T"}
	bottom := &probeWidget{measure: StreamWidgetHeight{Height: 2}, render: "B"}
	column := NewStreamColumn(top, bottom)
	column.Update(tea.WindowSizeMsg{Width: 8, Height: 4})

	column.Update(tea.MouseClickMsg{X: 5, Y: 3, Button: tea.MouseLeft})

	if !bottom.focused || top.focused {
		t.Fatal("expected bottom widget focused")
	}
	if bottom.lastMouseX != 5 || bottom.lastMouseY != 1 {
		t.Fatalf("bottom mouse = %d,%d, want 5,1", bottom.lastMouseX, bottom.lastMouseY)
	}
}

func TestStreamColumnForwardsKeysOnlyToFocusedChild(t *testing.T) {
	top := &probeWidget{measure: StreamWidgetHeight{Height: 1}, render: "T"}
	bottom := &probeWidget{measure: StreamWidgetHeight{Height: 1}, render: "B"}
	column := NewStreamColumn(top, bottom)
	column.Update(tea.WindowSizeMsg{Width: 8, Height: 2})
	column.Update(tea.MouseClickMsg{X: 0, Y: 1, Button: tea.MouseLeft})

	column.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	if top.keyCount != 0 || bottom.keyCount != 1 {
		t.Fatalf("key counts = %d/%d, want 0/1", top.keyCount, bottom.keyCount)
	}
}

func TestStreamColumnMouseReleaseReturnsToFocusedChild(t *testing.T) {
	top := &probeWidget{measure: StreamWidgetHeight{Height: 2}, render: "T"}
	bottom := &probeWidget{measure: StreamWidgetHeight{Height: 2}, render: "B"}
	column := NewStreamColumn(top, bottom)
	column.Update(tea.WindowSizeMsg{Width: 8, Height: 4})

	column.Update(tea.MouseClickMsg{X: 5, Y: 1, Button: tea.MouseLeft})
	column.Update(tea.MouseReleaseMsg{X: 5, Y: 3, Button: tea.MouseLeft})

	if top.lastMouseX != 5 || top.lastMouseY != 3 {
		t.Fatalf("release should return to focused top widget with local-ish coordinates, got %d,%d", top.lastMouseX, top.lastMouseY)
	}
	if bottom.lastMouseX != 0 || bottom.lastMouseY != 0 {
		t.Fatalf("release should not go to bottom widget, got %d,%d", bottom.lastMouseX, bottom.lastMouseY)
	}
}

func TestColumnRelocatorOffsetsAndRendersChild(t *testing.T) {
	child := &probeWidget{measure: StreamWidgetHeight{Height: 2}, render: "A\nB"}
	root := NewColumnRelocatorRoot(ColumnRelocator{
		Child:        child,
		WidthCompute: InsetWidth(2),
	})

	root.Update(tea.WindowSizeMsg{Width: 5, Height: 2})

	if child.lastSize.Width != 3 || child.lastSize.Height != 2 {
		t.Fatalf("child size = %dx%d, want 3x2", child.lastSize.Width, child.lastSize.Height)
	}
	if got, want := root.Render(), "  A  \n  B  "; got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}

	root.Update(tea.MouseReleaseMsg{X: 3, Y: 1, Button: tea.MouseLeft})
	if child.lastMouseX != 1 || child.lastMouseY != 1 {
		t.Fatalf("child mouse = %d,%d, want 1,1", child.lastMouseX, child.lastMouseY)
	}
}
