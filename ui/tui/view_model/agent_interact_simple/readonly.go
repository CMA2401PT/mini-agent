package agent_interact_simple

import (
	"strings"

	"mini_agent/core"

	tea "charm.land/bubbletea/v2"
)

type ReadOnlyModel struct {
	base baseSimpleModel
}

func NewReadOnlyModel(agentEvents core.OutStream[core.ConversationOutput]) *ReadOnlyModel {
	mirror := core.NewTurnMirror(nil)
	stringer := newTurnsStringerWithCache(1)
	m := &ReadOnlyModel{}
	m.base = baseSimpleModel{
		mirror:          mirror,
		historyStringer: stringer,
		agentEvents:     agentEvents,
	}
	m.base.initContent(stringer)
	return m
}

func (m *ReadOnlyModel) Init() tea.Cmd {
	return waitAgentEvent(m.base.agentEvents)
}

func (m *ReadOnlyModel) transcriptHeight() int {
	h := m.base.height - 1
	if h > 1 {
		return h
	}
	return 1
}

func (m *ReadOnlyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	wasAtBottom := m.base.transcript.Viewport.AtBottom()
	prevWidth := m.base.width

	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.base.width = msg.Width
		m.base.height = msg.Height

	case tea.MouseWheelMsg:
		m.base.handleScroll(msg)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		}
		m.base.handleNavKeys(msg)

	case core.ConversationOutput:
		if _, ok := msg.BeforeEvent.(core.KeyNotifyDone); ok {
			return m, tea.Quit
		}
		cmd = m.base.syncMirror(msg)
		if _, ok := msg.AfterEvent.(core.KeyNotifyDone); ok {
			return m, tea.Quit
		}
		return m, cmd

	case AgentStreamClosedMsg:
		return m, tea.Quit
	}

	m.base.applyViewport(wasAtBottom, prevWidth, m.transcriptHeight())
	return m, cmd
}

func (m *ReadOnlyModel) View() tea.View {
	mainArea := m.base.transcript.Viewport.View()
	if mainArea == "" {
		mainArea = strings.Repeat("\n", m.transcriptHeight())
	}
	bottom := "[流式输出中，按 Ctrl+C 退出]"
	v := tea.NewView(mainArea + "\n" + bottom)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
