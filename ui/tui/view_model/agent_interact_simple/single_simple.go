package agent_interact_simple

import (
	"strings"
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type UserInteract interface {
	UserInteract()
	String() string
}

type UserQuit struct{}

func (q UserQuit) UserInteract()  {}
func (q UserQuit) String() string { return "QuitConversation" }

type UserInput struct {
	Prompt string
}

func (q UserInput) UserInteract()  {}
func (q UserInput) String() string { return "UserInput: " + q.Prompt }

type UserInterrupt struct{}

func (q UserInterrupt) UserInteract()  {}
func (q UserInterrupt) String() string { return "UserInterrupt" }

type SingleConversationSimpleView struct {
	transcript      Transcript
	mirror          *core.TurnMirror
	historyStringer *turnsStringerWithCache
	textarea        textarea.Model
	readonly        bool
	out             chan<- UserInteract
	width           int
	height          int
	streaming       bool
	escCount        int
	lastEscTime     time.Time
}

const escTimeout = 500 * time.Millisecond

func NewSingleSimpleReadWrite() (*SingleConversationSimpleView, core.OutStream[UserInteract]) {
	ta := textarea.New()
	ta.Placeholder = "输入你的问题，回车发送..."
	ta.CharLimit = 16384
	ta.DynamicHeight = true
	ta.MinHeight = 1
	ta.MaxHeight = 5
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Focus()

	mirror := core.NewTurnMirror(nil)
	stringer := newTurnsStringerWithCache(1)
	out := make(chan UserInteract)

	v := &SingleConversationSimpleView{
		textarea:        ta,
		mirror:          mirror,
		historyStringer: stringer,
		out:             out,
		readonly:        false,
	}
	v.transcript = Transcript{Viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))}

	return v, out
}

func NewSingleSimpleReadOnly() *SingleConversationSimpleView {
	mirror := core.NewTurnMirror(nil)
	stringer := newTurnsStringerWithCache(1)

	v := &SingleConversationSimpleView{
		mirror:          mirror,
		historyStringer: stringer,
		readonly:        true,
	}
	v.transcript = Transcript{Viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))}

	return v
}

func (v *SingleConversationSimpleView) Measure(width int) common.StreamWidgetHeight {
	if width <= 0 {
		width = v.width
	}
	h := v.transcript.Viewport.Height()
	if !v.readonly {
		h += v.textarea.Height() + 2
	}
	return common.StreamWidgetHeight{Height: h, ExpectGrow: true}
}

func (v *SingleConversationSimpleView) Render() string {
	mainArea := v.transcript.Viewport.View()
	if mainArea == "" && v.height > 0 {
		mainArea = strings.Repeat("\n", v.transcriptHeight())
	}

	if v.readonly {
		if v.streaming {
			return mainArea + "\n[流式输出中，按 Ctrl+C 退出]"
		}
		return mainArea
	}

	var bottom strings.Builder
	bottom.WriteString(v.textarea.View())
	bottom.WriteByte('\n')

	if v.streaming {
		switch v.escCount {
		case 0:
			bottom.WriteString("[流式输出中，按两下 Esc 可打断]")
		case 1:
			bottom.WriteString("[流式输出中，再次按下 Esc 可打断]")
		default:
			bottom.WriteString("[流式输出中，请等待...]")
		}
	} else {
		bottom.WriteString("(回车发送，Ctrl+C 退出)")
	}

	return mainArea + "\n" + bottom.String()
}

func (v *SingleConversationSimpleView) Update(msg tea.Msg) (bool, tea.Cmd) {
	wasAtBottom := v.transcript.Viewport.AtBottom()
	prevWidth := v.width

	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if !v.readonly {
			v.textarea.SetWidth(msg.Width - 4)
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			v.transcript.Viewport.ScrollUp(3)
		case tea.MouseWheelDown:
			v.transcript.Viewport.ScrollDown(3)
		}

	case tea.KeyPressMsg:
		cmd = v.handleKeyPress(msg)

	case core.ConversationOutput:
		changed, c := v.handleSyncEvent(msg)
		return changed, c
	}

	v.applyViewport(wasAtBottom, prevWidth)
	return false, cmd
}

func (v *SingleConversationSimpleView) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if v.handleNavKeys(msg) {
		return nil
	}
	switch msg.String() {
	case "ctrl+c", "ctrl+d":
		return v.emit(UserQuit{})

	case "enter":
		if v.streaming || v.readonly {
			return nil
		}
		input := strings.TrimSpace(v.textarea.Value())
		if input == "" {
			return v.emit(UserQuit{})
		}
		v.textarea.Reset()
		v.streaming = true
		v.escCount = 0
		return v.emit(UserInput{Prompt: input})

	case "esc":
		return v.handleEsc()

	default:
		if !v.streaming && !v.readonly {
			var tCmd tea.Cmd
			v.textarea, tCmd = v.textarea.Update(msg)
			return tCmd
		}
	}
	return nil
}

func (v *SingleConversationSimpleView) handleEsc() tea.Cmd {
	if !v.streaming {
		return nil
	}
	now := time.Now()
	if now.Sub(v.lastEscTime) > escTimeout {
		v.escCount = 0
	}
	v.escCount++
	v.lastEscTime = now
	if v.escCount >= 2 {
		v.escCount = 0
		return v.emit(UserInterrupt{})
	}
	return nil
}

func (v *SingleConversationSimpleView) handleSyncEvent(msg core.ConversationOutput) (bool, tea.Cmd) {
	if _, ok := msg.BeforeEvent.(core.KeyNotifyWaitiningPrompt); ok {
		v.streaming = false
	}
	if _, ok := msg.AfterEvent.(core.KeyNotifyWaitiningPrompt); ok {
		v.streaming = false
	}
	v.syncMirror(msg)
	return true, nil
}

func (v *SingleConversationSimpleView) syncMirror(msg core.ConversationOutput) {
	increased, resetted := v.mirror.ConsumeEvent(msg.SyncPrimitives)
	for _, idx := range increased {
		v.historyStringer.markChange(idx)
	}
	for _, idx := range resetted {
		v.historyStringer.markChange(idx)
	}
	v.transcript.SetContent(v.historyStringer.string(v.mirror.HistoryNoCopy()), v.width)
}

func (v *SingleConversationSimpleView) handleNavKeys(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "pgup":
		v.transcript.Viewport.PageUp()
	case "pgdown":
		v.transcript.Viewport.PageDown()
	case "up":
		v.transcript.Viewport.ScrollUp(1)
	case "down":
		v.transcript.Viewport.ScrollDown(1)
	default:
		return false
	}
	return true
}

func (v *SingleConversationSimpleView) applyViewport(wasAtBottom bool, prevWidth int) {
	contentW := v.width - 1
	if contentW < 1 {
		contentW = 1
	}
	v.transcript.Viewport.SetWidth(contentW)
	th := v.transcriptHeight()
	v.transcript.Viewport.SetHeight(th)
	if v.width != prevWidth && v.historyStringer != nil {
		v.transcript.SetContent(v.historyStringer.string(v.mirror.HistoryNoCopy()), contentW)
	}
	if wasAtBottom {
		v.transcript.Viewport.GotoBottom()
	}
}

func (v *SingleConversationSimpleView) transcriptHeight() int {
	bottom := 1
	if !v.readonly {
		bottom = v.textarea.Height() + 2
	}
	h := v.height - bottom
	if h < 1 {
		h = 1
	}
	return h
}

func (v *SingleConversationSimpleView) emit(event UserInteract) tea.Cmd {
	return func() tea.Msg {
		v.out <- event
		return nil
	}
}
