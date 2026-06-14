package common

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestTextBlockRenderMatchesAssignedRect(t *testing.T) {
	w := NewTextBlock("abc", lipgloss.NewStyle(), nil)
	w.Update(tea.WindowSizeMsg{Width: 5, Height: 3})

	want := "abc  \n     \n     "
	if got := w.Render(); got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

func TestTextBlockWithPaddingAndMarginClickRange(t *testing.T) {
	clicks := 0
	w := NewTextBlockWithPaddingAndMargin("hello", lipgloss.NewStyle(), BoxStyle{
		Padding: Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
		Margin:  Insets{Top: 1, Right: 1, Bottom: 1, Left: 1},
		Border:  BorderSpec{Left: "┃", Style: lipgloss.NewStyle()},
	}, func(x, y int) (bool, tea.Cmd) {
		clicks++
		return true, nil
	})
	w.Update(tea.WindowSizeMsg{Width: 12, Height: w.Measure(12).Height})

	changed, _ := w.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 0, Y: 0})
	if changed || clicks != 0 {
		t.Fatalf("margin click changed=%v clicks=%d, want no click", changed, clicks)
	}

	changed, _ = w.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 1, Y: 1})
	if !changed || clicks != 1 {
		t.Fatalf("box click changed=%v clicks=%d, want click", changed, clicks)
	}
}

type clickRecorder struct {
	width, height int
	x, y          int
	clicked       bool
}

func (r *clickRecorder) Measure(width int) StreamWidgetHeight {
	return StreamWidgetHeight{Height: 1}
}

func (r *clickRecorder) Render() string {
	return fitBlock("x", r.width, r.height)
}

func (r *clickRecorder) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
	case tea.MouseReleaseMsg:
		r.x = msg.X
		r.y = msg.Y
		r.clicked = true
		return true, nil
	}
	return false, nil
}

func TestBlockWithPaddingAndMarginForwardsChildCoordinates(t *testing.T) {
	child := &clickRecorder{}
	w := NewBlockWithPaddingAndMargin(child, BoxStyle{
		Padding: Insets{Top: 1, Left: 2},
		Margin:  Insets{Top: 1, Left: 1},
		Border:  BorderSpec{Left: "┃", Top: "─", Style: lipgloss.NewStyle()},
	}, nil)
	w.Update(tea.WindowSizeMsg{Width: 12, Height: w.Measure(12).Height})

	changed, _ := w.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 4, Y: 3})
	if !changed || !child.clicked {
		t.Fatal("expected child click")
	}
	if child.x != 0 || child.y != 0 {
		t.Fatalf("child coords = (%d,%d), want (0,0)", child.x, child.y)
	}
}
