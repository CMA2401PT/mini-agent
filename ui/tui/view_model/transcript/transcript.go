package transcript

import (
	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/common/lazy_scroll_view"
	"mini_agent/ui/tui/view_model/block"

	tea "charm.land/bubbletea/v2"
)

type Transcript struct {
	*lazy_scroll_view.LazyScrollView[*block.Block]
	mirror  *core.TurnMirror
	focused bool
}

func NewTranscript(history []core.Turn) *Transcript {
	t := &Transcript{
		LazyScrollView: lazy_scroll_view.NewLazyScrollView(func(width int) *block.Block {
			return block.NewBlock(width)
		}),
		mirror: core.NewTurnMirror(history),
	}
	if len(history) > 0 {
		_ = t.syncBlocksFromMirror(nil)
	}
	return t
}

func (t Transcript) Measure(width int) common.StreamWidgetHeight {
	return common.StreamWidgetHeight{Height: 1, ExpectGrow: true}
}

func (t *Transcript) Focus() tea.Cmd {
	t.focused = true
	return nil
}

func (t *Transcript) Blur() {
	t.focused = false
}

func (t *Transcript) View() string { return t.LazyScrollView.Render() }

func (t *Transcript) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {

	case core.ConversationOutput:
		return false, t.consumeAgentEvent(msg)

	case tea.KeyPressMsg:
		if t.focused {
			switch msg.String() {
			case "pgup":
				t.PageUp()
			case "pgdown":
				t.PageDown()
			case "up":
				t.ScrollUp(1)
			case "down":
				t.ScrollDown(1)
			}
		}
		return false, nil
	}

	return false, t.LazyScrollView.Update(msg)
}

type BlockTurnUpdate struct {
	Idx      int
	Turn     core.Turn
	Phase    core.TurnPhase
	Complete bool
	Reset    bool
}

func (t *Transcript) consumeAgentEvent(event core.ConversationOutput) tea.Cmd {
	if t.mirror == nil {
		t.mirror = core.NewTurnMirror(nil)
	}
	increased, resetted := t.mirror.ConsumeEvent(event.SyncPrimitives)
	var touched []int
	touched = append(touched, t.consumeKeyNotify(event.BeforeEvent)...)
	touched = append(touched, increased...)
	touched = append(touched, resetted...)
	t.ensurePhaseLen(len(t.mirror.HistoryNoCopy()))
	t.trimBlocksToMirror()
	touched = append(touched, t.consumeKeyNotify(event.AfterEvent)...)
	resetSet := make(map[int]bool, len(resetted))
	for _, i := range resetted {
		resetSet[i] = true
	}
	return tea.Batch(t.syncBlocksFromMirror(resetSet, touched...))
}

func (t *Transcript) consumeKeyNotify(event core.KeyNotify) []int {
	if event == nil || t.mirror == nil {
		return nil
	}
	last := t.mirror.Last()
	if last < 0 {
		return nil
	}
	switch event.(type) {
	case core.KeyNotifyRequestSent:
		return t.setTurnPhase(last, core.TurnPhaseRequesting)
	case core.KeyNotifyReasoningStart:
		return t.setTurnPhase(last, core.TurnPhaseReasoning)
	case core.KeyNotifyOutputStart:
		return t.setTurnPhase(last, core.TurnPhaseOutput)
	case core.KeyNotifyToolUseStart, core.KeyNotifyToolUseEnd:
		return t.setTurnPhase(last, core.TurnPhaseTool)
	case core.KeyNotifyTurnEnd:
		return t.setTurnPhase(last, core.TurnPhaseFinished)
	case core.KeyNotifyTurnInterrupted:
		return t.setTurnPhase(last, core.TurnPhaseFinished)
	case core.KeyNotifyFailure:
		return t.setTurnPhase(last, core.TurnPhaseFinished)
	case core.KeyNotifyDone:
		return t.setTurnPhase(last, core.TurnPhaseFinished)
	}
	return nil
}

func (t *Transcript) setTurnPhase(idx int, phase core.TurnPhase) []int {
	if idx < 0 {
		return nil
	}
	t.EnsureLen(idx + 1)
	b := t.Block(idx)
	changed, _ := b.SetPhase(phase)
	if changed {
		t.InvalidateBlock(idx)
		t.UpdateBlockAnimationState(idx)
		return []int{idx}
	}
	return nil
}

func (t *Transcript) ensurePhaseLen(n int) {
	t.EnsureLen(n)
}

func (t *Transcript) trimBlocksToMirror() {
	n := len(t.mirror.HistoryNoCopy())
	if n < t.BlockCount() {
		t.TruncateBlocks(n)
	}
}

func (t *Transcript) syncBlocksFromMirror(resetSet map[int]bool, indexes ...int) tea.Cmd {
	if t.mirror == nil {
		return nil
	}
	history := t.mirror.HistoryNoCopy()
	t.EnsureLen(len(history))
	if len(indexes) == 0 && t.BlockCount() < len(history) {
		indexes = make([]int, 0, len(history)-t.BlockCount())
		for i := t.BlockCount(); i < len(history); i++ {
			indexes = append(indexes, i)
		}
	}
	if len(indexes) == 0 {
		return nil
	}
	seen := map[int]bool{}
	updates := make([]BlockTurnUpdate, 0, len(indexes))
	for _, i := range indexes {
		if i < 0 || i >= len(history) || seen[i] {
			continue
		}
		seen[i] = true
		b := t.Block(i)
		phase := b.Phase()
		complete := i < len(history)-1 || phase == core.TurnPhaseFinished
		updates = append(updates, BlockTurnUpdate{
			Idx:      i,
			Turn:     history[i],
			Phase:    phase,
			Complete: complete,
			Reset:    resetSet[i],
		})
	}
	return t.SetTurns(updates)
}

func (t *Transcript) SetTurns(turns []BlockTurnUpdate) tea.Cmd {
	var cmds []tea.Cmd
	var changed []int
	for _, tb := range turns {
		t.EnsureLen(tb.Idx + 1)
		b := t.Block(tb.Idx)

		touched := false

		blockChanged, blockCmd := b.Update(block.TurnDataMsg{Turn: tb.Turn, Reset: tb.Reset})
		if blockChanged {
			changed = append(changed, tb.Idx)
			touched = true
		}
		cmds = append(cmds, blockCmd)

		phaseChanged, phaseCmd := b.SetPhase(tb.Phase)
		if phaseChanged {
			changed = append(changed, tb.Idx)
			touched = true
		}
		cmds = append(cmds, phaseCmd)

		if tb.Complete {
			completeChanged, completeCmd := b.SetComplete()
			if completeChanged {
				changed = append(changed, tb.Idx)
				touched = true
			}
			cmds = append(cmds, completeCmd)
		}

		if touched {
			t.UpdateBlockAnimationState(tb.Idx)
		}
	}
	if len(changed) > 0 {
		t.RefreshBlocks(changed...)
	}
	return tea.Batch(cmds...)
}
