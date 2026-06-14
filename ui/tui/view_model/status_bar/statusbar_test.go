package status_bar

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestStatusBarAnimationUsesCommonTick(t *testing.T) {
	s := NewStatusBar()
	s.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyRequestSent{}})

	changed, cmd := s.Update(common.AnimationTickMsg(time.Now()))
	if changed || cmd != nil {
		t.Fatal("animation tick should return no further commands")
	}
	if s.spinner != 1 {
		t.Fatalf("spinner after tick = %d, want 1", s.spinner)
	}

	changed, cmd = s.Update(common.AnimationTickMsg(time.Now()))
	if changed || cmd != nil {
		t.Fatal("animation tick should return no further commands")
	}
	if s.spinner != 2 {
		t.Fatalf("spinner after second tick = %d, want 2", s.spinner)
	}
}

func TestStatusBarRendersUsageRightAligned(t *testing.T) {
	s := NewStatusBar()
	s.Update(tea.WindowSizeMsg{Width: 50, Height: 1})
	s.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyRequestSent{}})
	s.Update(core.ConversationOutput{
		SyncPrimitives: core.SyncPrimitiveDeltaChunk{
			Chunk: core.Chunk{
				Reasoning: "thinking",
				Usage:     &core.Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
			},
		},
		AfterEvent: core.KeyNotifyReasoningStart{},
	})

	got := ansi.Strip(s.Render())
	if !strings.HasSuffix(got, "Tokens [输入:10 输出:20 总计30]") {
		t.Fatalf("usage should be right aligned at end: %q", got)
	}
	if !strings.Contains(got, core.TurnPhaseReasoning.String()) {
		t.Fatalf("status should still show phase text with usage: %q", got)
	}
	if ansi.StringWidth(got) != 50 {
		t.Fatalf("status width = %d, want 50: %q", ansi.StringWidth(got), got)
	}
}

func TestStatusBarOwnsEscInterruptState(t *testing.T) {
	s := NewStatusBar()
	s.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyRequestSent{}})

	interrupted := false
	s.OnInterrupt = func() tea.Cmd {
		return func() tea.Msg {
			interrupted = true
			return nil
		}
	}

	if _, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEsc}); cmd != nil {
		t.Fatal("first esc should only update status text")
	}
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("second esc should return interrupt command")
	}
	cmd()
	if !interrupted {
		t.Fatal("second esc should call OnInterrupt")
	}
}

func TestStatusBarSuppressesFailureAfterInterruptUntilNextNotify(t *testing.T) {
	s := NewStatusBar()
	s.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyRequestSent{}})
	s.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyTurnInterrupted{Err: context.Canceled}})
	s.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyFailure{Err: context.Canceled}})

	if s.errText != " · 已打断当前输出" {
		t.Fatalf("failure after interrupt should be suppressed, errText = %q", s.errText)
	}

	s.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyRequestSent{}})
	s.Update(core.ConversationOutput{BeforeEvent: core.KeyNotifyFailure{Err: errors.New("boom")}})

	if !strings.Contains(s.errText, "boom") {
		t.Fatalf("failure after a different notify should be visible, errText = %q", s.errText)
	}
}
