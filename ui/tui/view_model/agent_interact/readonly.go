package agent_interact

import (
	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
)

type ReadOnlyModel struct {
	*baseInteractModel
	ExitAtEnd bool
}

func NewReadOnlyModel(agentEvents core.OutStream[core.ConversationOutput], exitAtEnd bool) *common.ModelWithAnimate[*ReadOnlyModel] {
	b := newBaseInteractModel(agentEvents)
	b.statusbar.ReadOnly = true
	b.root = common.NewStreamColumn(b.overlay, common.NewVerticalSpacer(1), b.statusbar)
	b.root.FocusChild(0)

	m := &ReadOnlyModel{baseInteractModel: b, ExitAtEnd: exitAtEnd}
	return &common.ModelWithAnimate[*ReadOnlyModel]{Inner: m}
}

func (m *ReadOnlyModel) Init() tea.Cmd {
	return tea.Batch(m.root.Focus(), waitAgentEvent(m.agentEvents), tea.RequestBackgroundColor)
}

func (m *ReadOnlyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	onDone := func() tea.Cmd { return nil }
	if m.ExitAtEnd {
		onDone = func() tea.Cmd { return tea.Quit }
	}
	return m.updateCommon(msg, m, func() tea.Cmd { return nil }, onDone)
}
