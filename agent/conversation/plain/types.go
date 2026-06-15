package plain

import "mini_agent/core"

type PromptInput struct {
	Prompt   string
	Provider core.Provider
}

func (i PromptInput) UserCommand() {}

type EndConversationCommand struct{}

func (EndConversationCommand) UserCommand() {}
