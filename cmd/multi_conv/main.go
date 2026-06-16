package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"mini_agent/agent/conversation/plain"
	"mini_agent/agent/conversation/swarm"
	"mini_agent/core"
	"mini_agent/providers/openai"
	"mini_agent/ui/tui/common"
	"mini_agent/ui/tui/view_model/conversation_multi"
	"mini_agent/ui/tui/view_model/conversation_single"

	tea "charm.land/bubbletea/v2"
)

func main() {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "DEEPSEEK_API_KEY not set")
		os.Exit(1)
	}

	runner := newToolRegistry(echoTool{}, addTool{})
	p, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "deepseek-v4-pro",
		Effort: "high",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctrl := &plain.PlainConversationCtrl{
		InitSystemPrompt: []core.Turn{
			{core.TextMsg{RoleName: "system", Content: "你是一个精准的助手。需要诚实的回答问题，如果遇到不清楚，不知道的消息，如实说自己不知道。"}},
		},
		InterruptAppendMessage: []core.Message{
			core.TextMsg{RoleName: "system", Content: "该轮输出被打断。"},
		},
		Provider: p,
		Tools:    runner,
	}

	s := swarm.NewSwarmController()
	ctx := context.Background()

	convID, err := s.StartConversation(ctx, nil, nil, ctrl)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start initial conversation:", err)
		os.Exit(1)
	}

	interactStream := make(chan conversation_multi.TaggedUserInteract, 64)
	model := NewModel(func(tui conversation_multi.TaggedUserInteract) {
		interactStream <- tui
	}, s.Output())
	tabIdx := model.widget.CreateTab(convID, "开始")
	// model.widget.SwitchTab(tabIdx)
	convIdx := 0
	prog := tea.NewProgram(&common.ModelWithAnimate[*MultiConversationModel]{Inner: model})
	go func() {
		prog.Send(SwitchTab(tabIdx))
		for event := range interactStream {
			handle := s.GetInstance(event.ConvID)
			switch e := event.UserInteract.(type) {
			case conversation_single.UserQuit:
				handle.LockCmds()
				handle.SetCmds([]core.UserCommand{core.EndConversationCommand{}})
				handle.UnlockCmds()
				handle.InterruptRunningCmd()
			case conversation_single.UserInput:
				p := e.Prompt
				if strings.HasPrefix(p, "/new") {
					convID, err := s.StartConversation(ctx, nil, nil, ctrl)
					if err != nil {
						fmt.Fprintln(os.Stderr, "start initial conversation:", err)
						os.Exit(1)
					}
					convIdx += 1
					tabIdx := model.widget.CreateTab(convID, fmt.Sprintf("%d", convIdx))
					prog.Send(SwitchTab(tabIdx))
					continue
				}
				handle.LockCmds()
				cmds := handle.GetCmds()
				cmds = append(cmds, core.PromptInput{Prompt: e.Prompt, Provider: nil})
				handle.SetCmds(cmds)
				handle.UnlockCmds()
			case conversation_single.UserInterrupt:
				handle.InterruptRunningCmd()
			}
		}
	}()

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
