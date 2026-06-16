package agent_interact

import (
	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/prompt_input"
	"mini_agent/ui/tui/view_model/status_bar"
	"mini_agent/ui/tui/view_model/transcript"

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

type SingleConversationView struct {
	root           *common.StreamColumn
	transcript     *transcript.Transcript
	statusbar      *status_bar.StatusBar
	promptInput    *prompt_input.PromptInput
	onUserInteract func(UserInteract)
}

func (v *SingleConversationView) Focus() tea.Cmd {
	if v.promptInput != nil {
		return v.root.Focus()
	}
	return nil
}

func (v *SingleConversationView) Blur() {
	if v.promptInput != nil {
		v.promptInput.Blur()
	}
}

func (v *SingleConversationView) Phase() core.TurnPhase {
	return v.statusbar.Phase
}

func NewSingleReadWrite(onUserInteract func(UserInteract)) *SingleConversationView {
	tscript := transcript.NewTranscript(nil)

	textArea := common.NewTextareaWidget(common.TextareaWidgetSizeConfig{
		MinHeight:     1,
		MaxHeight:     5,
		DefaultHeight: 1,
	})
	textareaBox := common.NewBlockWithPaddingAndMargin(
		textArea,
		common.BoxStyle{
			Padding: common.Insets{Top: 1, Right: 1, Bottom: 1, Left: 2},
			Border:  common.BorderSpec{Left: "\u2503", Style: common.ActiveTheme().AccentStyle()},
			Style:   common.ActiveTheme().CalloutStyle(),
		}, nil)
	pInput := prompt_input.NewPromptInput(textArea, textareaBox)

	sbar := status_bar.NewStatusBar()

	root := common.NewStreamColumn(tscript, common.NewVerticalSpacer(1), pInput, sbar)
	root.FocusChild(2)

	v := &SingleConversationView{
		root:           root,
		transcript:     tscript,
		statusbar:      sbar,
		promptInput:    pInput,
		onUserInteract: onUserInteract,
	}

	pInput.OnEnter = func(prompt string) tea.Cmd {
		v.onUserInteract(UserInput{Prompt: prompt})
		return nil
	}
	pInput.OnEmptyEnter = func() tea.Cmd {
		v.onUserInteract(UserQuit{})
		return nil
	}
	sbar.OnInterrupt = func() tea.Cmd {
		v.onUserInteract(UserInterrupt{})
		return nil
	}

	return v
}

func NewSingleReadOnly() *SingleConversationView {
	tscript := transcript.NewTranscript(nil)
	sbar := status_bar.NewStatusBar()
	sbar.ReadOnly = true

	root := common.NewStreamColumn(tscript, common.NewVerticalSpacer(1), sbar)
	root.FocusChild(0)

	return &SingleConversationView{
		root:       root,
		transcript: tscript,
		statusbar:  sbar,
	}
}

func (v *SingleConversationView) Measure(width int) common.StreamWidgetHeight {
	return v.root.Measure(width)
}

func (v *SingleConversationView) Render() string {
	return v.root.Render()
}

func (v *SingleConversationView) Update(msg tea.Msg) (bool, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "esc" {
		_, sCmd := v.statusbar.Update(msg)
		_, rCmd := v.root.Update(msg)
		return false, tea.Batch(sCmd, rCmd)
	}
	if _, ok := msg.(tea.MouseWheelMsg); ok {
		return v.transcript.Update(msg)
	}
	return v.root.Update(msg)
}
