// single_turn demonstrates a single-turn agent: two test tools (echo, add) and
// a three-turn conversation driving the deepseek provider through the core
// abstractions.
//
// Run: cd mini_agent && DEEPSEEK_API_KEY=$DEEPSEEK_API_KEY go run ./cmd/single_turn
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"mini_agent/agent/intra_turn"
	"mini_agent/core"
	"mini_agent/providers/openai"
)

// ============================================================================
// Tool registry — implements core.ToolSetAndRunner
// ============================================================================

// toolRegistry bundles tools and dispatches ToolCalls to them.
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

// ============================================================================
// Test tools
// ============================================================================

// echoTool echoes input text unchanged.
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

// addTool computes the sum of two integers.
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
// Consume stream
// ============================================================================

// outputState tracks what kind of streaming content we are currently in the
// middle of printing.  When idle, we insert section separators before starting
// new content.  When streaming (reasoning or text), we print content directly
// without any prefix so the user sees a continuous flow.
type outputState int

const (
	stateIdle      outputState = iota // nothing streaming
	stateReasoning                    // printing reasoning deltas inline
	stateContent                      // printing text deltas inline
)

// changeStreaming transitions from the current streaming state to newState.
// When leaving reasoning it prints [思考结束]; when entering reasoning it
// prints [思考].  Passing stateIdle as newState just ends the current stream
// without starting a new one.
func changeStreaming(current *outputState, newState outputState) {
	if *current == newState {
		return
	}
	if *current == stateReasoning {
		// fmt.Print("[思考结束]")
		fmt.Println()
	}
	if *current == stateContent {
		// fmt.Print("[回复结束]")
		fmt.Println()
	}
	if newState == stateReasoning {
		// fmt.Print("[思考开始] ")
	}
	if newState == stateContent {
		// fmt.Print("[回复开始] ")
	}
	*current = newState
}

// section prints a uniformly-formatted section header.
func section(title string) {
	fmt.Printf("─── %s ───\n", title)
}

func consumeTurn(stream core.OutStream[core.ConversationOutput]) {
	var state outputState
	for event := range stream {
		if event.BeforeEvent != nil {
			if state == stateReasoning {
				fmt.Println()
			}
			fmt.Println(event.BeforeEvent.String())
			state = stateIdle
		}
		printAfterEvent := func() {
			if event.AfterEvent != nil {
				if state == stateReasoning {
					fmt.Println()
				}
				fmt.Println(event.AfterEvent.String())
				state = stateIdle
			}
		}

		sync := event.SyncPrimitives
		// === 判断流式输出（行内追加，字符级）============================
		switch e := sync.(type) {
		case core.SyncPrimitiveDeltaChunk:
			c := e.Chunk
			if c.Reasoning != "" {
				if state != stateReasoning {
					changeStreaming(&state, stateReasoning)
				}
				fmt.Print(c.Reasoning)
				printAfterEvent()
				continue
			}
			if c.Text != "" {
				if state != stateContent {
					changeStreaming(&state, stateContent)
				}
				fmt.Print(c.Text)
				printAfterEvent()
				continue
			}
		}

		changeStreaming(&state, stateIdle)

		// === 非流式输出（完整行）==================================
		switch e := sync.(type) {
		case core.SyncPrimitiveStartResponding:
		case core.SyncPrimitiveDeltaChunk:
			c := e.Chunk
			if c.ToolCall != nil {
				fmt.Printf("[TOOL_CALL] %s(%s)\n", c.ToolCall.Name, c.ToolCall.Arguments)
			}
			if c.Usage != nil {
				fmt.Printf("[USAGE] prompt=%d, completion=%d, total=%d\n",
					c.Usage.PromptTokens, c.Usage.CompletionTokens, c.Usage.TotalTokens)
			}

		case core.SyncPrimitiveAppendMessages:
			section("工具调用结果")
			for _, msg := range e.Messages {
				fmt.Printf("  %s\n", msg.String())
			}
		}
		printAfterEvent()
	}
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(data)
}

// ============================================================================
// main
// ============================================================================

func main() {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "DEEPSEEK_API_KEY not set")
		os.Exit(1)
	}

	ctx := context.Background()
	runner := newToolRegistry(echoTool{}, addTool{})

	// === Test 1: deepseek-chat (no reasoning) ===
	{
		fmt.Println(strings.Repeat("#", 60))
		fmt.Println("模型: deepseek-chat (无 thinking)")
		fmt.Println(strings.Repeat("#", 60))

		// anthropic.Config{
		// 	APIKey: apiKey,
		// 	Model:  "deepseek-v4-flash",
		// }

		p, err := openai.New(openai.Config{APIKey: apiKey, Model: "deepseek-chat"})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		history := []core.Turn{
			{core.TextMsg{RoleName: "system", Content: "你是一个精准的助手。当用户要求使用工具时，必须调用对应的工具。"}},
		}

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("回合 0: 询问模型的名字")
		fmt.Println(strings.Repeat("=", 60))
		history = append(history, core.Turn{core.TextMsg{RoleName: "user", Content: `你是谁`}})
		consumeTurn(intra_turn.RunTurn(ctx, p, runner, history, intra_turn.DefaultInterruptedToolResult))

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("回合 1: 同时调用 echo 和 add")
		fmt.Println(strings.Repeat("=", 60))
		history = append(history, core.Turn{core.TextMsg{RoleName: "user", Content: `请同时使用两个工具：(1) 用 echo 回显 "hello world"，(2) 用 add 计算 3+5。`}})
		consumeTurn(intra_turn.RunTurn(ctx, p, runner, history, intra_turn.DefaultInterruptedToolResult))

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("回合 2: 单独调用 echo")
		fmt.Println(strings.Repeat("=", 60))
		history = append(history, core.Turn{core.TextMsg{RoleName: "user", Content: "请用 echo 工具重复这段话：DeepSeek is really great!"}})
		consumeTurn(intra_turn.RunTurn(ctx, p, runner, history, intra_turn.DefaultInterruptedToolResult))

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("回合 3: 单独调用 add")
		fmt.Println(strings.Repeat("=", 60))
		history = append(history, core.Turn{core.TextMsg{RoleName: "user", Content: "请用 add 工具计算 99+1。"}})
		consumeTurn(intra_turn.RunTurn(ctx, p, runner, history, intra_turn.DefaultInterruptedToolResult))
	}

	// === Test 2: deepseek-reasoner with thinking=high ===
	{
		fmt.Println(strings.Repeat("#", 60))
		fmt.Println("模型: deepseek-reasoner (thinking=high)")
		fmt.Println(strings.Repeat("#", 60))

		// anthropic.Config{
		// 	APIKey: apiKey,
		// 	Model:  "deepseek-v4-pro",
		// 	Effort: "high",
		// }

		p, err := openai.New(openai.Config{APIKey: apiKey, Model: "deepseek-reasoner", Effort: "high"})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		history := []core.Turn{
			{core.TextMsg{RoleName: "system", Content: "你是一个精准的助手。当用户要求使用工具时，必须调用对应的工具。"}},
		}

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("回合 0: 询问模型的名字")
		fmt.Println(strings.Repeat("=", 60))
		history = append(history, core.Turn{core.TextMsg{RoleName: "user", Content: `你是谁`}})
		consumeTurn(intra_turn.RunTurn(ctx, p, runner, history, intra_turn.DefaultInterruptedToolResult))

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("回合 1: 同时调用 echo 和 add")
		fmt.Println(strings.Repeat("=", 60))
		history = append(history, core.Turn{core.TextMsg{RoleName: "user", Content: `请同时使用两个工具：(1) 用 echo 回显 "hello world"，(2) 用 add 计算 3+5。`}})
		consumeTurn(intra_turn.RunTurn(ctx, p, runner, history, intra_turn.DefaultInterruptedToolResult))

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("回合 2: 单独调用 echo")
		fmt.Println(strings.Repeat("=", 60))
		history = append(history, core.Turn{core.TextMsg{RoleName: "user", Content: "请用 echo 工具重复这段话：DeepSeek is really great!"}})
		consumeTurn(intra_turn.RunTurn(ctx, p, runner, history, intra_turn.DefaultInterruptedToolResult))

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("回合 3: 单独调用 add")
		fmt.Println(strings.Repeat("=", 60))
		history = append(history, core.Turn{core.TextMsg{RoleName: "user", Content: "请用 add 工具计算 99+1。"}})
		consumeTurn(intra_turn.RunTurn(ctx, p, runner, history, intra_turn.DefaultInterruptedToolResult))
	}
}
