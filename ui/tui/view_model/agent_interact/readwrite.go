package agent_interact

import (
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/prompt_input"

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
	*baseInteractModel
	promptInput *prompt_input.PromptInput
	out         chan<- UserInteract
}

func NewReadWriteModel(agentEvents core.OutStream[core.ConversationOutput]) (*common.ModelWithAnimate[*ReadWriteModel], core.OutStream[UserInteract]) {
	b := newBaseInteractModel(agentEvents)
	textArea := common.NewTextareaWidget(common.TextareaWidgetSizeConfig{
		MinHeight:     1,
		MaxHeight:     5,
		DefaultHeight: 1,
	})
	textareaBox := common.NewBlockWithPaddingAndMargin(
		textArea,
		common.BoxStyle{
			Padding: common.Insets{Top: 1, Right: 1, Bottom: 1, Left: 2},
			Border:  common.BorderSpec{Left: "┃", Style: common.ActiveTheme().AccentStyle()},
			Style:   common.ActiveTheme().CalloutStyle(),
		}, nil)
	promptInput := prompt_input.NewPromptInput(textArea, textareaBox)
	b.root = common.NewStreamColumn(b.overlay, common.NewVerticalSpacer(1), promptInput, b.statusbar)
	out := make(chan UserInteract)

	m := &ReadWriteModel{
		baseInteractModel: b,
		promptInput:       promptInput,
		out:               out,
	}
	promptInput.OnEnter = func(prompt string) tea.Cmd {
		return m.emit(UserInput{Prompt: prompt})
	}
	promptInput.OnEmptyEnter = func() tea.Cmd {
		return tea.Batch(m.emit(UserQuit{}), tea.Quit)
	}
	b.statusbar.OnInterrupt = func() tea.Cmd {
		return m.emit(UserInterrupt{})
	}

	b.root.FocusChild(2)

	return &common.ModelWithAnimate[*ReadWriteModel]{Inner: m}, out
}

func (m *ReadWriteModel) Init() tea.Cmd {
	return tea.Batch(m.root.Focus(), waitAgentEvent(m.agentEvents), tea.RequestBackgroundColor, scheduleBgQuery())
}

func (m *ReadWriteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case pollBgMsg:
		return m, tea.Batch(tea.RequestBackgroundColor, scheduleBgQuery())

	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			_, statusCmd := m.statusbar.Update(msg)
			_, rootCmd := m.root.Update(msg)
			return m, tea.Batch(statusCmd, rootCmd)
		}

	case tea.MouseWheelMsg:
		_, cmd := m.overlay.Update(msg)
		return m, cmd
	}

	return m.updateCommon(msg, m,
		func() tea.Cmd { return m.emit(UserQuit{}) },
		func() tea.Cmd { return tea.Quit },
	)
}

func (m *ReadWriteModel) emit(event UserInteract) tea.Cmd {
	return func() tea.Msg {
		m.out <- event
		return nil
	}
}

type pollBgMsg struct{}

const bgQueryInterval = 3 * time.Second

func scheduleBgQuery() tea.Cmd {
	return tea.Tick(bgQueryInterval, func(t time.Time) tea.Msg { return pollBgMsg{} })
}
