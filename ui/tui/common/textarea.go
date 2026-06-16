package common

import (
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type TextareaWidget struct {
	Textarea textarea.Model
	focused  bool
	width    int
	height   int
	cfg      TextareaConfig
}

type TextareaConfig struct {
	MinHeight     int
	MaxHeight     int
	DefaultHeight int
	Style         *textarea.Styles
	Width         int
}

type TextareaWidgetSizeConfig = TextareaConfig

func NewTextareaWidget(cfg TextareaConfig) *TextareaWidget {
	ta := textarea.New()
	ta.Placeholder = "输入你的问题，回车发送..."
	ta.Prompt = ""
	ta.CharLimit = 16384
	ta.DynamicHeight = cfg.MinHeight != cfg.MaxHeight
	ta.MinHeight = cfg.MinHeight
	ta.MaxHeight = cfg.MaxHeight
	ta.SetHeight(cfg.DefaultHeight)
	ta.ShowLineNumbers = false
	if cfg.Style == nil {
		cfg.Style = PlainTextareaStyles()
	}
	ta.SetStyles(*cfg.Style)
	ta.SetWidth(cfg.Width)

	return &TextareaWidget{
		Textarea: ta,
		height:   cfg.DefaultHeight,
		cfg:      cfg,
	}
}

func (t *TextareaWidget) Measure(width int) StreamWidgetHeight {
	ta := t.Textarea
	if width > 0 {
		ta.SetWidth(width)
		ta, _ = ta.Update(nil)
	}
	y := ta.Height()
	if y < 1 {
		y = 1
	}
	return StreamWidgetHeight{Height: y}
}

func (t *TextareaWidget) Focus() tea.Cmd {
	t.focused = true
	return t.Textarea.Focus()
}

func (t *TextareaWidget) Blur() {
	t.focused = false
	t.Textarea.Blur()
}

func (t *TextareaWidget) Render() string {
	width := max(0, t.width)
	height := max(0, t.height)
	return FitBlock(t.Textarea.View(), width, height)
}

func (t *TextareaWidget) Update(msg tea.Msg) (bool, tea.Cmd) {
	beforeHeight := t.Measure(t.width).Height
	if m, ok := msg.(tea.WindowSizeMsg); ok {
		t.width = m.Width
		t.height = m.Height
		t.Textarea.SetWidth(t.width)
		t.Textarea.SetHeight(max(1, m.Height))
	}

	if _, ok := msg.(tea.BackgroundColorMsg); ok {
		t.Textarea.SetStyles(*PlainTextareaStyles())
		if !t.focused {
			t.Textarea.Blur()
		}
	}

	ta, cmd := t.Textarea.Update(msg)
	t.Textarea = ta

	afterHeight := t.Measure(t.width).Height
	return beforeHeight != afterHeight, cmd
}

func (t *TextareaWidget) Reset() bool {
	beforeHeight := t.Measure(t.width).Height
	t.Textarea.Reset()
	return beforeHeight != t.Measure(t.width).Height
}

func (t *TextareaWidget) Value() string {
	return t.Textarea.Value()
}

func (t *TextareaWidget) SetWidth(w int) {
	t.Textarea.SetWidth(w)
}

func PlainTextareaStyles() *textarea.Styles {
	theme := ActiveTheme()
	style := lipgloss.NewStyle().Foreground(theme.CalloutText).Background(theme.CalloutBackground)
	placeholder := style.Foreground(theme.Placeholder)
	endOfBuffer := style.Foreground(theme.CalloutBackground)
	return &textarea.Styles{
		Focused: textarea.StyleState{
			Base:        style,
			Text:        style,
			CursorLine:  style,
			Placeholder: placeholder,
			Prompt:      style,
			EndOfBuffer: endOfBuffer,
		},
		Blurred: textarea.StyleState{
			Base:        style,
			Text:        style,
			CursorLine:  style,
			Placeholder: placeholder,
			Prompt:      style,
			EndOfBuffer: endOfBuffer,
		},
		Cursor: textarea.CursorStyle{
			Color: theme.CursorColor(),
			Shape: tea.CursorBlock,
			Blink: true,
		},
	}
}
