package main

import (
	"context"
	"fmt"
	"os"

	"mini_agent/agent/conversation"
	"mini_agent/core"
	"mini_agent/providers/anthropic"
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
	p, err := anthropic.New(anthropic.Config{
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

	ctrl := &conversation.PlainConversationCtrl{
		InitSystemPrompt: []core.Turn{
			{core.TextMsg{RoleName: "system", Content: "你是一个精准的助手。需要诚实的回答问题，如果遇到不清楚，不知道的消息，如实说自己不知道。"}},
		},
		InterruptAppendMessage: []core.Message{
			core.TextMsg{RoleName: "system", Content: "该轮输出被打断。"},
		},
	}
	handle, stream, err := ctrl.Emit(ctx,
		core.PromptCommand{Provider: p, Tools: runner}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	commands := []string{
		"你是谁？",
		"请依次调用 echo 和 add 两个工具试试。",
		"从1数到100，只输出结果不需要解释。",
	}
	handle.LockCmds()
	cmds := make([]core.UserCommand, 0, len(commands)+1)
	for _, text := range commands {
		cmds = append(cmds, &core.PromptCommand{
			Messages: []core.Message{
				core.TextMsg{RoleName: "user", Content: text},
			},
		})
	}
	cmds = append(cmds, &core.EndConversationCommand{})
	handle.SetCmds(cmds)
	handle.UnlockCmds()

	m := agent_interact.NewReadOnlyModel(stream, false)
	prog := tea.NewProgram(m)

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
