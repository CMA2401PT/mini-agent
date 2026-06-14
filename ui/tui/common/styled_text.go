package common

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type StyledText struct {
	Text   string
	Style  lipgloss.Style
	Width  int
	Height int
}

func NewStyledText(text string, style lipgloss.Style) *StyledText {
	return &StyledText{Text: text, Style: style}
}

func (w *StyledText) Measure(width int) StreamWidgetHeight {
	return StreamWidgetHeight{Height: len(textLines(w.Style.Width(max(1, width)).Render(w.Text)))}
}

func (w *StyledText) Render() string {
	rendered := w.Style.Width(max(1, w.Width)).Render(w.Text)
	return fitBlock(rendered, w.Width, w.Height)
}

func (w *StyledText) Update(msg tea.Msg) (bool, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		w.Width = size.Width
		w.Height = size.Height
	}
	return false, nil
}
