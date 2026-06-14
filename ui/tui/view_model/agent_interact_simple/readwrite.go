package agent_interact_simple

import (
	"strings"
	"time"

	"mini_agent/core"

	"charm.land/bubbles/v2/textarea"
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

type ReadWriteModel struct {
	base        baseSimpleModel
	textarea    textarea.Model
	out         chan<- UserInteract
	streaming   bool
	escCount    int
	lastEscTime time.Time
}

const escTimeout = 500 * time.Millisecond

func NewReadWriteModel(agentEvents core.OutStream[core.ConversationOutput]) (*ReadWriteModel, core.OutStream[UserInteract]) {
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

	m := &ReadWriteModel{
		textarea: ta,
		out:      out,
	}
	m.base = baseSimpleModel{
		mirror:          mirror,
		historyStringer: stringer,
		agentEvents:     agentEvents,
	}
	m.base.initContent(stringer)
	return m, out
}

func (m *ReadWriteModel) emit(event UserInteract) tea.Cmd {
	return func() tea.Msg {
		m.out <- event
		return nil
	}
}

func (m *ReadWriteModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, waitAgentEvent(m.base.agentEvents))
}

func (m *ReadWriteModel) bottomRows() int {
	return m.textarea.Height() + 1
}

func (m *ReadWriteModel) transcriptHeight() int {
	h := m.base.height - m.bottomRows() - 1
	if h > 1 {
		return h
	}
	return 1
}

func (m *ReadWriteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	wasAtBottom := m.base.transcript.Viewport.AtBottom()
	prevWidth := m.base.width

	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.base.width = msg.Width
		m.base.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)

	case tea.MouseWheelMsg:
		m.base.handleScroll(msg)

	case tea.KeyPressMsg:
		cmd = m.handleKeyPress(msg)

	case core.ConversationOutput:
		return m, m.handleSyncEvent(msg)

	case AgentStreamClosedMsg:
		return m, tea.Quit
	}

	m.base.applyViewport(wasAtBottom, prevWidth, m.transcriptHeight())
	return m, cmd
}

func (m *ReadWriteModel) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if m.base.handleNavKeys(msg) {
		return nil
	}
	switch msg.String() {
	case "ctrl+c", "ctrl+d":
		return tea.Batch(m.emit(UserQuit{}), tea.Quit)

	case "enter":
		if m.streaming {
			return nil
		}
		input := strings.TrimSpace(m.textarea.Value())
		if input == "" {
			return tea.Batch(m.emit(UserQuit{}), tea.Quit)
		}
		m.textarea.Reset()
		m.streaming = true
		m.escCount = 0
		return m.emit(UserInput{Prompt: input})

	case "esc":
		return m.handleEsc()

	default:
		if !m.streaming {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m *ReadWriteModel) handleEsc() tea.Cmd {
	if !m.streaming {
		return nil
	}
	now := time.Now()
	if now.Sub(m.lastEscTime) > escTimeout {
		m.escCount = 0
	}
	m.escCount++
	m.lastEscTime = now
	if m.escCount >= 2 {
		m.escCount = 0
		return m.emit(UserInterrupt{})
	}
	return nil
}

func (m *ReadWriteModel) handleSyncEvent(msg core.ConversationOutput) tea.Cmd {
	if _, ok := msg.BeforeEvent.(core.KeyNotifyWaitiningPrompt); ok {
		m.streaming = false
	}
	if _, ok := msg.BeforeEvent.(core.KeyNotifyDone); ok {
		return tea.Quit
	}
	cmd := m.base.syncMirror(msg)
	if _, ok := msg.AfterEvent.(core.KeyNotifyWaitiningPrompt); ok {
		m.streaming = false
	}
	if _, ok := msg.AfterEvent.(core.KeyNotifyDone); ok {
		return tea.Quit
	}
	return cmd
}

func (m *ReadWriteModel) View() tea.View {
	mainArea := m.base.transcript.Viewport.View()
	if mainArea == "" {
		mainArea = strings.Repeat("\n", m.transcriptHeight())
	}

	var bottom strings.Builder
	bottom.WriteString(m.textarea.View())
	bottom.WriteByte('\n')

	if m.streaming {
		switch m.escCount {
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

	v := tea.NewView(mainArea + "\n" + bottom.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
