package common

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

type TimedNotice struct {
	Text  string
	Until time.Time
}

func (n *TimedNotice) Show(text string, duration time.Duration) {
	n.Text = text
	n.Until = time.Now().Add(duration)
}

func (n *TimedNotice) Active(now time.Time) bool {
	return !n.Until.IsZero() && !now.After(n.Until)
}

func (n *TimedNotice) Clear() {
	n.Until = time.Time{}
}

type TimedNoticeExpiredMsg struct {
	Key string
	At  time.Time
}

func ScheduleTimedNotice(key string, duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(now time.Time) tea.Msg {
		return TimedNoticeExpiredMsg{Key: key, At: now}
	})
}
