package prompt_input

import (
	"testing"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
)

func TestPromptInputSubmitsAndResetsItself(t *testing.T) {
	textarea := common.NewTextareaWidget(common.TextareaWidgetSizeConfig{
		MinHeight:     1,
		MaxHeight:     5,
		DefaultHeight: 1,
	})
	input := NewPromptInput(textarea, common.NewBlockWithPaddingAndMargin(textarea, common.BoxStyle{}, nil))
	input.Update(tea.WindowSizeMsg{Width: 20, Height: 3})
	input.Focus()

	var got string
	input.OnEnter = func(prompt string) tea.Cmd {
		return func() tea.Msg {
			got = prompt
			return nil
		}
	}

	input.Update(testKey('h'))
	input.Update(testKey('i'))
	_, cmd := input.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submit should return OnEnter command")
	}
	cmd()
	if got != "hi" {
		t.Fatalf("submitted prompt = %q, want hi", got)
	}
	if textarea.Value() != "" {
		t.Fatalf("textarea should reset itself, got %q", textarea.Value())
	}
}

func TestPromptInputIgnoresEnterWhenNotAccepting(t *testing.T) {
	textarea := common.NewTextareaWidget(common.TextareaWidgetSizeConfig{
		MinHeight:     1,
		MaxHeight:     5,
		DefaultHeight: 1,
	})
	input := NewPromptInput(textarea, common.NewBlockWithPaddingAndMargin(textarea, common.BoxStyle{}, nil))
	input.Update(tea.WindowSizeMsg{Width: 20, Height: 3})
	input.Focus()

	called := false
	input.OnEnter = func(prompt string) tea.Cmd {
		called = true
		return nil
	}
	input.Update(testKey('h'))
	input.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyRequestSent{}})
	changed, cmd := input.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if changed || cmd != nil || called {
		t.Fatal("input should ignore enter while not accepting input")
	}

	input.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyWaitiningPrompt{}})
	_, _ = input.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !called {
		t.Fatal("input should accept enter after waiting prompt notify")
	}
}

func testKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}
