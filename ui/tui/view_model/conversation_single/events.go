package conversation_single

import (
	"mini_agent/core"

	tea "charm.land/bubbletea/v2"
)

type AgentStreamClosedMsg struct{}

func WaitAgentEvent(stream core.OutStream[core.ConversationOutput]) tea.Cmd {
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

func IsDoneEvent(msg core.ConversationOutput) bool {
	_, beforeDone := msg.BeforeEvent.(core.KeyNotifyDone)
	_, afterDone := msg.AfterEvent.(core.KeyNotifyDone)
	return beforeDone || afterDone
}
