package common

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

type Theme struct {
	Text              color.Color
	Muted             color.Color
	Placeholder       color.Color
	Accent            color.Color
	CalloutText       color.Color
	CalloutBackground color.Color
	SystemText        color.Color
	SystemBackground  color.Color
	Cursor            color.Color
	ScrollbarThumb    color.Color
	ScrollbarTrack    color.Color
}

var activeTheme = DefaultTheme()

func DefaultTheme() Theme {
	return Theme{
		Text:              compat.AdaptiveColor{Light: lipgloss.Color("#111111"), Dark: lipgloss.Color("#e6edf3")},
		Muted:             compat.AdaptiveColor{Light: lipgloss.Color("#666666"), Dark: lipgloss.Color("#9aa4ad")},
		Placeholder:       compat.AdaptiveColor{Light: lipgloss.Color("#777777"), Dark: lipgloss.Color("#8b949e")},
		Accent:            compat.AdaptiveColor{Light: lipgloss.Color("#2b6ea6"), Dark: lipgloss.Color("#7cc7ff")},
		CalloutText:       compat.AdaptiveColor{Light: lipgloss.Color("#111111"), Dark: lipgloss.Color("#e6edf3")},
		CalloutBackground: compat.AdaptiveColor{Light: lipgloss.Color("#eeeeee"), Dark: lipgloss.Color("#262a2f")},
		SystemText:        compat.AdaptiveColor{Light: lipgloss.Color("#ffffff"), Dark: lipgloss.Color("#081018")},
		SystemBackground:  compat.AdaptiveColor{Light: lipgloss.Color("#2b6ea6"), Dark: lipgloss.Color("#7cc7ff")},
		Cursor:            compat.AdaptiveColor{Light: lipgloss.Color("#111111"), Dark: lipgloss.Color("#7cc7ff")},
		ScrollbarThumb:    compat.AdaptiveColor{Light: lipgloss.Color("#6c6c6c"), Dark: lipgloss.Color("#8b949e")},
		ScrollbarTrack:    compat.AdaptiveColor{Light: lipgloss.Color("#d0d0d0"), Dark: lipgloss.Color("#3c3c3c")},
	}
}

func ActiveTheme() Theme {
	return activeTheme
}

func SetTheme(theme Theme) {
	activeTheme = theme
}

func SetDarkBackground(dark bool) {
	compat.HasDarkBackground = dark
}

func (t Theme) TextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Text)
}

func (t Theme) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

func (t Theme) PlaceholderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Placeholder)
}

func (t Theme) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent)
}

func (t Theme) CalloutStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.CalloutText).Background(t.CalloutBackground)
}

func (t Theme) SystemStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.SystemText).Background(t.SystemBackground)
}

func (t Theme) CursorColor() color.Color {
	return t.Accent
}
