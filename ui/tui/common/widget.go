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
