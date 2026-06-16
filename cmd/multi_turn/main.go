package main

import (
	"context"
	"fmt"
	"os"

	"mini_agent/agent/conversation/plain"
	"mini_agent/core"
	"mini_agent/providers/openai"
	"mini_agent/ui/tui/common"
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

	interactStream := make(core.OutStream[conversation_single.UserInteract], 64)
	singleView := conversation_single.NewSingleReadWrite(func(ui conversation_single.UserInteract) {
		interactStream <- ui
	})
	model := newReadwriteModel(singleView, stream)

	go func() {
		for event := range interactStream {
			switch e := event.(type) {
			case conversation_single.UserQuit:
				handle.LockCmds()
				handle.SetCmds([]core.UserCommand{core.EndConversationCommand{}})
				handle.UnlockCmds()
				handle.InterruptRunningCmd()
			case conversation_single.UserInput:
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

	prog := tea.NewProgram(&common.ModelWithAnimate[*readwriteModel]{Inner: model})

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
