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

func OffsetMouseMsg(msg tea.Msg, ox, oy int) tea.Msg {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		msg.X -= ox
		msg.Y -= oy
		return msg
	case tea.MouseMotionMsg:
		msg.X -= ox
		msg.Y -= oy
		return msg
	case tea.MouseReleaseMsg:
		msg.X -= ox
		msg.Y -= oy
		return msg
	default:
		return msg
	}
}

func mouseX(msg tea.Msg) int {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		return msg.X
	case tea.MouseMotionMsg:
		return msg.X
	case tea.MouseReleaseMsg:
		return msg.X
	default:
		return 0
	}
}

func mouseY(msg tea.Msg) int {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		return msg.Y
	case tea.MouseMotionMsg:
		return msg.Y
	case tea.MouseReleaseMsg:
		return msg.Y
	default:
		return 0
	}
}
