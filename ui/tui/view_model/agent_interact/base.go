package agent_interact

import (
	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/status_bar"
	"mini_agent/ui/tui/view_model/transcript"

	tea "charm.land/bubbletea/v2"
)

type baseInteractModel struct {
	root        *common.StreamColumn
	overlay     *common.SelectionOverlay
	statusbar   *status_bar.StatusBar
	agentEvents core.OutStream[core.ConversationOutput]
}

func newBaseInteractModel(agentEvents core.OutStream[core.ConversationOutput]) *baseInteractModel {
	tscript := transcript.NewTranscript(nil)
	overlay := &common.SelectionOverlay{Inner: tscript, NoticeText: "输出已复制"}
	sbar := status_bar.NewStatusBar()
	return &baseInteractModel{
		overlay:     overlay,
		statusbar:   sbar,
		agentEvents: agentEvents,
	}
}

func (b *baseInteractModel) View() tea.View {
	v := tea.NewView(b.root.Render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

type AgentStreamClosedMsg struct{}

func waitAgentEvent(stream core.OutStream[core.ConversationOutput]) tea.Cmd {
	if stream == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-stream
		if !ok {
			return AgentStreamClosedMsg{}
		}
		return event
	}
}

func isDoneEvent(msg core.ConversationOutput) bool {
	_, beforeDone := msg.BeforeEvent.(core.KeyNotifyDone)
	_, afterDone := msg.AfterEvent.(core.KeyNotifyDone)
	return beforeDone || afterDone
}

func (b *baseInteractModel) updateCommon(
	msg tea.Msg,
	self tea.Model,
	onCtrlC func() tea.Cmd,
	onDone func() tea.Cmd,
) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.BackgroundColorMsg:
		common.SetDarkBackground(msg.IsDark())
		b.root.Update(msg)
		return self, nil

	case core.ConversationOutput:
		_, cmd := b.root.Update(msg)
		if isDoneEvent(msg) {
			return self, tea.Batch(cmd, onDone())
		}
		return self, tea.Batch(cmd, waitAgentEvent(b.agentEvents))

	case AgentStreamClosedMsg:
		return self, tea.Quit

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "super+c", "meta+c", "ctrl+d":
			return self, tea.Batch(onCtrlC(), tea.Quit)
		}
	}

	_, cmd := b.root.Update(msg)
	return self, cmd
}
