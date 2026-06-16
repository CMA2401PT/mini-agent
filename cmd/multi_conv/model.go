package main

import (
	"mini_agent/agent/conversation/swarm"
	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/conversation_multi"
	"mini_agent/ui/tui/view_model/conversation_single"

	tea "charm.land/bubbletea/v2"
)

type MultiConversationModel struct {
	widget    *conversation_multi.MultiConversationWidget
	overlay   *common.SelectionOverlay
	inputChan core.OutStream[swarm.TaggedConversationOutput]
}

type SwarmClosedMsg struct{}

func NewModel(
	onInteract func(conversation_multi.TaggedUserInteract),
	inputChan core.OutStream[swarm.TaggedConversationOutput],
) *MultiConversationModel {
	w := conversation_multi.NewMultiConversationWidget(onInteract, func(onUserInteract func(conversation_single.UserInteract)) conversation_multi.SingleViewWidget {
		if onUserInteract == nil {
			return conversation_single.NewSingleReadOnly()
		}
		return conversation_single.NewSingleReadWrite(onUserInteract)
	})
	overlay := &common.SelectionOverlay{Inner: w, NoticeText: "输出已复制"}
	m := &MultiConversationModel{
		widget:    w,
		overlay:   overlay,
		inputChan: inputChan,
	}
	return m
}

func (m *MultiConversationModel) Init() tea.Cmd {
	return tea.Batch(
		m.listenSwarm(),
		tea.RequestBackgroundColor,
	)
}

type SwitchTab int

func (m *MultiConversationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SwitchTab:
		_, cmd := m.widget.SwitchTab(int(msg))
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		_, cmd := m.overlay.Update(msg)
		return m, cmd

	case swarm.TaggedConversationOutput:
		_, cmd := m.overlay.Update(msg)
		return m, tea.Batch(cmd, m.listenSwarm())

	case tea.BackgroundColorMsg:
		common.SetDarkBackground(msg.IsDark())
		_, cmd := m.overlay.Update(msg)
		return m, cmd
	}

	_, cmd := m.overlay.Update(msg)
	return m, cmd
}

func (m *MultiConversationModel) View() tea.View {
	v := tea.NewView(m.overlay.Render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *MultiConversationModel) listenSwarm() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.inputChan
		if !ok {
			return SwarmClosedMsg{}
		}
		return event
	}
}
