package core

import (
	"context"
	"encoding/json"
	"hash/fnv"
)

// ============================================================================
// Tool
// ============================================================================

// ToolSchema defines a tool exposed to the model. Parameters is JSON Schema.
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func (ts ToolSchema) Hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(ts.Name))
	h.Write([]byte{0})
	h.Write([]byte(ts.Description))
	h.Write([]byte{0})
	h.Write(ts.Parameters)
	return h.Sum64()
}

// Tool is a capability the model can invoke.
type Tool interface {
	Name() string
	Description() string
	// Schema returns the JSON Schema for the tool's parameters.
	Schema() json.RawMessage
	// Execute parses the model-generated raw JSON args and returns result text
	// to feed back to the model.
	Execute(ctx context.Context, args json.RawMessage) (string, error)
	// ReadOnly reports whether the tool has no observable side effects on the
	// host. The agent parallelises a batch of tool calls only when every call
	// in the batch is ReadOnly; mixed batches stay sequential.
	ReadOnly() bool
}

type ToolSetAndRunner interface {
	RunCalls(ctx context.Context, calls []ToolCall) ([]ToolResultMsg, error)
	ToolSchemas() []ToolSchema
	Name() string
}
