package block

import (
	"fmt"
	"strings"
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type SectionType uint32

const (
	BlockSectionInput SectionType = 1 << iota
	BlockSectionReasoning
	BlockSectionAnswer
	BlockSectionTools
	BlockSectionSystem
	BlockSectionMeta
)

type SectionVisualState interface {
	Kind() SectionType
	ExpandWidget(section blockSectionModel, width int) common.StreamWidget
}

type InputVisualState struct {
	Folded bool
}

func truncateToOneLine(text string, maxWidth int) string {
	w := ansi.StringWidth(text)
	if w <= maxWidth {
		return text
	}
	runes := []rune(text)
	target := maxWidth - 1
	if target < 0 {
		target = 0
	}
	if target > len(runes) {
		target = len(runes)
	}
	return string(runes[:target]) + "…"
}

func (*InputVisualState) Kind() SectionType {
	return BlockSectionInput
}

func (s *InputVisualState) ExpandWidget(section blockSectionModel, width int) common.StreamWidget {
	content := section.Content
	box := common.BoxStyle{
		Padding: common.Insets{Top: 1, Right: 1, Bottom: 1, Left: 2},
		Border:  common.BorderSpec{Left: "┃", Style: accentStyle()},
		Style:   calloutStyle(),
	}
	if s.Folded {
		content = truncateToOneLine(singleLine(section.Content), width-6)
		box = common.BoxStyle{
			Padding: common.Insets{Right: 1, Left: 2},
			Border:  common.BorderSpec{Left: "┃", Style: accentStyle()},
			Style:   calloutStyle(),
		}
	}
	var widget *common.TextBlockWithPaddingAndMargin
	widget = common.NewTextBlockWithPaddingAndMargin(content, calloutStyle(), box, func(x, y int) (bool, tea.Cmd) {
		if !s.Folded && !canFoldSection(section.Type, section.Content, widget.Text.Width) {
			return false, nil
		}
		s.Folded = !s.Folded
		return true, nil
	})
	return widget
}

type ReasoningVisualState struct {
	Folded    bool
	Spinner   int
	StartedAt time.Time
	EndedAt   time.Time
}

func (*ReasoningVisualState) Kind() SectionType {
	return BlockSectionReasoning
}

func reasoningLines(state *ReasoningVisualState, content string, width int, includeContent bool, folded bool) []string {
	reasoningSeconds := func(state *ReasoningVisualState, now time.Time) float64 {
		if state == nil || state.StartedAt.IsZero() {
			return 0
		}
		if !state.EndedAt.IsZero() {
			return state.EndedAt.Sub(state.StartedAt).Seconds()
		}
		return now.Sub(state.StartedAt).Seconds()
	}
	banner := fmt.Sprintf("Reasoning: %.1fs", reasoningSeconds(state, time.Now()))
	if state != nil && state.EndedAt.IsZero() && !state.StartedAt.IsZero() {
		spin := common.SpinnerFrame(spinnerFrames, state.Spinner, accentStyle())
		banner = fmt.Sprintf("%s %s", spin, banner)
	}
	if folded {
		banner = "▸ " + banner
	}
	out := []string{accentStyle().Render(banner)}
	if !includeContent {
		return out
	}
	for _, line := range wrapLines(content, width) {
		out = append(out, mutedStyle().Render(padANSI(line, width)))
	}
	return out
}

func (s *ReasoningVisualState) ExpandWidget(section blockSectionModel, width int) common.StreamWidget {
	onClick := func(x, y int) (bool, tea.Cmd) {
		s.Folded = !s.Folded
		return true, nil
	}
	banner := common.NewTextBlock(strings.Join(reasoningLines(s, section.Content, width, false, s.Folded), "\n"), accentStyle(), onClick)
	if s.Folded {
		return banner
	}
	content := common.NewTextBlock(section.Content, mutedStyle(), onClick)
	return common.NewStreamColumn(banner, content)
}

type AnswerVisualState struct {
	Folded bool
}

func (*AnswerVisualState) Kind() SectionType {
	return BlockSectionAnswer
}

func (s *AnswerVisualState) ExpandWidget(section blockSectionModel, width int) common.StreamWidget {
	text := section.Content
	if s.Folded {
		prefix := "▸ "
		text = prefix + truncateToOneLine(singleLine(section.Content), width-ansi.StringWidth(prefix)-2)
	}
	var widget *common.TextBlock
	widget = common.NewTextBlock(text, normalStyle(), func(x, y int) (bool, tea.Cmd) {
		if !s.Folded && !canFoldSection(section.Type, section.Content, widget.Width) {
			return false, nil
		}
		s.Folded = !s.Folded
		return true, nil
	})
	return widget
}

type ToolVisualState struct {
	Folded bool
}

func (*ToolVisualState) Kind() SectionType {
	return BlockSectionTools
}

func (s *ToolVisualState) ExpandWidget(section blockSectionModel, width int) common.StreamWidget {
	text := section.Content
	if s.Folded {
		if section.Summary != "" {
			text = section.Summary
		} else {
			prefix := "▸ "
			text = prefix + truncateToOneLine(singleLine(section.Content), width-ansi.StringWidth(prefix)-2)
		}
	}
	return common.NewTextBlock(text, mutedStyle(), func(x, y int) (bool, tea.Cmd) {
		s.Folded = !s.Folded
		return true, nil
	})
}

type SystemVisualState struct {
	Folded bool
}

func (*SystemVisualState) Kind() SectionType {
	return BlockSectionSystem
}

func (s *SystemVisualState) ExpandWidget(section blockSectionModel, width int) common.StreamWidget {
	content := "[系统] " + section.Content
	box := common.BoxStyle{
		Padding: common.Insets{Top: 0, Right: 1, Bottom: 0, Left: 2},
		Border:  common.BorderSpec{Left: " ", Style: nonStyle()},
		Style:   systemStyle(),
	}
	if s.Folded {
		content = truncateToOneLine(singleLine("[系统] " + section.Content), width-6)
		box = common.BoxStyle{
			Padding: common.Insets{Right: 1, Left: 2},
			Border:  common.BorderSpec{Left: " ", Style: nonStyle()},
			Style:   systemStyle(),
		}
	}
	var widget *common.TextBlockWithPaddingAndMargin
	widget = common.NewTextBlockWithPaddingAndMargin(content, systemStyle(), box, func(x, y int) (bool, tea.Cmd) {
		if !s.Folded && !canFoldSection(section.Type, "[系统] "+section.Content, widget.Text.Width) {
			return false, nil
		}
		s.Folded = !s.Folded
		return true, nil
	})
	return widget
}

type MetaVisualState struct{}

func (*MetaVisualState) Kind() SectionType {
	return BlockSectionMeta
}

func (*MetaVisualState) ExpandWidget(section blockSectionModel, width int) common.StreamWidget {
	return common.NewTextBlock(section.Content, mutedStyle(), nil)
}

func sectionFolded(state SectionVisualState) bool {
	switch state := state.(type) {
	case *InputVisualState:
		return state.Folded
	case *ReasoningVisualState:
		return state.Folded
	case *AnswerVisualState:
		return state.Folded
	case *ToolVisualState:
		return state.Folded
	default:
		return false
	}
}

func (b *Block) activeReasoningIndex(sections []blockSectionModel) int {
	if b.phase() != core.TurnPhaseReasoning {
		return -1
	}
	for i := len(sections) - 1; i >= 0; i-- {
		if sections[i].Type == BlockSectionReasoning {
			return sections[i].Index
		}
	}
	return -1
}

func (b *Block) updateReasoningStates(now time.Time) {
	sections := collectTurnSections(b.TurnData)
	active := b.activeReasoningIndex(sections)
	for i, state := range b.SectionVisualStates {
		reasoning, ok := state.(*ReasoningVisualState)
		if !ok {
			continue
		}
		if i == active {
			if reasoning.StartedAt.IsZero() {
				reasoning.StartedAt = now
			}
			reasoning.EndedAt = time.Time{}
			continue
		}
		if !reasoning.StartedAt.IsZero() && reasoning.EndedAt.IsZero() {
			reasoning.EndedAt = now
		}
	}
}

func (b *Block) tickActiveReasoning(now time.Time) bool {
	sections := collectTurnSections(b.TurnData)
	active := b.activeReasoningIndex(sections)
	if active < 0 || active >= len(b.SectionVisualStates) {
		b.updateReasoningStates(now)
		return false
	}
	state, ok := b.SectionVisualStates[active].(*ReasoningVisualState)
	if !ok {
		return false
	}
	if state.StartedAt.IsZero() {
		state.StartedAt = now
	}
	state.EndedAt = time.Time{}
	state.Spinner = (state.Spinner + 1) % len(spinnerFrames)
	return true
}

// 使得 visualStates 数量与 sections 匹配
func increaseSectionVisualStates(sections []blockSectionModel, visualStates []SectionVisualState) []SectionVisualState {
	newSectionVisualState := func(kind SectionType) SectionVisualState {
		switch kind {
		case BlockSectionInput:
			return &InputVisualState{}
		case BlockSectionReasoning:
			return &ReasoningVisualState{}
		case BlockSectionAnswer:
			return &AnswerVisualState{}
		case BlockSectionTools:
			return &ToolVisualState{Folded: true}
		case BlockSectionSystem:
			return &SystemVisualState{}
		case BlockSectionMeta:
			return &MetaVisualState{}
		default:
			return &SystemVisualState{}
		}
	}

	if len(visualStates) > len(sections) {
		visualStates = visualStates[:len(sections)]
	}
	for i, section := range sections {
		if i < len(visualStates) && visualStates[i] != nil && visualStates[i].Kind() == section.Type {
			continue
		}
		if i < len(visualStates) {
			visualStates = visualStates[:i]
		}
		for len(visualStates) <= i {
			next := sections[len(visualStates)]
			visualStates = append(visualStates, newSectionVisualState(next.Type))
		}
	}
	return visualStates
}

func updateReasoningStates(
	now time.Time, activeReasoningIndex int,
	visualStates []SectionVisualState,
) []SectionVisualState {
	for i, state := range visualStates {
		reasoning, ok := state.(*ReasoningVisualState)
		if !ok {
			continue
		}
		if i == activeReasoningIndex {
			if reasoning.StartedAt.IsZero() {
				reasoning.StartedAt = now
			}
			reasoning.EndedAt = time.Time{}
			continue
		}
		if !reasoning.StartedAt.IsZero() && reasoning.EndedAt.IsZero() {
			reasoning.EndedAt = now
		}
	}
	return visualStates
}

func canFoldSection(kind SectionType, content string, width int) bool {
	width = max(1, width)
	switch kind {
	case BlockSectionInput:
		return len(wrapLines(content, width)) > 1
	case BlockSectionReasoning:
		return strings.TrimSpace(content) != ""
	case BlockSectionAnswer:
		return len(wrapLines(content, width)) > 1
	case BlockSectionTools:
		return true
	default:
		return false
	}
}
