package agent_interact_simple

import (
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
)

func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}

type Transcript struct {
	Viewport viewport.Model
	dirty    bool
}

func (t *Transcript) SetContent(content string, width int) {
	t.Viewport.SetContent(wrapText(content, width))
	t.dirty = false
}
