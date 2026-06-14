package common

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestTextWidgetMeasuresByAssignedWidth(t *testing.T) {
	w := NewTextWidget("abcdefgh", TextWidgetSizeConfig{
		MinHeight: 1,
		MaxHeight: -1,
	})

	if got := w.Measure(2).Height; got != 4 {
		t.Fatalf("height = %d, want 4", got)
	}
	w.Update(tea.WindowSizeMsg{Width: 2, Height: 4})
	want := "ab\ncd\nef\ngh"
	if got := w.Render(); got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

func TestTextWidgetClipsHeightWithEllipsis(t *testing.T) {
	w := NewTextWidget("abcdefghijklmno", TextWidgetSizeConfig{
		MinHeight: 1,
		MaxHeight: 2,
	})

	w.Update(tea.WindowSizeMsg{Width: 5, Height: 2})

	want := "abcde\nfg..."
	if got := w.Render(); got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

func TestTextWidgetRenderAlwaysMatchesAssignedRect(t *testing.T) {
	w := NewTextWidget("abc", TextWidgetSizeConfig{
		MinHeight: 1,
		MaxHeight: -1,
	})

	w.Update(tea.WindowSizeMsg{Width: 5, Height: 3})

	want := "abc  \n     \n     "
	if got := w.Render(); got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}
