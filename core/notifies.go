package core

// KeyNotify 描述会话进行中的关键事件，不会对 []Turn 造成修改，可以被安全丢弃
type KeyNotify interface {
	Event
	KeyNotify()
}

type KeyNotifyWaitingPrompt struct{}

func (e KeyNotifyWaitingPrompt) String() string { return "[Waiting Prompt]" }
func (e KeyNotifyWaitingPrompt) KeyNotify()     {}

// KeyNotifyRequestSent is emitted before each call to provider.Stream.
// It is a control marker and does not change conversation content.
type KeyNotifyRequestSent struct{}

func (e KeyNotifyRequestSent) String() string { return "[Request Sent]" }
func (e KeyNotifyRequestSent) KeyNotify()     {}

type KeyNotifyReasoningStart struct{}

func (e KeyNotifyReasoningStart) String() string { return "[Reasoning Start]" }
func (e KeyNotifyReasoningStart) KeyNotify()     {}

type KeyNotifyReasoningEnd struct{}

func (e KeyNotifyReasoningEnd) String() string { return "[Reasoning End]" }
func (e KeyNotifyReasoningEnd) KeyNotify()     {}

type KeyNotifyOutputStart struct{}

func (e KeyNotifyOutputStart) String() string { return "[Output Start]" }
func (e KeyNotifyOutputStart) KeyNotify()     {}

type KeyNotifyOutputEnd struct{}

func (e KeyNotifyOutputEnd) String() string { return "[Output End]" }
func (e KeyNotifyOutputEnd) KeyNotify()     {}

// KeyNotifyToolUseStart 告知 Provider 输出已经结束，开始进行工具调用
type KeyNotifyToolUseStart struct{}

func (e KeyNotifyToolUseStart) String() string { return "[Tool Use Start]" }
func (e KeyNotifyToolUseStart) KeyNotify()     {}

// KeyNotifyToolUseEnd 告知工具调用已经完成
type KeyNotifyToolUseEnd struct{}

func (e KeyNotifyToolUseEnd) String() string { return "[Tool Use End]" }
func (e KeyNotifyToolUseEnd) KeyNotify()     {}

// KeyNotifyTurnEnd 告知该轮已结束且 Provider 已经给出回答
type KeyNotifyTurnEnd struct{}

func (e KeyNotifyTurnEnd) String() string { return "[Turn End]" }
func (e KeyNotifyTurnEnd) KeyNotify()     {}

// KeyNotifyTurnInterrupted is emitted when the turn is interrupted by the user.
type KeyNotifyTurnInterrupted struct {
	Err error
}

func (e KeyNotifyTurnInterrupted) String() string {
	if e.Err == nil {
		return "[Turn Interrupted]"
	}
	return "[Turn Interrupted]: " + e.Err.Error()
}
func (e KeyNotifyTurnInterrupted) KeyNotify() {}

// 代表出现了一个错误，无法继续（但可能是用户主动打断的）
type KeyNotifyFailure struct {
	Err error
}

func (e KeyNotifyFailure) String() string {
	if e.Err == nil {
		return "[Failure]"
	}
	return "[Failure]: " + e.Err.Error()
}
func (e KeyNotifyFailure) KeyNotify() {}

// 代表对话已经完成
type KeyNotifyDone struct{}

func (e KeyNotifyDone) String() string { return "[Done]" }
func (e KeyNotifyDone) KeyNotify()     {}
