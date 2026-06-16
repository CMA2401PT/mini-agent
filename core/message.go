package core

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"strings"
)

// ============================================================================
// Message
// ============================================================================

// Message is a single conversation message. Different roles/purposes use
// different concrete types. Type() discriminates for serialization.
type Message interface {
	Role() string // system / user / assistant / tool
	Hash() uint64
	String() string
	Clone() Message
}

// --- concrete Message implementations ---

// TextMsg is a plain text message with a role: system or user.
type TextMsg struct {
	RoleName string // "system" or "user"
	Content  string
}

func (m TextMsg) Role() string { return m.RoleName }

func (m TextMsg) Clone() Message { return m }

func (m TextMsg) Hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.RoleName))
	h.Write([]byte{0})
	h.Write([]byte(m.Content))
	return h.Sum64()
}

// String returns a debug-friendly single-line representation.
func (m TextMsg) String() string {
	return strings.TrimRight(fmt.Sprintf("[%s]: %s", m.RoleName, m.Content), "\n\r\t")
}

// AssistantMsg is a model reply.  It maps 1:1 to the wire assistant message:
// content, reasoning_content, and tool_calls are all fields of the same object.
type AssistantMsg struct {
	Content            string     // visible answer text (may be empty when pure tool calls)
	Reasoning          string     // chain-of-thought content (present with thinking models)
	ReasoningActive    bool       // reasoning has begun (set from content_block_start thinking in Anthropic SSE)
	ReasoningSignature string     // opaque proof for Reasoning (Anthropic thinking signature); empty when N/A
	ToolCalls          []ToolCall // tool invocations (nil or empty when none)
}

func (m AssistantMsg) Role() string { return "assistant" }

func (m AssistantMsg) Clone() Message {
	clone := m
	if m.ToolCalls != nil {
		clone.ToolCalls = make([]ToolCall, len(m.ToolCalls))
		copy(clone.ToolCalls, m.ToolCalls)
	}
	return clone
}

func (msg AssistantMsg) IncreaseFromChunk(c Chunk) AssistantMsg {
	msg.Content += c.Text
	msg.Reasoning += c.Reasoning
	if c.ReasoningActive {
		msg.ReasoningActive = true
	}
	if c.Signature != "" || c.Text != "" {
		msg.ReasoningActive = false
	}
	if c.Signature != "" {
		msg.ReasoningSignature = c.Signature
	}
	if c.ToolCall != nil {
		msg.ToolCalls = append(msg.ToolCalls, *c.ToolCall)
	}
	return msg
}

func (m AssistantMsg) Hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.Content))
	h.Write([]byte{0})
	h.Write([]byte(m.Reasoning))
	h.Write([]byte{0})
	if m.ReasoningActive {
		h.Write([]byte{1})
	}
	h.Write([]byte(m.ReasoningSignature))
	for _, tc := range m.ToolCalls {
		h.Write([]byte{0})
		h.Write([]byte(tc.ID))
		h.Write([]byte{0})
		h.Write([]byte(tc.Name))
		h.Write([]byte{0})
		h.Write([]byte(tc.Arguments))
	}
	return h.Sum64()
}

// String returns a debug-friendly single-line representation.
func (m AssistantMsg) String() string {
	var parts []string
	if m.ReasoningActive {
		s := "reasoning: (active)"
		if m.Reasoning != "" {
			s = "reasoning: " + m.Reasoning
		}
		if m.ReasoningSignature != "" {
			s += fmt.Sprintf(" [sig: %s]", m.ReasoningSignature)
		}
		parts = append(parts, s)
	} else if m.Reasoning != "" {
		s := fmt.Sprintf("reasoning: %s", m.Reasoning)
		if m.ReasoningSignature != "" {
			s += fmt.Sprintf(" [sig: %s]", m.ReasoningSignature)
		}
		parts = append(parts, s)
	}
	if m.Content != "" {
		parts = append(parts, fmt.Sprintf("content: %s", m.Content))
	}
	if len(m.ToolCalls) > 0 {
		names := make([]string, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			names[i] = tc.Name
		}
		parts = append(parts, fmt.Sprintf("tool_call: [%s]", strings.Join(names, ",")))
	}
	if len(parts) == 0 {
		return "[assistant]: (empty)"
	}
	return strings.TrimRight(fmt.Sprintf("[assistant]:\n  %s", strings.Join(parts, "\n  ")), "\n\r\t")
}

// ToolCall represents one tool invocation (ID + function name + JSON arguments).
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResultMsg is a tool execution result, paired to a ToolCallMsg via CallID.
type ToolResultMsg struct {
	CallID  string
	Name    string
	Content string
}

func (m ToolResultMsg) Role() string { return "tool" }

func (m ToolResultMsg) Clone() Message { return m }

func (m ToolResultMsg) Hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.CallID))
	h.Write([]byte{0})
	h.Write([]byte(m.Name))
	h.Write([]byte{0})
	h.Write([]byte(m.Content))
	return h.Sum64()
}

// String returns a debug-friendly single-line representation.
func (m ToolResultMsg) String() string {
	return strings.TrimRight(fmt.Sprintf("[tool_result]: id=%s %s → %s", m.CallID, m.Name, m.Content), "\n\r\t")
}

func MessagesHash(msgs []Message) uint64 {
	h := fnv.New64a()
	for _, m := range msgs {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], m.Hash())
		h.Write(buf[:])
	}
	return h.Sum64()
}

// ProviderTransparentMessage 不发送给 LLM
// 用于工具内部保存意义适合且生命周期&存储方式与对话相同的信息
type ProviderTransparentMessage interface {
	Message
	ProviderTransparentMessage()
}

// TextMsg is a plain text message with a role: system or user.
type TransparentTextMsg struct {
	RoleName string // "system" or "user"
	Content  string
}

func (m TransparentTextMsg) Role() string { return m.RoleName }

func (m TransparentTextMsg) Clone() Message { return m }

func (m TransparentTextMsg) Hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.RoleName))
	h.Write([]byte{0})
	h.Write([]byte(m.Content))
	return h.Sum64()
}

// String returns a debug-friendly single-line representation.
func (m TransparentTextMsg) String() string {
	return strings.TrimRight(fmt.Sprintf("[%s]: %s", m.RoleName, m.Content), "\n\r\t")
}

func (m TransparentTextMsg) ProviderTransparentMessage() {}
