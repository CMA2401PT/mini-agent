package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"mini_agent/core"
)

// ============================================================================
// Tool registry — implements core.ToolSetAndRunner
// ============================================================================

type toolRegistry struct {
	tools   map[string]core.Tool
	order   []string
	schemas []core.ToolSchema
}

func newToolRegistry(tools ...core.Tool) *toolRegistry {
	r := &toolRegistry{tools: map[string]core.Tool{}}
	for _, t := range tools {
		name := t.Name()
		if _, ok := r.tools[name]; !ok {
			r.order = append(r.order, name)
		}
		r.tools[name] = t
	}
	r.schemas = make([]core.ToolSchema, len(r.order))
	for i, name := range r.order {
		t := r.tools[name]
		r.schemas[i] = core.ToolSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		}
	}
	return r
}

func (r *toolRegistry) ToolSchemas() []core.ToolSchema {
	sort.Slice(r.schemas, func(i, j int) bool {
		return r.schemas[i].Name < r.schemas[j].Name
	})
	return r.schemas
}

func (r *toolRegistry) Name() string {
	return fmt.Sprintf("TestEnv: %d tools", len(r.tools))
}

func (r *toolRegistry) RunCalls(ctx context.Context, calls []core.ToolCall) ([]core.ToolResultMsg, error) {
	out := make([]core.ToolResultMsg, len(calls))
	for i, tc := range calls {
		t, ok := r.tools[tc.Name]
		if !ok {
			out[i] = core.ToolResultMsg{
				CallID:  tc.ID,
				Name:    tc.Name,
				Content: fmt.Sprintf("unknown tool %q", tc.Name),
			}
			continue
		}
		result, err := t.Execute(ctx, json.RawMessage(tc.Arguments))
		if err != nil {
			out[i] = core.ToolResultMsg{
				CallID:  tc.ID,
				Name:    tc.Name,
				Content: fmt.Sprintf("error: %v", err),
			}
			continue
		}
		out[i] = core.ToolResultMsg{CallID: tc.ID, Name: tc.Name, Content: result}
	}
	return out, nil
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(data)
}

// ============================================================================
// Test tools
// ============================================================================

type echoTool struct{}

func (e echoTool) Name() string        { return "echo" }
func (e echoTool) Description() string { return "回显输入文本，不做任何修改" }
func (e echoTool) ReadOnly() bool      { return true }
func (e echoTool) Schema() json.RawMessage {
	return mustMarshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{"type": "string", "description": "要回显的文本"},
		},
		"required": []string{"text"},
	})
}
func (e echoTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct{ Text string }
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("echo: %w", err)
	}
	return p.Text, nil
}

type addTool struct{}

func (a addTool) Name() string        { return "add" }
func (a addTool) Description() string { return "计算两个整数之和" }
func (a addTool) ReadOnly() bool      { return true }
func (a addTool) Schema() json.RawMessage {
	return mustMarshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": map[string]any{"type": "integer", "description": "第一个加数"},
			"b": map[string]any{"type": "integer", "description": "第二个加数"},
		},
		"required": []string{"a", "b"},
	})
}
func (a addTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("add: %w", err)
	}
	return fmt.Sprintf("%d + %d = %d", p.A, p.B, p.A+p.B), nil
}

// ============================================================================
// Streaming output state + formatting
// ============================================================================
