package block

import (
	"strings"
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
)

type Block struct {
	TurnData   core.Turn
	phaseState core.TurnPhase
	Width      int
	SectionVisualStates []SectionVisualState
}

type TurnDataMsg struct {
	Turn  core.Turn
	Reset bool
}

type PhaseMsg struct {
	Phase core.TurnPhase
}

func NewBlock(width int) *Block {
	return &Block{Width: width, phaseState: core.TurnPhaseWaitingInput}
}

func (b *Block) Render() string {
	return strings.TrimRight(b.BuildHelper().Render(), "\n")
}

func (b *Block) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case TurnDataMsg:
		b.TurnData = msg.Turn
		if msg.Reset {
			b.SectionVisualStates = nil
		}
		b.BuildHelper().WriteBack(b)
		return true, nil
	case PhaseMsg:
		return b.SetPhase(msg.Phase)

	case common.AnimationTickMsg:
		b.SectionVisualStates = increaseSectionVisualStates(collectTurnSections(b.TurnData), b.SectionVisualStates)
		if b.tickActiveReasoning(time.Now()) {
			return true, nil
		}
		return false, nil

	default:
		helper := b.BuildHelper()
		changed, cmd := helper.Update(msg)
		helper.WriteBack(b)
		return changed, cmd
	}
}

func (b *Block) SetWidth(width int) {
	b.Width = width
}

func (b *Block) SetComplete() (bool, tea.Cmd) {
	return b.SetPhase(core.TurnPhaseFinished)
}

func (b *Block) SetPhase(phase core.TurnPhase) (bool, tea.Cmd) {
	if b.phaseState == phase {
		return false, nil
	}
	b.phaseState = phase
	b.BuildHelper().WriteBack(b)
	return true, nil
}

func (b *Block) IsAnimating() bool {
	sections := collectTurnSections(b.TurnData)
	return b.activeReasoningIndex(sections) >= 0
}

func (b *Block) Phase() core.TurnPhase {
	return b.phaseState
}
