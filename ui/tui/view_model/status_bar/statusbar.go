package status_bar

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
)

type StatusBar struct {
	Phase                         core.TurnPhase
	running                       bool
	errText                       string
	escCount                      int
	spinner                       int
	usage                         *core.Usage
	suppressFailureAfterInterrupt bool
	width                         int
	height                        int
	lastEscTime                   time.Time
	ReadOnly                      bool

	OnInterrupt func() tea.Cmd
}

func NewStatusBar() *StatusBar {
	return &StatusBar{Phase: core.TurnPhaseWaitingInput}
}

func (s *StatusBar) Measure(width int) common.StreamWidgetHeight {
	return common.StreamWidgetHeight{Height: 1}
}

func (s *StatusBar) Focus() tea.Cmd { return nil }
func (s *StatusBar) Blur()          {}

func (s *StatusBar) Render() string {
	helper := newStatusBarRenderHelper(s)
	return helper.Render()
}

func (s *StatusBar) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return false, nil
	case core.ConversationOutput:
		return false, s.consumeAgentEvent(msg)
	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			return false, s.handleEsc()
		}
	case common.AnimationTickMsg:
		if !s.running || s.Phase == core.TurnPhaseWaitingInput || s.Phase == core.TurnPhaseFinished {
			return false, nil
		}
		s.spinner = (s.spinner + 1) % len(statusSpinnerFrames)
		return false, nil
	}
	return false, nil
}

const escTimeout = 500 * time.Millisecond

func (s *StatusBar) consumeAgentEvent(event core.ConversationOutput) tea.Cmd {
	wasRunning := s.running
	s.consumeKeyNotify(event.BeforeEvent)
	if delta, ok := event.SyncPrimitives.(core.SyncPrimitiveDeltaChunk); ok {
		s.setUsage(delta.Chunk.Usage)
	}
	s.consumeKeyNotify(event.AfterEvent)
	if !wasRunning && s.running {
		return nil
	}
	return nil
}

func (s *StatusBar) consumeKeyNotify(event core.KeyNotify) {
	if event == nil {
		return
	}
	if s.suppressFailureAfterInterrupt {
		if _, ok := event.(core.KeyNotifyFailure); ok {
			return
		}
		s.suppressFailureAfterInterrupt = false
	}
	switch e := event.(type) {
	case core.KeyNotifyRequestSent:
		s.running = true
		s.Phase = core.TurnPhaseRequesting
		s.errText = ""
		s.escCount = 0
	case core.KeyNotifyReasoningStart:
		s.running = true
		s.Phase = core.TurnPhaseReasoning
	case core.KeyNotifyOutputStart:
		s.running = true
		s.Phase = core.TurnPhaseOutput
	case core.KeyNotifyToolUseStart, core.KeyNotifyToolUseEnd:
		s.running = true
		s.Phase = core.TurnPhaseTool
	case core.KeyNotifyTurnEnd:
		s.running = false
		s.Phase = core.TurnPhaseFinished
		s.escCount = 0
	case core.KeyNotifyWaitiningPrompt:
		s.running = false
		s.Phase = core.TurnPhaseWaitingInput
		s.escCount = 0
	case core.KeyNotifyDone:
		s.running = false
		s.Phase = core.TurnPhaseFinished
		s.escCount = 0
	case core.KeyNotifyTurnInterrupted:
		s.running = false
		s.Phase = core.TurnPhaseWaitingInput
		s.errText = " · 已打断当前输出"
		s.escCount = 0
		s.suppressFailureAfterInterrupt = true
	case core.KeyNotifyFailure:
		s.running = false
		s.Phase = core.TurnPhaseFailure
		s.escCount = 0
		if e.Err != nil {
			if errors.Is(e.Err, context.Canceled) {
				s.errText = " · 已打断当前输出"
			} else {
				s.errText = fmt.Sprintf(" #错误：%v", e.Err)
			}
		}
	}
}

func (s *StatusBar) setUsage(usage *core.Usage) {
	if usage == nil {
		return
	}
	u := *usage
	s.usage = &u
}

func (s *StatusBar) handleEsc() tea.Cmd {
	if !s.running {
		return nil
	}
	now := time.Now()
	if now.Sub(s.lastEscTime) > escTimeout {
		s.escCount = 0
	}
	s.escCount++
	s.lastEscTime = now
	if s.escCount < 2 {
		return nil
	}
	s.escCount = 0
	if s.OnInterrupt == nil {
		return nil
	}
	return s.OnInterrupt()
}

var statusSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
