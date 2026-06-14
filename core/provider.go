package core

import (
	"context"
	"fmt"
)

// Usage is a token usage record.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Chunk is a single streaming event.  Content deltas (Text, Reasoning, ToolCall)
// accumulate into one AssistantMsg via ChunksToMessages.  Control events are
// discriminated by non-nil fields: Usage ≠ nil → token report; Err ≠ nil → error.
// A closed channel signals the stream ended normally.
type Chunk struct {
	Text            string    // visible text delta
	Reasoning       string    // reasoning delta
	ReasoningActive bool      // reasoning has begun (Anthropic content_block_start thinking)
	Signature       string    // opaque proof for reasoning (Anthropic signature_delta), set alongside Reasoning
	ToolCall        *ToolCall // completed tool call (pre-accumulated by provider)
	Usage           *Usage    // token usage report
	Err             error     // stream error
}

// Request is a single model completion request.  Messages is the flat history
// (callers flatten []Turn via Flatten).
type Request struct {
	Messages    []Message    // flat conversation history
	Tools       []ToolSchema // available tool definitions
	Temperature float64
	MaxTokens   int
}

// Provider abstracts a model backend.  It only exposes Stream; chunk→message
// conversion is the standalone MergeChunks function.
type Provider interface {
	Stream(ctx context.Context, req Request) (OutStream[Chunk], error)
	Name() string
}

// String implements Event. Content deltas (Text, Reasoning) are returned raw so
// that concatenating String() calls across all chunks in a stream produces a
// readable transcript. Control events (ToolCall, Usage, Err) are tagged with a
// [LABEL] prefix.
func (c Chunk) String() string {
	switch {
	case c.Err != nil:
		return fmt.Sprintf("\n[ERROR] %v\n", c.Err)
	case c.Usage != nil:
		return fmt.Sprintf("\n[USAGE] prompt=%d, completion=%d, total=%d\n",
			c.Usage.PromptTokens, c.Usage.CompletionTokens, c.Usage.TotalTokens)
	case c.ToolCall != nil:
		return fmt.Sprintf("\n[TOOL_CALL] %s(%s)\n", c.ToolCall.Name, c.ToolCall.Arguments)
	case c.Reasoning != "":
		return c.Reasoning
	case c.Text != "":
		return c.Text
	default:
		return ""
	}
}
