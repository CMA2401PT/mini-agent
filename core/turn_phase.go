package core

type TurnPhase int

const (
	TurnPhaseWaitingInput TurnPhase = iota
	TurnPhaseRequesting
	TurnPhaseReasoning
	TurnPhaseOutput
	TurnPhaseTool
	TurnPhaseFinished
)

func (p TurnPhase) String() string {
	switch p {
	case TurnPhaseWaitingInput:
		return "等待输入"
	case TurnPhaseRequesting:
		return "请求中"
	case TurnPhaseReasoning:
		return "推理中"
	case TurnPhaseOutput:
		return "输出中"
	case TurnPhaseTool:
		return "工具调用"
	case TurnPhaseFinished:
		return "完成"
	default:
		return "未知状态"
	}
}
