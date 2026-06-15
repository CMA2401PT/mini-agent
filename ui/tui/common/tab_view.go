package common

import tea "charm.land/bubbletea/v2"

type TabItem[W StreamWidget] struct {
	Widget W
	ID     string
}

type TabView[W StreamWidget] struct {
	Items     []*TabItem[W]
	ActiveIdx int
	Width     int
	Height    int
}

func NewTabView[W StreamWidget]() *TabView[W] {
	return &TabView[W]{ActiveIdx: -1}
}

func (tv *TabView[W]) AddItem(id string, widget W) int {
	idx := len(tv.Items)
	tv.Items = append(tv.Items, &TabItem[W]{Widget: widget, ID: id})
	if tv.ActiveIdx < 0 {
		tv.ActiveIdx = idx
	}
	return idx
}

func (tv *TabView[W]) RemoveItem(idx int) {
	if idx < 0 || idx >= len(tv.Items) {
		return
	}
	tv.Items = append(tv.Items[:idx], tv.Items[idx+1:]...)
	if tv.ActiveIdx >= len(tv.Items) {
		tv.ActiveIdx = len(tv.Items) - 1
	}
}

func (tv *TabView[W]) ActiveItem() *TabItem[W] {
	if tv.ActiveIdx < 0 || tv.ActiveIdx >= len(tv.Items) {
		return nil
	}
	return tv.Items[tv.ActiveIdx]
}

func (tv *TabView[W]) SwitchTo(idx int) {
	if idx < 0 || idx >= len(tv.Items) || idx == tv.ActiveIdx {
		return
	}
	tv.ActiveIdx = idx
	if item := tv.ActiveItem(); item != nil && tv.Width > 0 && tv.Height > 0 {
		item.Widget.Update(tea.WindowSizeMsg{Width: tv.Width, Height: tv.Height})
	}
}

func (tv *TabView[W]) Measure(width int) StreamWidgetHeight {
	return StreamWidgetHeight{Height: tv.Height, ExpectGrow: true}
}

func (tv *TabView[W]) Render() string {
	if item := tv.ActiveItem(); item != nil {
		return item.Widget.Render()
	}
	return ""
}

func (tv *TabView[W]) Update(msg tea.Msg) (bool, tea.Cmd) {
	if item := tv.ActiveItem(); item != nil {
		return item.Widget.Update(msg)
	}
	return false, nil
}

func (tv *TabView[W]) UpdateItem(idx int, msg tea.Msg) (bool, tea.Cmd) {
	if idx < 0 || idx >= len(tv.Items) {
		return false, nil
	}
	return tv.Items[idx].Widget.Update(msg)
}
