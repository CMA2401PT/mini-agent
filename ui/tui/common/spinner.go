package common

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type SpinnerWidget struct {
	Frames []string
	Frame  int
	Style  lipgloss.Style

	width  int
	height int
}

func NewSpinnerWidget(frames []string, style lipgloss.Style) *SpinnerWidget {
	return &SpinnerWidget{Frames: frames, Style: style}
}

func (w *SpinnerWidget) Measure(width int) StreamWidgetHeight {
	return StreamWidgetHeight{Height: 1}
}

func (w *SpinnerWidget) Render() string {
	return fitBlock(w.View(), w.width, w.height)
}

func (w *SpinnerWidget) Update(msg tea.Msg) (bool, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		w.width = size.Width
		w.height = size.Height
	}
	return false, nil
}

func (w *SpinnerWidget) View() string {
	if len(w.Frames) == 0 {
		return ""
	}
	return w.Style.Render(w.Frames[w.Frame%len(w.Frames)])
}

func SpinnerFrame(frames []string, frame int, style lipgloss.Style) string {
	return NewSpinnerWidget(frames, style).WithFrame(frame).View()
}

func (w *SpinnerWidget) WithFrame(frame int) *SpinnerWidget {
	w.Frame = frame
	return w
}
