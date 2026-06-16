package prompt_input

import (
	"strings"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
)

type PromptInput struct {
	textarea *common.TextareaWidget
	box      *common.BlockWithPaddingAndMargin

	accepting bool

	OnEnter      func(prompt string) tea.Cmd
	OnEmptyEnter func() tea.Cmd
}

func NewPromptInput(textarea *common.TextareaWidget, box *common.BlockWithPaddingAndMargin) *PromptInput {
	return &PromptInput{
		textarea:  textarea,
		box:       box,
		accepting: true,
	}
}

func (p *PromptInput) Measure(width int) common.StreamWidgetHeight {
	return p.box.Measure(width)
}

func (p *PromptInput) Render() string {
	return p.box.Render()
}

func (p *PromptInput) Focus() tea.Cmd {
	return p.box.Focus()
}

func (p *PromptInput) Blur() {
	p.box.Blur()
}

func (p *PromptInput) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case core.ConversationOutput:
		p.consumeKeyNotify(msg.BeforeEvent)
		p.consumeKeyNotify(msg.AfterEvent)
		return false, nil
	case tea.BackgroundColorMsg:
		theme := common.ActiveTheme()
		p.box.Box.Style = theme.CalloutStyle()
		p.box.Box.Border.Style = theme.AccentStyle()
		return false, nil
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			return p.submit()
		}
	}
	return p.box.Update(msg)
}

func (p *PromptInput) consumeKeyNotify(event core.KeyNotify) {
	switch event.(type) {
	case core.KeyNotifyWaitingPrompt:
		p.accepting = true
	case core.KeyNotifyDone:
		p.accepting = false
	case core.KeyNotifyRequestSent, core.KeyNotifyReasoningStart, core.KeyNotifyOutputStart, core.KeyNotifyToolUseStart:
		p.accepting = false
	}
}

func (p *PromptInput) submit() (bool, tea.Cmd) {
	if !p.accepting {
		return false, nil
	}
	input := strings.TrimSpace(p.textarea.Value())
	if input == "" {
		if p.OnEmptyEnter == nil {
			return false, nil
		}
		return false, p.OnEmptyEnter()
	}
	p.accepting = false
	changed := p.textarea.Reset()
	var cmd tea.Cmd
	if p.OnEnter != nil {
		cmd = p.OnEnter(input)
	}
	return changed, cmd
}
