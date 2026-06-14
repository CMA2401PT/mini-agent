package agent_interact_simple

import (
	"mini_agent/core"

	tea "charm.land/bubbletea/v2"
)

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

type baseSimpleModel struct {
	transcript      Transcript
	mirror          *core.TurnMirror
	historyStringer *turnsStringerWithCache
	agentEvents     core.OutStream[core.ConversationOutput]
	width           int
	height          int
	content         string
}

func (b *baseSimpleModel) initContent(stringer *turnsStringerWithCache) {
	b.content = stringer.string(nil)
	b.transcript.dirty = true
}

func (b *baseSimpleModel) applyViewport(wasAtBottom bool, prevWidth int, transcriptHeight int) {
	contentW := b.width - 1
	if contentW < 1 {
		contentW = 1
	}
	b.transcript.Viewport.SetWidth(contentW)
	b.transcript.Viewport.SetHeight(transcriptHeight)
	if b.content != "" && (b.transcript.dirty || b.width != prevWidth) {
		b.transcript.SetContent(b.content, contentW)
		if wasAtBottom {
			b.transcript.Viewport.GotoBottom()
		}
	}
}

func (b *baseSimpleModel) handleScroll(msg tea.MouseWheelMsg) {
	switch msg.Button {
	case tea.MouseWheelUp:
		b.transcript.Viewport.ScrollUp(3)
	case tea.MouseWheelDown:
		b.transcript.Viewport.ScrollDown(3)
	}
}

func (b *baseSimpleModel) handleNavKeys(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "pgup":
		b.transcript.Viewport.PageUp()
	case "pgdown":
		b.transcript.Viewport.PageDown()
	case "up":
		b.transcript.Viewport.ScrollUp(1)
	case "down":
		b.transcript.Viewport.ScrollDown(1)
	default:
		return false
	}
	return true
}

func (b *baseSimpleModel) syncMirror(msg core.ConversationOutput) tea.Cmd {
	increased, resetted := b.mirror.ConsumeEvent(msg.SyncPrimitives)
	for _, idx := range increased {
		b.historyStringer.markChange(idx)
	}
	for _, idx := range resetted {
		b.historyStringer.markChange(idx)
	}
	b.transcript.SetContent(b.historyStringer.string(b.mirror.HistoryNoCopy()), b.width)
	return waitAgentEvent(b.agentEvents)
}
