package common

import tea "charm.land/bubbletea/v2"

type VerticalSpacer struct {
	NRows int
	Width int
}

func NewVerticalSpacer(rows int) *VerticalSpacer {
	return &VerticalSpacer{NRows: rows}
}

func (s *VerticalSpacer) Measure(width int) StreamWidgetHeight {
	return StreamWidgetHeight{Height: max(0, s.NRows)}
}

func (s *VerticalSpacer) Render() string {
	return FitBlock("", s.Width, max(0, s.NRows))
}

func (s *VerticalSpacer) Update(msg tea.Msg) (bool, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		s.Width = size.Width
		s.NRows = size.Height
	}
	return false, nil
}

type VerticalSpring struct {
	Width, Height int
}

func (s *VerticalSpring) Measure(width int) StreamWidgetHeight {
	return StreamWidgetHeight{ExpectGrow: true}
}

func (s *VerticalSpring) Render() string {
	return FitBlock("", s.Width, s.Height)
}

func (s *VerticalSpring) Update(msg tea.Msg) (bool, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		s.Width = size.Width
		s.Height = size.Height
	}
	return false, nil
}
