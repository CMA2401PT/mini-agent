package multi_conversation

import (
	"mini_agent/agent/conversation/swarm"
	"mini_agent/core"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/agent_interact"

	tea "charm.land/bubbletea/v2"
)

type TaggedUserInteract struct {
	ConvID       string
	UserInteract agent_interact.UserInteract
}

type MultiConversationWidget struct {
	*common.StreamColumn
	tabBar     *TabBar
	tabView    *common.TabView[*agent_interact.SingleConversationView]
	onInteract func(TaggedUserInteract)
	convID     map[string]int
}

func NewMultiConversationWidget(onInteract func(TaggedUserInteract)) *MultiConversationWidget {
	tabBar := &TabBar{}
	tabView := common.NewTabView[*agent_interact.SingleConversationView]()
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
	idx := w.convID[ev.ConvID]
	view := w.tabView.Items[idx].Widget
	changed, cmd := view.Update(ev.ConversationOutput)
	tabBarEntry := w.tabBar.Entries[idx]
	tabBarEntry.Status = view.Phase()
	return changed, cmd
}

// func (w *MultiConversationWidget) handleInteract(ev InteractEvent) (bool, tea.Cmd) {
// 	switch ev.Type {
// 	case InteractInput:
// 		text := strings.TrimSpace(ev.Prompt)
// 		if strings.HasPrefix(text, "/new") {
// 			title := strings.TrimPrefix(text, "/new")
// 			title = strings.TrimSpace(title)
// 			if title == "" {
// 				title = "新会话"
// 			}
// 			convID, err := w.swarm.StartConversation(context.Background(), []core.UserCommand{
// 				core.PromptInput{Prompt: title},
// 			}, nil)
// 			if err != nil {
// 				return false, nil
// 			}
// 			idx, initCmd := w.createTab(convID, title)
// 			if idx >= 0 {
// 				w.metas[idx].title = title
// 				w.syncTabBarEntry(idx, w.metas[idx])
// 				w.tabView.SwitchTo(idx)
// 				w.tabBar.ActiveIdx = idx
// 			}
// 			return false, initCmd
// 		}

// 		_, ok := w.convID[ev.ConvID]
// 		if !ok {
// 			return false, nil
// 		}
// 		err := w.swarm.SendCommand(core.RoutedUserCommand{
// 			ConvID:  ev.ConvID,
// 			Command: core.PromptInput{Prompt: text},
// 		})
// 		_ = err
// 		return false, nil

// 	case InteractInterrupt:
// 		_ = w.swarm.InterruptConversation(ev.ConvID)
// 		return false, nil

// 	default:
// 		return false, nil
// 	}
// }

func (w *MultiConversationWidget) CreateTab(convID string, title string) int {
	view := agent_interact.NewSingleReadWrite(func(ui agent_interact.UserInteract) {
		ie := TaggedUserInteract{
			ConvID:       convID,
			UserInteract: ui,
		}
		ie.ConvID = convID
		w.onInteract(ie)
	})

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
