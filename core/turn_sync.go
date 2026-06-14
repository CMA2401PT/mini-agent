package core

import (
	"fmt"
	"strings"
)

// TurnSyncPrimitive 描述同步两个 Turn 持有者持有的 Turn 所需的事件
// 即，如果 A B 都持有一份相同的 []Turn，
// A 可以通过一系列 TurnSyncPrimitive 将自身持有的 []Turn 信息变化告知 B
type TurnSyncPrimitive interface {
	Event
	TurnSyncPrimitive()
}

func (e ConversationOutput) String() string {
	out := ""
	if e.BeforeEvent != nil {
		out += e.BeforeEvent.String()
	}
	if e.SyncPrimitives != nil {
		if out != "" {
			out += "\n"
		}
		out += e.SyncPrimitives.String()
	}
	if e.AfterEvent != nil {
		if out != "" {
			out += "\n"
		}
		out += e.AfterEvent.String()
	}
	return out
}

// SyncPrimitiveNewTurn 要求构造一个新 Turn
type SyncPrimitiveNewTurn struct {
}

func (e SyncPrimitiveNewTurn) String() string     { return "[Turn Start]" }
func (e SyncPrimitiveNewTurn) TurnSyncPrimitive() {}

// SyncPrimitiveStartResponding 代表已经开始发出响应，应当构造一个空的 AssistantMsg
type SyncPrimitiveStartResponding struct{}

func (e SyncPrimitiveStartResponding) String() string     { return "[Respond Start]" }
func (e SyncPrimitiveStartResponding) TurnSyncPrimitive() {}

// SyncPrimitiveDeltaChunk carries a streaming delta from the provider together with the
// accumulated AssistantMsg snapshot after applying this delta.  Chunk.Usage
// and Chunk.Err are delivered through the same event type.
type SyncPrimitiveDeltaChunk struct {
	Chunk // the current delta
}

func (e SyncPrimitiveDeltaChunk) String() string     { return e.Chunk.String() }
func (e SyncPrimitiveDeltaChunk) TurnSyncPrimitive() {}

// SyncPrimitiveAppendMessages 在最后一个 Turn 中追加信息
type SyncPrimitiveAppendMessages struct {
	Messages []Message
}

func (e SyncPrimitiveAppendMessages) String() string {
	var sb strings.Builder
	for _, r := range e.Messages {
		fmt.Fprintf(&sb, "[Append Messages]:")
		fmt.Fprintf(&sb, "\n  [Message]: %s", r.String())
	}
	return sb.String()
}
func (e SyncPrimitiveAppendMessages) TurnSyncPrimitive() {}

// TurnModifyOrAppendAction 描述对当前 []Turn 的一次替换或追加操作
// Index 为序号，对于一个 len 为 N 的 []Turn, Idx 只能为 0~N (为N时追加一条)
type TurnModifyOrAppendAction struct {
	Index int
	Data  Turn
}

// TurnModifyOrAppendAction 描述对当前 []Turn 的一串替换或追加操作
// 操作逐个进行，每次都在前一次变更基础上变更
// 例如，对于空列表，Action 的 Index 可以为 0,1,2,3,... 但是不能为 1,0,2,3,...
type SyncPrimitiveModifyOrAppendTurns struct {
	Actions []TurnModifyOrAppendAction
}

func (e SyncPrimitiveModifyOrAppendTurns) String() string {
	var sb strings.Builder
	for _, r := range e.Actions {
		fmt.Fprintf(&sb, "[Modify Turn]:")
		fmt.Fprintf(&sb, "\n  [Turn @ %d]: %s", r.Index, r.Data.String())
	}
	return sb.String()
}
func (e SyncPrimitiveModifyOrAppendTurns) TurnSyncPrimitive() {}

// TurnModifyOrAppendAction 描述对当前 []Turn 的一次移除操作
// 移除区间为 [StartIndex,EndIndex)，移除逻辑是从 EndIndex 开始的数据被前移到 StartIndex, *不是置为nil*
// 其中 StartIndex 被移除，而 EndIndex 则前移到 StartIndex
// 需要注意的是尽管没有被移除，但是，从 StartIndex 开始，后续所有数据都变为脏数据
type SyncPrimitiveRemove struct {
	StartIndex int //包括
	EndIndex   int // 不包括
}

func (e SyncPrimitiveRemove) String() string {
	return fmt.Sprintf("[Remove Turn]: [%d,%d)", e.StartIndex, e.EndIndex)
}
func (e SyncPrimitiveRemove) TurnSyncPrimitive() {}

type TurnMirror struct {
	turns []Turn
}

func NewTurnMirror(history []Turn) *TurnMirror {
	return &TurnMirror{
		turns: CloneTurns(history),
	}
}

func (m *TurnMirror) HistoryNoCopy() []Turn {
	return m.turns
}

func (m *TurnMirror) Last() int {
	return len(m.turns) - 1
}

// ConsumeEvent applies a TurnSyncPrimitive to the mirror's local copy of []Turn.
//
//	increased: indices where data was appended but the existing message
//	           order and types remain unchanged — only the last message
//	           grew, or new messages were appended.
//	resetted:  indices where the turn data may have completely changed;
//	           message order, types, or count may differ from before.
func (m *TurnMirror) ConsumeEvent(ev TurnSyncPrimitive) (increased []int, resetted []int) {
	if ev == nil {
		return nil, nil
	}
	switch e := ev.(type) {
	case SyncPrimitiveNewTurn:
		newTurn := Turn{}
		idx := len(m.turns)
		m.turns = append(m.turns, newTurn)
		return nil, []int{idx}

	case SyncPrimitiveAppendMessages:
		last := m.Last()
		m.turns[last] = append(m.turns[last], e.Messages...)
		return []int{last}, nil
	case SyncPrimitiveStartResponding:
		last := m.Last()
		m.turns[last] = append(m.turns[last], AssistantMsg{})
		return []int{last}, nil
	case SyncPrimitiveDeltaChunk:
		last := m.Last()
		msgIdx := len(m.turns[last]) - 1
		msg, ok := m.turns[last][msgIdx].(AssistantMsg)
		if !ok {
			msg = AssistantMsg{}
		}
		msg = msg.IncreaseFromChunk(e.Chunk)
		m.turns[last][msgIdx] = msg
		return []int{last}, nil

	case SyncPrimitiveModifyOrAppendTurns:
		var resetted []int
		for _, action := range e.Actions {
			if action.Index >= 0 && action.Index < len(m.turns) {
				m.turns[action.Index] = action.Data
				resetted = append(resetted, action.Index)
			} else if action.Index == len(m.turns) {
				m.turns = append(m.turns, action.Data)
				resetted = append(resetted, action.Index)
			} else {
				panic(fmt.Errorf("SyncPrimitiveModifyOrAppendTurns: invalid index %d for len %d", action.Index, len(m.turns)))
			}
		}
		return nil, resetted

	case SyncPrimitiveRemove:
		if e.StartIndex < 0 || e.EndIndex <= e.StartIndex || e.EndIndex > len(m.turns) {
			panic(fmt.Errorf("SyncPrimitiveRemove: invalid range [%d,%d) for len %d", e.StartIndex, e.EndIndex, len(m.turns)))
		}
		n := e.EndIndex - e.StartIndex
		copy(m.turns[e.StartIndex:], m.turns[e.EndIndex:])
		for i := len(m.turns) - n; i < len(m.turns); i++ {
			m.turns[i] = nil
		}
		m.turns = m.turns[:len(m.turns)-n]
		var resetted []int
		for i := e.StartIndex; i < len(m.turns); i++ {
			resetted = append(resetted, i)
		}
		return nil, resetted

	default:
		panic(fmt.Errorf("unknown sync primitive: %T: %v", ev, ev))
	}
}
