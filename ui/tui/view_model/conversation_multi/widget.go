package conversation_multi

import (
	"mini_agent/agent/conversation/swarm"
	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/conversation_single"

	tea "charm.land/bubbletea/v2"
)

type TaggedUserInteract struct {
	ConvID       string
	UserInteract conversation_single.UserInteract
}

type SingleViewWidget interface {
	common.StreamWidget
	common.CanFocus
	Phase() core.TurnPhase
}

type MultiConversationWidget struct {
	*common.StreamColumn
	tabBar     *TabBar
	tabView    *common.TabView[SingleViewWidget]
	onInteract func(TaggedUserInteract)
	convID     map[string]int
	createView func(onUserInteract func(conversation_single.UserInteract)) SingleViewWidget
}

func NewMultiConversationWidget(
	onInteract func(TaggedUserInteract),
	createView func(onUserInteract func(conversation_single.UserInteract)) SingleViewWidget,
) *MultiConversationWidget {
	tabBar := NewTabBar()
	tabView := common.NewTabView[SingleViewWidget]()
	mainLayout := common.NewStreamColumn(
		tabBar,
		tabView,
	)

	w := &MultiConversationWidget{
		StreamColumn: mainLayout,
		tabBar:       tabBar,
		tabView:      tabView,
		onInteract:   onInteract,
		convID:       make(map[string]int),
		createView:   createView,
	}

	tabBar.OnSwitch = w.SwitchTab
	return w
}

func (w *MultiConversationWidget) SwitchTab(idx int) (bool, tea.Cmd) {
	remeasure, cmd := w.tabView.SwitchTo(idx)
	w.tabBar.ActiveIdx = idx
	return remeasure, tea.Batch(cmd, w.StreamColumn.FocusChild(1))
}

func (w *MultiConversationWidget) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case swarm.TaggedConversationOutput:
		return w.handleConversationEvent(msg)

	default:
		return w.StreamColumn.Update(msg)
	}
}

func (w *MultiConversationWidget) handleConversationEvent(ev swarm.TaggedConversationOutput) (bool, tea.Cmd) {
	idx, ok := w.convID[ev.ConvID]
	if !ok {
		return false, nil
	}
	view := w.tabView.Items[idx].Widget
	changed, cmd := view.Update(ev.ConversationOutput)
	w.tabBar.Entries[idx].Status = view.Phase()
	return changed, cmd
}

func (w *MultiConversationWidget) CreateTab(convID string, title string, readOnly bool) int {
	var view SingleViewWidget

	if readOnly {
		view = w.createView(nil)
	} else {
		view = w.createView(func(ui conversation_single.UserInteract) {
			ie := TaggedUserInteract{
				ConvID:       convID,
				UserInteract: ui,
			}
			w.onInteract(ie)
		})
	}

	if w.tabView.Width > 0 && w.tabView.Height > 0 {
		view.Update(tea.WindowSizeMsg{Width: w.tabView.Width, Height: w.tabView.Height})
	}

	idx := w.tabView.AddItem(convID, view)
	w.convID[convID] = idx
	w.tabBar.Entries = append(w.tabBar.Entries, TabBarEntry{
		ID:     convID,
		Title:  title,
		Status: core.TurnPhaseWaitingInput,
	})

	return idx
}

func (w *MultiConversationWidget) ActiveConvID() string {
	item := w.tabView.ActiveItem()
	if item == nil {
		return ""
	}
	return item.ID
}
