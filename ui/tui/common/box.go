package common

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func fitBlock(in string, width, height int) string {
	return strings.Join(fitLines(textLines(in), width, height), "\n")
}

func FitBlock(in string, width, height int) string {
	return fitBlock(in, width, height)
}

func fitLines(lines []string, width, height int) []string {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	out := make([]string, height)
	blank := strings.Repeat(" ", width)
	for i := 0; i < height; i++ {
		if i >= len(lines) {
			out[i] = blank
			continue
		}
		line := ansi.Truncate(lines[i], width, "")
		pad := width - ansi.StringWidth(line)
		if pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		out[i] = line
	}
	return out
}

func OverlaySegment(line string, x, width int, segment string) string {
	if width <= 0 {
		return line
	}
	lineWidth := ansi.StringWidth(line)
	if x < 0 {
		x = 0
	}
	if x > lineWidth {
		x = lineWidth
	}
	if x+width > lineWidth {
		width = lineWidth - x
	}
	left := ansi.Cut(line, 0, x)
	right := ansi.Cut(line, x+width, lineWidth)
	return left + FitBlock(segment, width, 1) + right
}
