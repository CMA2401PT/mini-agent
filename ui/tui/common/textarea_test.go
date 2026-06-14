package common

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestTextareaWidgetReportsRemeasureWhenHeightChanges(t *testing.T) {
	w := NewTextareaWidget(TextareaWidgetSizeConfig{
		MinHeight:     1,
		MaxHeight:     5,
		DefaultHeight: 1,
	})
	w.Update(tea.WindowSizeMsg{Width: 8, Height: 3})
	w.Focus()

	changed, _ := w.Update(keyPressMsg('a'))
	if changed {
		t.Fatal("single character should not change textarea height")
	}

	for _, r := range "bcdefghijklmnop" {
		changed, _ = w.Update(keyPressMsg(r))
		if changed {
			return
		}
	}
	t.Fatal("expected wrapped input to request remeasure")
}

func TestStreamColumnRelayoutsWhenTextareaContentGrows(t *testing.T) {
	transcript := &probeWidget{measure: StreamWidgetHeight{Height: 1, ExpectGrow: true}, render: "T"}
	textarea := NewTextareaWidget(TextareaWidgetSizeConfig{
		MinHeight:     1,
		MaxHeight:     5,
		DefaultHeight: 1,
	})
	status := &probeWidget{measure: StreamWidgetHeight{Height: 1}, render: "S"}
	root := NewStreamColumn(transcript, textarea, status)
	root.FocusChild(1)

	root.Update(tea.WindowSizeMsg{Width: 8, Height: 10})
	initial := root.Elements[1].WidgetCurrentHeight

	for _, r := range "abcdefghijklmnop" {
		root.Update(keyPressMsg(r))
	}

	if root.Elements[1].WidgetCurrentHeight <= initial {
		t.Fatalf("textarea height = %d, want > %d", root.Elements[1].WidgetCurrentHeight, initial)
	}
}

func TestTextareaWidgetRendersWithoutBoxChrome(t *testing.T) {
	w := NewTextareaWidget(TextareaWidgetSizeConfig{
		MinHeight:     1,
		MaxHeight:     5,
		DefaultHeight: 1,
	})
	w.Update(tea.WindowSizeMsg{Width: 12, Height: 1})
	w.Focus()
	w.Update(keyPressMsg('h'))
	w.Update(keyPressMsg('i'))

	got := ansi.Strip(w.Render())
	if strings.Contains(got, "┃") {
		t.Fatalf("textarea should not render border chrome: %q", got)
	}
	if strings.HasPrefix(got, " ") {
		t.Fatalf("textarea should not render own left padding: %q", got)
	}
	if !strings.Contains(got, "hi") {
		t.Fatalf("textarea render missing content: %q", got)
	}
}

func TestTextareaWidgetFocusAndBlinkCommands(t *testing.T) {
	w := NewTextareaWidget(TextareaWidgetSizeConfig{
		MinHeight:     1,
		MaxHeight:     5,
		DefaultHeight: 1,
	})
	w.Update(tea.WindowSizeMsg{Width: 12, Height: 1})
	focusCmd := w.Focus()
	if focusCmd == nil {
		t.Fatal("focus should return textarea blink command")
	}
	blinkMsg := focusCmd()
	if blinkMsg == nil {
		t.Fatal("focus command should produce blink message")
	}

	_, nextCmd := w.Update(blinkMsg)
	if nextCmd == nil {
		t.Fatal("textarea should continue scheduling blink commands")
	}
}

func keyPressMsg(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}
