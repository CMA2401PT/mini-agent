package common

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type OnClickFunc func(x, y int) (bool, tea.Cmd)

type TextBlock struct {
	Text    string
	Style   lipgloss.Style
	OnClick OnClickFunc

	Width, Height int
}

func NewTextBlock(text string, style lipgloss.Style, onClick OnClickFunc) *TextBlock {
	return &TextBlock{Text: text, Style: style, OnClick: onClick}
}

func textLines(text string) []string {
	if text == "" {
		return []string{""}
	}
	return strings.Split(text, "\n")
}

func (b *TextBlock) Measure(width int) StreamWidgetHeight {
	return StreamWidgetHeight{Height: len(textLines(b.renderForWidth(width)))}
}

func (b *TextBlock) Render() string {
	return FitBlock(b.renderForWidth(b.Width), b.Width, b.Height)
}

func (b *TextBlock) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		oldHeight := b.Measure(msg.Width).Height
		b.Width = msg.Width
		b.Height = msg.Height
		return oldHeight != b.Measure(msg.Width).Height, nil
	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft && b.OnClick != nil && msg.X >= 0 && msg.X < b.Width && msg.Y >= 0 && msg.Y < b.Height {
			return b.OnClick(msg.X, msg.Y)
		}
	}
	return false, nil
}

func (b *TextBlock) renderForWidth(width int) string {
	return b.Style.Width(max(1, width)).Render(b.Text)
}

type Insets struct {
	Top, Right, Bottom, Left int
}

func (i Insets) horizontal() int {
	return max(0, i.Left) + max(0, i.Right)
}

func (i Insets) vertical() int {
	return max(0, i.Top) + max(0, i.Bottom)
}

type BorderSpec struct {
	Top, Right, Bottom, Left string
	Style                    lipgloss.Style
}

func (b BorderSpec) leftWidth() int {
	if b.Left == "" {
		return 0
	}
	return 1
}

func (b BorderSpec) rightWidth() int {
	if b.Right == "" {
		return 0
	}
	return 1
}

func (b BorderSpec) topHeight() int {
	if b.Top == "" {
		return 0
	}
	return 1
}

func (b BorderSpec) bottomHeight() int {
	if b.Bottom == "" {
		return 0
	}
	return 1
}

func (b BorderSpec) horizontal() int {
	return b.leftWidth() + b.rightWidth()
}

func (b BorderSpec) vertical() int {
	return b.topHeight() + b.bottomHeight()
}

func (b BorderSpec) renderLeft() string {
	if b.Left == "" {
		return ""
	}
	return b.Style.Render(b.Left)
}

func (b BorderSpec) renderRight() string {
	if b.Right == "" {
		return ""
	}
	return b.Style.Render(b.Right)
}

func (b BorderSpec) renderTop(width int) string {
	if b.Top == "" {
		return ""
	}
	return b.Style.Render(repeatCell(b.Top, width))
}

func (b BorderSpec) renderBottom(width int) string {
	if b.Bottom == "" {
		return ""
	}
	return b.Style.Render(repeatCell(b.Bottom, width))
}

type BoxStyle struct {
	Padding Insets
	Margin  Insets
	Border  BorderSpec
	Style   lipgloss.Style
}

type TextBlockWithPaddingAndMargin struct {
	*BlockWithPaddingAndMargin
	Text *TextBlock
}

func NewTextBlockWithPaddingAndMargin(text string, textStyle lipgloss.Style, boxStyle BoxStyle, onClick OnClickFunc) *TextBlockWithPaddingAndMargin {
	child := NewTextBlock(text, textStyle, nil)
	box := NewBlockWithPaddingAndMargin(child, boxStyle, onClick)
	return &TextBlockWithPaddingAndMargin{BlockWithPaddingAndMargin: box, Text: child}
}
