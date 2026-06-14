package block

import (
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
)

type BlockRenderHelper struct {
	visualStates []SectionVisualState
	*common.StreamColumn
}

func NewBlockRenderHelper(block *Block) *BlockRenderHelper {
	sections := collectTurnSections(block.TurnData)
	visualStates := increaseSectionVisualStates(sections, block.SectionVisualStates)
	activateReasoningIdx := -1
	if block.phase() == core.TurnPhaseReasoning {
		activateReasoningIdx = sections.GetLastReasoning()
	}
	visualStates = updateReasoningStates(time.Now(), activateReasoningIdx, visualStates)
	layout := buildLayout(sections, visualStates, block.Width)
	return &BlockRenderHelper{
		visualStates: visualStates,
		StreamColumn: layout,
	}
}

func (b *Block) RetrieveVisualState(helper *BlockRenderHelper) {
	b.Width = helper.Width
	b.SectionVisualStates = helper.visualStates
}

func buildLayout(sections blockSections, visualStates []SectionVisualState, width int) *common.StreamColumn {
	var widgets []common.StreamWidget
	y := 0
	for i, section := range sections {
		visualState := visualStates[i]
		widget := visualState.ExpandWidget(section, width)
		var sectionWidget common.StreamWidget = widget
		if sections[i].Type != BlockSectionInput && sections[i].Type != BlockSectionSystem && sections[i].Type != BlockSectionMeta {
			sectionWidget = common.NewColumnRelocatorRoot(common.ColumnRelocator{
				Child:        widget,
				WidthCompute: common.InsetWidth(2),
			})
		}

		height := sectionWidget.Measure(width).Height
		sections[i].StartY = y
		sections[i].Height = height
		widgets = append(widgets, sectionWidget)
		y += height
		if i < len(sections)-1 && !adjacentToolSections(sections[i], sections[i+1]) {
			widgets = append(widgets, common.NewVerticalSpacer(1))
			y++
		}
	}
	layout := common.NewStreamColumn(widgets...)
	height := layout.Measure(width).Height
	layout.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return layout
}

func adjacentToolSections(left, right blockSectionModel) bool {
	return left.Type == BlockSectionTools && right.Type == BlockSectionTools
}
