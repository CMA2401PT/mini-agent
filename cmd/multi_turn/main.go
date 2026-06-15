package main

import (
	"context"
	"fmt"
	"os"

	"mini_agent/agent/conversation/plain"
	"mini_agent/core"
	"mini_agent/providers/openai"
	"mini_agent/ui/tui/view_model/agent_interact"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	handle, stream, err := ctrl.Emit(ctx, nil, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	m, interactStream := agent_interact.NewReadWriteModel(stream)
	go func() {
		for event := range interactStream {
			switch e := event.(type) {
			case agent_interact.UserQuit:
				handle.LockCmds()
				handle.SetCmds([]core.UserCommand{plain.EndConversationCommand{}})
				handle.UnlockCmds()
				handle.InterruptRunningCmd()
			case agent_interact.UserInput:
				handle.LockCmds()
				cmds := handle.GetCmds()
				cmds = append(cmds, plain.PromptInput{Prompt: e.Prompt, Provider: nil})
				handle.SetCmds(cmds)
				handle.UnlockCmds()
			case agent_interact.UserInterrupt:
				handle.InterruptRunningCmd()
			}
		}
	}()
	prog := tea.NewProgram(m)

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
