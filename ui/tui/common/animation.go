package common

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

type AnimationTickMsg time.Time

const animationInterval = 120 * time.Millisecond

func AnimationTickCmd() tea.Cmd {
	return tea.Tick(animationInterval, func(t time.Time) tea.Msg {
		return AnimationTickMsg(t)
	})
}

type ModelWithAnimate[M tea.Model] struct {
	Inner M
}

func (m *ModelWithAnimate[M]) Init() tea.Cmd {
	return tea.Batch(m.Inner.Init(), AnimationTickCmd())
}

func (m *ModelWithAnimate[M]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case AnimationTickMsg:
		innerModel, cmd := m.Inner.Update(msg)
		m.Inner = innerModel.(M)
		return m, tea.Batch(cmd, AnimationTickCmd())
	default:
		innerModel, cmd := m.Inner.Update(msg)
		m.Inner = innerModel.(M)
		return m, cmd
	}
}

func (m *ModelWithAnimate[M]) View() tea.View {
	return m.Inner.View()
}
