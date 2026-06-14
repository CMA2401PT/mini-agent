package common

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type TextWidgetSizeConfig struct {
	MinHeight     int
	MaxHeight     int
	DefaultHeight int
	ExpectGrow    bool
}

type TextWidget struct {
	text    string
	width   int
	height  int
	focused bool
	cfg     TextWidgetSizeConfig
}

func NewTextWidget(text string, cfgs ...TextWidgetSizeConfig) *TextWidget {
	cfg := defaultTextWidgetSizeConfig(text)
	if len(cfgs) > 0 {
		cfg = normalizeTextWidgetSizeConfig(cfgs[0])
	}
	return &TextWidget{text: text, cfg: cfg}
}

func (w *TextWidget) Measure(width int) StreamWidgetHeight {
	if width <= 0 {
		width = w.assumedWidth()
	}
	height := w.clampHeight(len(w.wrapLines(width)))
	return StreamWidgetHeight{Height: height, ExpectGrow: w.cfg.ExpectGrow}
}

func (w *TextWidget) Focus() tea.Cmd {
	w.focused = true
	return nil
}

func (w *TextWidget) Blur() {
	w.focused = false
}

func (w *TextWidget) Render() string {
	height := w.height
	if height <= 0 {
		height = w.Measure(w.assumedWidth()).Height
	}
	width := w.width
	if width <= 0 {
		width = w.assumedWidth()
	}

	lines := w.wrapLines(width)
	if len(lines) > height && height > 0 {
		lines = lines[:height]
		lines[height-1] = withEllipsis(lines[height-1], width)
	}
	return strings.Join(fitLines(lines, width, height), "\n")
}

func (w *TextWidget) Update(msg tea.Msg) (bool, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		oldHeight := w.Measure(size.Width).Height
		w.width = size.Width
		w.height = size.Height
		return oldHeight != w.Measure(size.Width).Height, nil
	}
	return false, nil
}

func (w *TextWidget) clampHeight(height int) int {
	if height < w.cfg.MinHeight {
		height = w.cfg.MinHeight
	}
	if w.cfg.MaxHeight >= 0 && height > w.cfg.MaxHeight {
		height = w.cfg.MaxHeight
	}
	if height < 1 {
		height = 1
	}
	return height
}

func (w *TextWidget) wrapLines(width int) []string {
	if width < 1 {
		width = 1
	}
	rendered := lipgloss.NewStyle().Width(width).Render(w.text)
	return textLines(rendered)
}

func (w *TextWidget) assumedWidth() int {
	return 1
}

func defaultTextWidgetSizeConfig(text string) TextWidgetSizeConfig {
	lines := textLines(text)
	y := len(lines)
	if y < 1 {
		y = 1
	}
	return TextWidgetSizeConfig{MinHeight: 1, MaxHeight: -1, DefaultHeight: y}
}

func normalizeTextWidgetSizeConfig(cfg TextWidgetSizeConfig) TextWidgetSizeConfig {
	if cfg.MinHeight <= 0 {
		cfg.MinHeight = 1
	}
	if cfg.MaxHeight == 0 {
		cfg.MaxHeight = -1
	}
	if cfg.MaxHeight >= 0 && cfg.MaxHeight < cfg.MinHeight {
		cfg.MaxHeight = cfg.MinHeight
	}
	if cfg.DefaultHeight <= 0 {
		cfg.DefaultHeight = cfg.MinHeight
	}
	cfg.DefaultHeight = clamp(cfg.DefaultHeight, cfg.MinHeight, cfg.MaxHeight)
	return cfg
}

func withEllipsis(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if width <= 3 {
		return ansi.Truncate("...", width, "")
	}
	return ansi.Truncate(line, width-3, "") + "..."
}

func clamp(v, minV, maxV int) int {
	if v < minV {
		v = minV
	}
	if maxV >= 0 && v > maxV {
		v = maxV
	}
	return v
}

func textLines(text string) []string {
	if text == "" {
		return []string{""}
	}
	return strings.Split(text, "\n")
}
