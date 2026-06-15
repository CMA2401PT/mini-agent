package multi_conversation

import (
	"strings"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type TabBarEntry struct {
	ID     string
	Title  string
	Status core.TurnPhase
}

type TabBar struct {
	Entries      []TabBarEntry
	ActiveIdx    int
	Width        int
	OnSwitch     func(idx int) (bool, tea.Cmd)
	spinnerFrame int
}

var tabSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (tb *TabBar) Measure(width int) common.StreamWidgetHeight {
	return common.StreamWidgetHeight{Height: 2, ExpectGrow: false}
}

func (tb *TabBar) Render() string {
	if len(tb.Entries) == 0 || tb.Width <= 0 {
		return ""
	}

	theme := common.ActiveTheme()
	activeStyle := theme.AccentStyle()
	inactiveStyle := theme.MutedStyle()

	indent := 2
	available := tb.Width - indent
	tabWidth := available / max(1, len(tb.Entries))
	if tabWidth > 20 {
		tabWidth = 20
	}
	if tabWidth < 8 {
		tabWidth = 8
	}

	var topParts []string
	var labelParts []string

	for i, entry := range tb.Entries {
		label := entry.Title
		if label == "" {
			label = entry.ID
			if len(label) > 8 {
				label = label[:8]
			}
		}

		indicator := tb.statusIndicator(entry.Status)
		fullText := indicator + " " + label

		maxText := tabWidth - 2
		if maxText < 3 {
			maxText = 3
		}
		if ansi.StringWidth(fullText) > maxText {
			trim := maxText - 1
			if trim < 0 {
				trim = 0
			}
			fullText = indicator + " " + label[:trim] + "\u2026"
		}

		var style = inactiveStyle
		if i == tb.ActiveIdx {
			style = activeStyle
		}

		topLine := style.Render(strings.Repeat("\u2500", tabWidth-2) + "  ")
		topParts = append(topParts, topLine)

		labelText := style.Width(max(1, tabWidth)).Render(fullText)
		labelParts = append(labelParts, labelText)
	}

	indentStr := strings.Repeat(" ", indent)
	labels := lipgloss.NewStyle().Width(tb.Width).Render(indentStr + strings.Join(labelParts, ""))
	buttomLine := lipgloss.NewStyle().Width(tb.Width).Render(indentStr + strings.Join(topParts, ""))

	return labels + "\n" + buttomLine
}

func (tb *TabBar) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tb.Width = msg.Width
	case tea.MouseClickMsg:
		if msg.Y < 0 || msg.Y > 1 {
			break
		}
		if len(tb.Entries) == 0 || tb.Width <= 0 || tb.OnSwitch == nil {
			break
		}
		available := tb.Width - 2
		tabWidth := available / len(tb.Entries)
		if tabWidth > 30 {
			tabWidth = 30
		}
		if tabWidth < 8 {
			tabWidth = 8
		}
		x := msg.X - 2
		if x < 0 {
			break
		}
		idx := x / tabWidth
		if idx >= 0 && idx < len(tb.Entries) {
			return tb.OnSwitch(idx)
		}
	case common.AnimationTickMsg:
		tb.spinnerFrame++
	}
	return false, nil
}

func (tb *TabBar) statusIndicator(status core.TurnPhase) string {
	switch status {
	case core.TurnPhaseWaitingInput:
		return "\u25a1"
	case core.TurnPhaseRequesting, core.TurnPhaseReasoning, core.TurnPhaseOutput, core.TurnPhaseTool:
		return tabSpinnerFrames[tb.spinnerFrame%len(tabSpinnerFrames)]
	case core.TurnPhaseFinished:
		return "\u2713"
	case core.TurnPhaseFailure:
		return "\u2717"
	default:
		return " "
	}
}
