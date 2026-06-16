package common

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"
)

const copyNoticeKey = "copy"

type SelectPos struct{ Line, Col int }

type Selection struct {
	Active       bool
	Anchor, Head SelectPos
}

func (s Selection) Ordered() (start, end SelectPos) {
	if s.Anchor.Line > s.Head.Line || (s.Anchor.Line == s.Head.Line && s.Anchor.Col > s.Head.Col) {
		return s.Head, s.Anchor
	}
	return s.Anchor, s.Head
}

func (s Selection) Empty() bool { return s.Anchor == s.Head }

type SelectionOverlay struct {
	Inner       StreamWidget
	Sel         Selection
	NoticeText  string
	notice      TimedNotice
	cachedLines []string
}

func (s *SelectionOverlay) Measure(width int) StreamWidgetHeight {
	return s.Inner.Measure(width)
}

func (s *SelectionOverlay) Focus() tea.Cmd {
	return Focus(s.Inner)
}

func (s *SelectionOverlay) Blur() {
	Blur(s.Inner)
}

func (s *SelectionOverlay) Render() string {
	rendered := s.Inner.Render()
	if rendered == "" {
		return ""
	}
	s.cachedLines = strings.Split(rendered, "\n")

	lines := make([]string, len(s.cachedLines))
	copy(lines, s.cachedLines)

	if s.Sel.Active && !s.Sel.Empty() {
		start, end := s.Sel.Ordered()
		for i := range lines {
			if lo, hi, ok := selSpan(i, start, end, ansi.StringWidth(lines[i])); ok {
				lines[i] = lipgloss.StyleRanges(lines[i], lipgloss.NewRange(lo, hi, selStyle))
			}
		}
	}

	s.renderCopyNotice(lines)
	return strings.Join(lines, "\n")
}

var selStyle = lipgloss.NewStyle().Reverse(true)

func selSpan(idx int, start, end SelectPos, cw int) (lo, hi int, ok bool) {
	if idx < start.Line || idx > end.Line {
		return 0, 0, false
	}
	lo, hi = 0, cw
	if idx == start.Line {
		lo = start.Col
	}
	if idx == end.Line {
		hi = end.Col
	}
	if hi > cw {
		hi = cw
	}
	if lo >= hi {
		return 0, 0, false
	}
	return lo, hi, true
}

var clipboardWriter = clipboard.WriteAll

func SetClipboardWriter(f func(string) error) func(string) error {
	old := clipboardWriter
	clipboardWriter = f
	return old
}

func copyToClipboard(text string) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			clipboardWriter(text)
			return nil
		},
		tea.SetClipboard(text),
	)
}

func (s *SelectionOverlay) selectedText() string {
	if !s.Sel.Active || s.Sel.Empty() {
		return ""
	}
	start, end := s.Sel.Ordered()
	var out []string
	for idx := start.Line; idx <= end.Line && idx < len(s.cachedLines); idx++ {
		line := s.cachedLines[idx]
		lo, hi := 0, ansi.StringWidth(line)
		if idx == start.Line {
			lo = start.Col
		}
		if idx == end.Line {
			hi = end.Col
		}
		out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(line, lo, hi)), " "))
	}
	return strings.Join(out, "\n")
}

func (s *SelectionOverlay) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {

	case TimedNoticeExpiredMsg:
		if msg.Key == copyNoticeKey {
			s.notice.Clear()
		}
		return false, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseRight && s.Sel.Active && !s.Sel.Empty() {
			text := s.selectedText()
			s.Sel = Selection{}
			_, innerCmd := s.Inner.Update(msg)
			return false, tea.Batch(innerCmd, s.scheduleCopyNotice(text))
		} else if msg.Button == tea.MouseLeft && msg.Y < len(s.cachedLines) {
			s.Sel = Selection{Active: true, Anchor: SelectPos{Line: msg.Y, Col: msg.X}, Head: SelectPos{Line: msg.Y, Col: msg.X}}
		}
		return s.Inner.Update(msg)

	case tea.MouseMotionMsg:
		if s.Sel.Active && msg.Y < len(s.cachedLines) {
			s.Sel.Head = SelectPos{Line: msg.Y, Col: msg.X}
		}
		return s.Inner.Update(msg)

	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft && s.Sel.Active {
			if !s.Sel.Empty() {
				text := s.selectedText()
				s.Sel = Selection{}
				innerChanged, innerCmd := s.Inner.Update(msg)
				return innerChanged, tea.Batch(innerCmd, s.scheduleCopyNotice(text))
			}
			s.Sel = Selection{}
		}
		return s.Inner.Update(msg)
	}

	return s.Inner.Update(msg)
}

func (s *SelectionOverlay) scheduleCopyNotice(text string) tea.Cmd {
	if s.NoticeText == "" {
		return copyToClipboard(text)
	}
	s.notice.Show(s.NoticeText, 2*time.Second)
	return tea.Batch(
		copyToClipboard(text),
		ScheduleTimedNotice(copyNoticeKey, 2*time.Second),
	)
}

func (s *SelectionOverlay) renderCopyNotice(rows []string) {
	if s.NoticeText == "" || len(rows) == 0 || !s.notice.Active(time.Now()) {
		return
	}
	width := ansi.StringWidth(rows[0])
	if width <= 0 {
		return
	}
	textareaBox := NewTextBlockWithPaddingAndMargin(
		s.NoticeText,
		ActiveTheme().TextStyle(),
		BoxStyle{
			Padding: Insets{Top: 1, Right: 2, Bottom: 1, Left: 2},
			Border:  BorderSpec{Left: "┃", Right: "┃", Style: ActiveTheme().AccentStyle()},
			Style:   ActiveTheme().CalloutStyle(),
		}, nil)
	textareaBox.Update(tea.WindowSizeMsg{Width: ansi.StringWidth(s.NoticeText) + 6, Height: 3})
	str := textareaBox.Render()
	x := width - textareaBox.Width
	for i, line := range strings.Split(str, "\n") {
		if i >= len(rows) {
			return
		}
		rows[i] = OverlaySegment(rows[i], x, textareaBox.Width, line)
	}
}
