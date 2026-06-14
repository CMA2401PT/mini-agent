package core

import (
	"context"
)

type ConversationOutput struct {
	BeforeEvent    KeyNotify
	SyncPrimitives TurnSyncPrimitive
	AfterEvent     KeyNotify
}

// 代表可排队、先进先出的用户请求
type UserCommand interface {
	UserCommand()
}

// 提交 Prompt
type PromptCommand struct {
	Messages []Message        // input messages for the turn
	Tools    ToolSetAndRunner // nil = 使用上一个设定
	Provider Provider         // nil = 使用上一个设定
}

func (ic PromptCommand) UserCommand() {}

// 结束会话
type EndConversationCommand struct{}

func (ic EndConversationCommand) UserCommand() {}

// ControlHandle 提供用户对  Conversation 的控制，
// 通过 LockCmds & GetCmds & SetCmds & UnlockCmds 可以实现完全自由的对队列的控制
type ControlHandle interface {
	// 禁止 ConversationCtrl 修改或者读取 命令队列
	LockCmds()
	// SetCmds replaces the command list and wakes the run loop if it is waiting
	SetCmds(cmds []UserCommand)
	// GetCmds returns a copy of the current command list
	GetCmds() []UserCommand
	// 交还 命令队列 的控制权
	UnlockCmds()
	// 打断正在进行的 命令，ConversationCtrl 可以添加相应的打断提示、处理打断并进行下一个命令
	InterruptRunningCmd()
}

// ConversationCtrlTemplate 控制与 LLM 的一次会话中除了模型，工具集和用户操作以外的所有逻辑
// 例如是否需要自动插入提示词，提示词是什么，对于用户的操作如何向 Agent 提供必要的附加信息等等
type ConversationCtrlTemplate interface {
	Emit(
		ctx context.Context,
		initCmds PromptCommand, // 初始用户控制信息
		history []Turn, // 历史会话信息（可以为空）
	) (
		handle ControlHandle, // 控制接口
		output OutStream[ConversationOutput], // 会话信息输出
		err error,
	)
}
