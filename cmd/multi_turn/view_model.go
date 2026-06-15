package main

import (
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/agent_interact"

	tea "charm.land/bubbletea/v2"
)

type readwriteModel struct {
	view    *agent_interact.SingleConversationView
	overlay *common.SelectionOverlay
	events  core.OutStream[core.ConversationOutput]
}

func newReadwriteModel(
	view *agent_interact.SingleConversationView,
	events core.OutStream[core.ConversationOutput],
) *readwriteModel {
	overlay := &common.SelectionOverlay{Inner: view, NoticeText: "输出已复制"}
	return &readwriteModel{view: view, overlay: overlay, events: events}
}

func (m *readwriteModel) Init() tea.Cmd {
	return tea.Batch(
		agent_interact.WaitAgentEvent(m.events),
		scheduleBgQuery(),
	)
}

func (m *readwriteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case pollBgMsg:
		return m, tea.Batch(tea.RequestBackgroundColor, scheduleBgQuery())

	case core.ConversationOutput:
		_, cmd := m.overlay.Update(msg)
		return m, tea.Batch(cmd, agent_interact.WaitAgentEvent(m.events))

	case agent_interact.AgentStreamClosedMsg:
		return m, tea.Quit

	case tea.BackgroundColorMsg:
		common.SetDarkBackground(msg.IsDark())
		m.overlay.Update(msg)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "super+c", "meta+c", "ctrl+d":
			return m, tea.Quit
		}
	}

	_, cmd := m.overlay.Update(msg)
	return m, cmd
}

func (m *readwriteModel) View() tea.View {
	v := tea.NewView(m.overlay.Render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

type pollBgMsg struct{}

const bgQueryInterval = 3 * time.Second

func scheduleBgQuery() tea.Cmd {
	return tea.Tick(bgQueryInterval, func(t time.Time) tea.Msg { return pollBgMsg{} })
}
