package main

import (
	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/agent_interact"

	tea "charm.land/bubbletea/v2"
)

type readonlyModel struct {
	view      *agent_interact.SingleConversationView
	overlay   *common.SelectionOverlay
	events    core.OutStream[core.ConversationOutput]
	exitAtEnd bool
}

func newReadonlyModel(
	view *agent_interact.SingleConversationView,
	events core.OutStream[core.ConversationOutput],
	exitAtEnd bool,
) *readonlyModel {
	overlay := &common.SelectionOverlay{Inner: view, NoticeText: "输出已复制"}
	return &readonlyModel{view: view, overlay: overlay, events: events, exitAtEnd: exitAtEnd}
}

func (m *readonlyModel) Init() tea.Cmd {
	return agent_interact.WaitAgentEvent(m.events)
}

func (m *readonlyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case core.ConversationOutput:
		_, cmd := m.overlay.Update(msg)
		if m.exitAtEnd && agent_interact.IsDoneEvent(msg) {
			return m, tea.Batch(cmd, tea.Quit)
		}
		return m, tea.Batch(cmd, agent_interact.WaitAgentEvent(m.events))

	case agent_interact.AgentStreamClosedMsg:
		return m, tea.Quit

	case tea.BackgroundColorMsg:
		common.SetDarkBackground(msg.IsDark())
		m.overlay.Update(msg)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		}
	}

	_, cmd := m.overlay.Update(msg)
	return m, cmd
}

func (m *readonlyModel) View() tea.View {
	v := tea.NewView(m.overlay.Render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
