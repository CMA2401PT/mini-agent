package multi_conversation

// import (
// 	"mini_agent/core"
// 	"mini_agent/ui/tui/common"

// 	tea "charm.land/bubbletea/v2"
// )

// type MultiConversationModel struct {
// 	widget  *MultiConversationWidget
// 	overlay *common.SelectionOverlay
// 	swarm   core.ConversationSwarm
// 	width   int
// 	height  int
// }

// type SwarmClosedMsg struct{}

// func NewModel(swarm core.ConversationSwarm, initialConvID string) *MultiConversationModel {
// 	w := NewMultiConversationWidget(swarm)
// 	overlay := &common.SelectionOverlay{Inner: w, NoticeText: "输出已复制"}
// 	m := &MultiConversationModel{
// 		widget:  w,
// 		overlay: overlay,
// 		swarm:   swarm,
// 		width:   80,
// 		height:  24,
// 	}
// 	if initialConvID != "" {
// 		w.Update(core.ConversationEvent{
// 			ConvID:             initialConvID,
// 			ConversationOutput: core.ConversationOutput{},
// 		})
// 	}
// 	return m
// }

// func (m *MultiConversationModel) Init() tea.Cmd {
// 	return tea.Batch(
// 		m.listenSwarm(),
// 		m.listenInteract(),
// 		tea.RequestBackgroundColor,
// 	)
// }

// func (m *MultiConversationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	switch msg := msg.(type) {

// 	case tea.WindowSizeMsg:
// 		m.width = msg.Width
// 		m.height = msg.Height
// 		_, cmd := m.overlay.Update(msg)
// 		return m, cmd

// 	case core.ConversationEvent:
// 		_, cmd := m.overlay.Update(msg)
// 		return m, tea.Batch(cmd, m.listenSwarm())

// 	case InteractEvent:
// 		_, cmd := m.widget.Update(msg)
// 		return m, tea.Batch(cmd, m.listenInteract())

// 	case common.AnimationTickMsg:
// 		cid := m.widget.ActiveConvID()
// 		if cid == "" || !m.widget.Finished(cid) {
// 			_, _ = m.overlay.Update(msg)
// 		}
// 		return m, nil

// 	case tea.KeyPressMsg:
// 		switch msg.String() {
// 		case "ctrl+c", "ctrl+d":
// 			return m, tea.Quit
// 		case "ctrl+tab", "ctrl+shift+tab":
// 			_, cmd := m.overlay.Update(msg)
// 			return m, cmd
// 		}
// 		_, cmd := m.overlay.Update(msg)
// 		return m, cmd

// 	case SwarmClosedMsg:
// 		return m, tea.Quit

// 	case tea.BackgroundColorMsg:
// 		common.SetDarkBackground(msg.IsDark())
// 		m.overlay.Update(msg)
// 		return m, nil
// 	}

// 	_, cmd := m.overlay.Update(msg)
// 	return m, cmd
// }

// func (m *MultiConversationModel) View() tea.View {
// 	v := tea.NewView(m.overlay.Render())
// 	v.AltScreen = true
// 	v.MouseMode = tea.MouseModeCellMotion
// 	return v
// }

// func (m *MultiConversationModel) listenSwarm() tea.Cmd {
// 	return func() tea.Msg {
// 		event, ok := <-m.swarm.Output()
// 		if !ok {
// 			return SwarmClosedMsg{}
// 		}
// 		return event
// 	}
// }

// func (m *MultiConversationModel) listenInteract() tea.Cmd {
// 	return func() tea.Msg {
// 		event, ok := <-m.widget.mergedInteract
// 		if !ok {
// 			return nil
// 		}
// 		return event
// 	}
// }
