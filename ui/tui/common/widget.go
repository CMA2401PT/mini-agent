package common

import tea "charm.land/bubbletea/v2"

type StreamWidget interface {
	Measure(width int) StreamWidgetHeight
	Render() string
	Update(msg tea.Msg) (shouldRemeasure bool, cmd tea.Cmd)
}

type StreamWidgetHeight struct {
	Height     int
	ExpectGrow bool
}

type CanFocus interface {
	Focus() tea.Cmd
	Blur()
}

func Focus(v any) tea.Cmd {
	if f, ok := v.(CanFocus); ok {
		return f.Focus()
	}
	return nil
}

func Blur(v any) {
	if f, ok := v.(CanFocus); ok {
		f.Blur()
	}
}
