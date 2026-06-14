package common

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type BlockWithPaddingAndMargin struct {
	Child   StreamWidget
	Box     BoxStyle
	OnClick OnClickFunc

	Width, Height int

	childX, childY          int
	childWidth, childHeight int
}

func NewBlockWithPaddingAndMargin(child StreamWidget, box BoxStyle, onClick OnClickFunc) *BlockWithPaddingAndMargin {
	return &BlockWithPaddingAndMargin{Child: child, Box: normalizeBoxStyle(box), OnClick: onClick}
}

func (b *BlockWithPaddingAndMargin) Measure(width int) StreamWidgetHeight {
	childWidth := b.computeChildWidth(width)
	childHeight := 0
	grow := false
	if b.Child != nil {
		measured := b.Child.Measure(childWidth)
		childHeight = max(0, measured.Height)
		grow = measured.ExpectGrow
	}
	return StreamWidgetHeight{Height: childHeight + b.outerVertical(), ExpectGrow: grow}
}

func (b *BlockWithPaddingAndMargin) Render() string {
	width, height := max(0, b.Width), max(0, b.Height)
	if width == 0 || height == 0 {
		return fitBlock("", width, height)
	}

	b.computeChildRect(width, height)
	rows := fitLines(nil, width, height)
	boxLeft := max(0, b.Box.Margin.Left)
	boxTop := max(0, b.Box.Margin.Top)
	boxWidth := max(0, width-b.Box.Margin.horizontal())
	boxHeight := max(0, height-b.Box.Margin.vertical())
	if boxWidth <= 0 || boxHeight <= 0 {
		return strings.Join(rows, "\n")
	}

	contentWidth := max(0, boxWidth-b.Box.Border.horizontal())
	y := boxTop
	if b.Box.Border.topHeight() > 0 && y < height {
		rows[y] = replaceSegment(rows[y], boxLeft, boxWidth, b.Box.Border.renderTop(boxWidth))
		y++
	}

	bodyRows := boxHeight - b.Box.Border.vertical()
	for row := 0; row < bodyRows && y+row < height; row++ {
		line := b.Box.Style.Render(strings.Repeat(" ", contentWidth))
		rows[y+row] = replaceSegment(rows[y+row], boxLeft, boxWidth, b.Box.Border.renderLeft()+line+b.Box.Border.renderRight())
	}

	if b.Child != nil && b.childWidth > 0 && b.childHeight > 0 {
		childBlock := fitBlock(b.Child.Render(), b.childWidth, b.childHeight)
		for yOff, line := range strings.Split(childBlock, "\n") {
			row := b.childY + yOff
			if row < 0 || row >= height {
				continue
			}
			rows[row] = replaceSegment(rows[row], b.childX, b.childWidth, b.Box.Style.Render(fitBlock(line, b.childWidth, 1)))
		}
	}

	bottomY := boxTop + boxHeight - 1
	if b.Box.Border.bottomHeight() > 0 && bottomY >= 0 && bottomY < height {
		rows[bottomY] = replaceSegment(rows[bottomY], boxLeft, boxWidth, b.Box.Border.renderBottom(boxWidth))
	}
	return strings.Join(rows, "\n")
}

func (b *BlockWithPaddingAndMargin) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.Width = msg.Width
		b.Height = msg.Height
		b.computeChildRect(b.Width, b.Height)
		if b.Child == nil {
			return false, nil
		}
		return b.Child.Update(tea.WindowSizeMsg{Width: b.childWidth, Height: b.childHeight})
	case tea.MouseClickMsg:
		if b.Child != nil && b.insideChild(msg.X, msg.Y) {
			return b.Child.Update(OffsetMsg(msg, b.childX, b.childY))
		}
	case tea.MouseMotionMsg:
		if b.Child != nil && b.insideChild(msg.X, msg.Y) {
			return b.Child.Update(OffsetMsg(msg, b.childX, b.childY))
		}
	case tea.MouseReleaseMsg:
		if msg.Button != tea.MouseLeft || !b.insideClickable(msg.X, msg.Y) {
			return false, nil
		}
		if b.Child != nil && b.insideChild(msg.X, msg.Y) {
			childChanged, cmd := b.Child.Update(OffsetMsg(msg, b.childX, b.childY))
			if childChanged || cmd != nil {
				return childChanged, cmd
			}
		}
		if b.OnClick != nil {
			return b.OnClick(msg.X, msg.Y)
		}
	case tea.KeyPressMsg, tea.MouseWheelMsg:
		if b.Child != nil {
			return b.Child.Update(msg)
		}
	default:
		if b.Child != nil {
			return b.Child.Update(msg)
		}
	}
	return false, nil
}

func (b *BlockWithPaddingAndMargin) Focus() tea.Cmd {
	if b.Child == nil {
		return nil
	}
	return focusWidget(b.Child)
}

func (b *BlockWithPaddingAndMargin) Blur() {
	if b.Child != nil {
		blurWidget(b.Child)
	}
}

func (b *BlockWithPaddingAndMargin) computeChildWidth(width int) int {
	return max(1, width-b.outerHorizontal())
}

func (b *BlockWithPaddingAndMargin) computeChildRect(width, height int) {
	b.childX = max(0, b.Box.Margin.Left) + b.Box.Border.leftWidth() + max(0, b.Box.Padding.Left)
	b.childY = max(0, b.Box.Margin.Top) + b.Box.Border.topHeight() + max(0, b.Box.Padding.Top)
	b.childWidth = max(0, width-b.outerHorizontal())
	b.childHeight = max(0, height-b.outerVertical())
}

func (b *BlockWithPaddingAndMargin) outerHorizontal() int {
	return b.Box.Margin.horizontal() + b.Box.Border.horizontal() + b.Box.Padding.horizontal()
}

func (b *BlockWithPaddingAndMargin) outerVertical() int {
	return b.Box.Margin.vertical() + b.Box.Border.vertical() + b.Box.Padding.vertical()
}

func (b *BlockWithPaddingAndMargin) insideClickable(x, y int) bool {
	return x >= max(0, b.Box.Margin.Left) &&
		x < b.Width-max(0, b.Box.Margin.Right) &&
		y >= max(0, b.Box.Margin.Top) &&
		y < b.Height-max(0, b.Box.Margin.Bottom)
}

func (b *BlockWithPaddingAndMargin) insideChild(x, y int) bool {
	return x >= b.childX &&
		x < b.childX+b.childWidth &&
		y >= b.childY &&
		y < b.childY+b.childHeight
}

func normalizeBoxStyle(box BoxStyle) BoxStyle {
	box.Padding = normalizeInsets(box.Padding)
	box.Margin = normalizeInsets(box.Margin)
	return box
}

func repeatCell(cell string, width int) string {
	if cell == "" || width <= 0 {
		return ""
	}
	var out strings.Builder
	for out.Len() < width {
		out.WriteString(cell)
	}
	return fitBlock(out.String(), width, 1)
}

func replaceSegment(line string, x, width int, segment string) string {
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
	return left + fitBlock(segment, width, 1) + right
}

func normalizeInsets(in Insets) Insets {
	return Insets{
		Top:    max(0, in.Top),
		Right:  max(0, in.Right),
		Bottom: max(0, in.Bottom),
		Left:   max(0, in.Left),
	}
}
