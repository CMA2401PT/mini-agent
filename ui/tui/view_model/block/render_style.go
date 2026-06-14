package block

import (
	"strings"

	"mini_agent/ui/tui/common"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func singleLine(content string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(content), " "))
}

func wrapLines(s string, width int) []string {
	s = strings.TrimRight(s, "\n\r\t ")
	if s == "" {
		return nil
	}
	return strings.Split(lipgloss.NewStyle().Width(max(1, width)).Render(s), "\n")
}

func padANSI(s string, width int) string {
	s = ansi.Truncate(s, width, "")
	pad := width - ansi.StringWidth(s)
	if pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func nonStyle() lipgloss.Style     { return lipgloss.NewStyle() }
func accentStyle() lipgloss.Style  { return common.ActiveTheme().AccentStyle() }
func calloutStyle() lipgloss.Style { return common.ActiveTheme().CalloutStyle() }
func mutedStyle() lipgloss.Style   { return common.ActiveTheme().MutedStyle() }
func normalStyle() lipgloss.Style  { return common.ActiveTheme().TextStyle() }
func systemStyle() lipgloss.Style  { return common.ActiveTheme().SystemStyle() }
