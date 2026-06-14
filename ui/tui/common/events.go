package common

import tea "charm.land/bubbletea/v2"

func OffsetMsg(msg tea.Msg, ox, oy int) tea.Msg {
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

func focusWidget(widget StreamWidget) tea.Cmd {
	focusable, ok := widget.(CanFocus)
	if !ok {
		return nil
	}
	return focusable.Focus()
}

func blurWidget(widget StreamWidget) {
	focusable, ok := widget.(CanFocus)
	if ok {
		focusable.Blur()
	}
}
