package common

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type FoldableTextBlock struct {
	*TextBlock
	FullText     string
	FoldedText   string
	foldedPrefix string
	openPrefix   string
	Folded       bool
	onFold       func(folded bool)
}

func NewFoldableTextBlock(fullText, foldedText string, style lipgloss.Style, foldedPrefix, openPrefix string, folded bool, onFold func(folded bool)) *FoldableTextBlock {
	tb := NewTextBlock(fullText, style, nil)
	return &FoldableTextBlock{
		TextBlock:    tb,
		FullText:     fullText,
		FoldedText:   foldedText,
		foldedPrefix: foldedPrefix,
		openPrefix:   openPrefix,
		Folded:       folded,
		onFold:       onFold,
	}
}

func singleLineStr(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func truncateToWidth(text string, maxWidth int, suffix string) string {
	w := ansi.StringWidth(text)
	if w <= maxWidth {
		return text
	}
	sw := ansi.StringWidth(suffix)
	target := maxWidth - sw
	if target < 0 {
		target = 0
	}
	return ansi.Truncate(text, target, suffix)
}

func (f *FoldableTextBlock) isMultiLineAt(width int) bool {
	if f.FullText == "" {
		return false
	}
	w := max(1, width)
	rendered := f.Style.Width(w).Render(f.FullText)
	return strings.Count(rendered, "\n") > 0
}

func (f *FoldableTextBlock) displayText(width int) string {
	if f.Folded {
		txt := f.FoldedText
		if txt == "" {
			txt = singleLineStr(f.FullText)
		}
		return truncateToWidth(f.foldedPrefix+txt, width, "\u2026")
	}
	if f.isMultiLineAt(width) {
		return f.openPrefix + f.FullText
	}
	return f.FullText
}

func (f *FoldableTextBlock) CanFold() bool {
	if f.Folded {
		return true
	}
	return f.isMultiLineAt(f.Width)
}

func (f *FoldableTextBlock) Measure(width int) StreamWidgetHeight {
	if f.Folded {
		return StreamWidgetHeight{Height: 1}
	}
	text := f.displayText(width)
	w := max(1, width)
	rendered := f.Style.Width(w).Render(text)
	return StreamWidgetHeight{Height: len(strings.Split(rendered, "\n"))}
}

func (f *FoldableTextBlock) Render() string {
	text := f.displayText(f.Width)
	w := max(1, f.Width)
	styled := f.Style.Width(w).Render(text)
	return fitBlock(styled, w, f.Height)
}

func (f *FoldableTextBlock) Update(msg tea.Msg) (bool, tea.Cmd) {
	if click, ok := msg.(tea.MouseReleaseMsg); ok {
		if click.Button == tea.MouseLeft && click.X >= 0 && click.X < f.Width && click.Y >= 0 && click.Y < f.Height {
			if !f.CanFold() {
				return false, nil
			}
			f.Folded = !f.Folded
			if f.onFold != nil {
				f.onFold(f.Folded)
			}
			return true, nil
		}
		return false, nil
	}
	return f.TextBlock.Update(msg)
}
