package status_bar

import (
	"fmt"
	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type statusBarRenderHelper struct {
	bar    *StatusBar
	layout *common.ColumnRelocatorRoot
}

func newStatusBarRenderHelper(bar *StatusBar) *statusBarRenderHelper {
	h := &statusBarRenderHelper{bar: bar}
	h.layout = h.buildLayout()
	if h.layout != nil {
		h.layout.Update(tea.WindowSizeMsg{Width: bar.width, Height: bar.height})
	}
	return h
}

func (h *statusBarRenderHelper) Render() string {
	if h.layout == nil {
		return common.FitBlock("", h.bar.width, h.bar.height)
	}
	return h.layout.Render()
}

func (h *statusBarRenderHelper) buildLayout() *common.ColumnRelocatorRoot {
	left := h.statusWidget()
	usage := h.usageWidget()
	usageWidth := 0
	if h.bar.usage != nil {
		usageWidth = ansi.StringWidth(usage.Text)
	}

	return common.NewColumnRelocatorRoot(
		common.ColumnRelocator{
			Child: left,
			WidthCompute: func(rootWidth int) (int, int, int) {
				if usageWidth <= 0 {
					return 0, 0, rootWidth
				}
				return 0, 0, max(0, rootWidth-usageWidth-1)
			},
		},
		common.ColumnRelocator{
			Child: usage,
			WidthCompute: func(rootWidth int) (int, int, int) {
				if usageWidth <= 0 {
					return rootWidth, 0, 0
				}
				if usageWidth >= rootWidth {
					return 0, 0, rootWidth
				}
				return rootWidth - usageWidth, 0, usageWidth
			},
		},
	)
}

func (h *statusBarRenderHelper) statusWidget() *common.TextBlock {
	mutedStatusStyle := common.ActiveTheme().MutedStyle()
	accentStatusStyle := common.ActiveTheme().AccentStyle()
	if h.bar.errText != "" {
		text := h.bar.errText
		if !h.bar.ReadOnly {
			text += " · 回车发送 · ctrl+c 退出"
		}
		return common.NewTextBlock(text, mutedStatusStyle, nil)
	} else if h.bar.phase == core.TurnPhaseWaitingInput {
		text := " · 等待输入"
		if !h.bar.ReadOnly {
			text += " · 回车发送 · ctrl+c 退出"
		}
		return common.NewTextBlock(text, mutedStatusStyle, nil)
	} else if h.bar.phase == core.TurnPhaseFinished {
		text := " ✓ 完成"
		if !h.bar.ReadOnly {
			text += " · 回车发送 · ctrl+c 退出"
		}
		return common.NewTextBlock(text, mutedStatusStyle, nil)
	}
	spin := common.SpinnerFrame(statusSpinnerFrames, h.bar.spinner, accentStatusStyle)
	text := "  " + spin + " " + h.bar.phase.String()
	if h.bar.running && !h.bar.ReadOnly {
		if h.bar.escCount == 1 {
			text += " · 再次按下 esc 以打断"
		} else {
			text += " · 按下 esc 打断"
		}
	}
	return common.NewTextBlock(text, accentStatusStyle, nil)
}

func (h *statusBarRenderHelper) usageWidget() *common.TextBlock {
	mutedStatusStyle := common.ActiveTheme().MutedStyle()
	if h.bar.usage == nil {
		return common.NewTextBlock("", mutedStatusStyle, nil)
	}
	return common.NewTextBlock(fmt.Sprintf("Tokens [输入:%d 输出:%d 总计%d]",
		h.bar.usage.PromptTokens,
		h.bar.usage.CompletionTokens,
		h.bar.usage.TotalTokens,
	), mutedStatusStyle, nil)
}
