package agent_interact

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"mini_agent/agent/intra_turn"
	"mini_agent/core"
	"mini_agent/providers/openai"

	tea "charm.land/bubbletea/v2"
)

const (
	benchmarkTurnCopies = 1000
	benchmarkWidth      = 120
	benchmarkHeight     = 40
	benchmarkReasoning  = "Reasoning fixture:\n1. Identify that the user requested two tool calls in sequence.\n2. Select two available tools and prepare simple arguments for each call.\n3. Keep the tool-call order stable so the transcript renders input, reasoning, tool calls, tool results, and final answer.\n4. Produce a short final answer after both tool results are available."
)

var (
	benchViewSink  tea.View
	benchCmdSink   tea.Cmd
	benchModelSink tea.Model
)

type benchmarkTurnFixture struct {
	Source      string                    `json:"source"`
	Prompt      string                    `json:"prompt"`
	GeneratedAt string                    `json:"generated_at"`
	Messages    []benchmarkFixtureMessage `json:"messages"`
}

type benchmarkFixtureMessage struct {
	Type            string          `json:"type"`
	RoleName        string          `json:"role_name,omitempty"`
	Content         string          `json:"content,omitempty"`
	Reasoning       string          `json:"reasoning,omitempty"`
	ReasoningActive bool            `json:"reasoning_active,omitempty"`
	Signature       string          `json:"signature,omitempty"`
	ToolCalls       []core.ToolCall `json:"tool_calls,omitempty"`
	CallID          string          `json:"call_id,omitempty"`
	Name            string          `json:"name,omitempty"`
}

func TestCaptureBenchmarkTurnFromAPI(t *testing.T) {
	if os.Getenv("AGENT_INTERACT_CAPTURE_FIXTURE") != "1" {
		t.Skip("set AGENT_INTERACT_CAPTURE_FIXTURE=1 to refresh testdata/agent_interact_turn_fixture.json from the API")
	}
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Fatal("DEEPSEEK_API_KEY not set")
	}

	model := envOr("DEEPSEEK_MODEL", "deepseek-v4-pro")
	effort := envOr("DEEPSEEK_EFFORT", "high")
	provider, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  model,
		Effort: effort,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	history := []core.Turn{
		{core.TextMsg{RoleName: "system", Content: "你是工具调用测试助手。用户要求调用工具时必须实际调用工具，最后只做一句简短总结。"}},
		{core.TextMsg{RoleName: "user", Content: "请按顺序调用两个工具"}},
	}
	events := intra_turn.RunTurn(ctx, provider, benchmarkToolRegistry{echoTool{}, addTool{}}, history, "")
	for event := range events {
		if failure, ok := event.BeforeEvent.(core.KeyNotifyFailure); ok && failure.Err != nil {
			t.Fatalf("agent turn failed: %v", failure.Err)
		}
	}

	turn := history[len(history)-1]
	if got := countToolCalls(turn); got < 2 {
		t.Fatalf("captured turn has %d tool call(s), want at least 2", got)
	}
	if got := countToolResults(turn); got < 2 {
		t.Fatalf("captured turn has %d tool result(s), want at least 2", got)
	}

	fixture := benchmarkTurnFixture{
		Source:      fmt.Sprintf("API model=%s effort=%s", model, effort),
		Prompt:      "请按顺序调用两个工具",
		GeneratedAt: time.Now().Format(time.RFC3339),
		Messages:    encodeBenchmarkTurn(turn),
	}
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "agent_interact_turn_fixture.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", path)
}

func BenchmarkReadWriteModel1000Turns(b *testing.B) {
	b.Run("ScrollWheel_Update", func(b *testing.B) {
		m := newBenchmarkReadWriteModel(b)
		msgs := []tea.MouseWheelMsg{
			{Button: tea.MouseWheelDown},
			{Button: tea.MouseWheelUp},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model, cmd := m.Update(msgs[i%len(msgs)])
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
		}
		benchModelSink = m
	})

	b.Run("ScrollWheel_UpdateAndRender", func(b *testing.B) {
		m := newBenchmarkReadWriteModel(b)
		msgs := []tea.MouseWheelMsg{
			{Button: tea.MouseWheelDown},
			{Button: tea.MouseWheelUp},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model, cmd := m.Update(msgs[i%len(msgs)])
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
			benchViewSink = m.View()
		}
		benchModelSink = m
	})

	b.Run("ClickFold_Update", func(b *testing.B) {
		m := newBenchmarkReadWriteModel(b)
		click, release := firstFoldClickMessages(b, m)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model, cmd := m.Update(click)
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
			model, cmd = m.Update(release)
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
		}
		benchModelSink = m
	})

	b.Run("ClickFold_UpdateAndRender", func(b *testing.B) {
		m := newBenchmarkReadWriteModel(b)
		click, release := firstFoldClickMessages(b, m)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model, cmd := m.Update(click)
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
			model, cmd = m.Update(release)
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
			benchViewSink = m.View()
		}
		benchModelSink = m
	})

	b.Run("Resize_Update", func(b *testing.B) {
		m := newBenchmarkReadWriteModel(b)
		sizes := []tea.WindowSizeMsg{
			{Width: benchmarkWidth - 20, Height: benchmarkHeight},
			{Width: benchmarkWidth, Height: benchmarkHeight + 8},
			{Width: benchmarkWidth, Height: benchmarkHeight},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model, cmd := m.Update(sizes[i%len(sizes)])
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
		}
		benchModelSink = m
	})

	b.Run("Resize_UpdateAndRender", func(b *testing.B) {
		m := newBenchmarkReadWriteModel(b)
		sizes := []tea.WindowSizeMsg{
			{Width: benchmarkWidth - 20, Height: benchmarkHeight},
			{Width: benchmarkWidth, Height: benchmarkHeight + 8},
			{Width: benchmarkWidth, Height: benchmarkHeight},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model, cmd := m.Update(sizes[i%len(sizes)])
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
			benchViewSink = m.View()
		}
		benchModelSink = m
	})

	b.Run("NewTurnEvent_Update", func(b *testing.B) {
		m := newBenchmarkReadWriteModel(b)
		event := core.ConversationOutput{
			SyncPrimitives: core.SyncPrimitiveNewTurn{},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i > 0 && i%100 == 0 {
				b.StopTimer()
				m = newBenchmarkReadWriteModel(b)
				b.StartTimer()
			}
			model, cmd := m.Update(event)
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
		}
		benchModelSink = m
	})

	b.Run("StreamDeltaEvent_Update", func(b *testing.B) {
		m := newBenchmarkReadWriteModel(b)
		m.Update(core.ConversationOutput{SyncPrimitives: core.SyncPrimitiveNewTurn{}})
		m.Update(core.ConversationOutput{SyncPrimitives: core.SyncPrimitiveStartResponding{}})
		event := core.ConversationOutput{
			SyncPrimitives: core.SyncPrimitiveDeltaChunk{
				Chunk: core.Chunk{Text: "新增流式输出"},
			},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model, cmd := m.Update(event)
			m = model.(*ReadWriteModel)
			benchCmdSink = cmd
		}
		benchModelSink = m
	})
}

func newBenchmarkReadWriteModel(tb testing.TB) *ReadWriteModel {
	tb.Helper()
	wrapper, _ := NewReadWriteModel(nil)
	m := wrapper.Inner
	history := benchmarkHistory(tb)
	actions := make([]core.TurnModifyOrAppendAction, len(history))
	for i, t := range history {
		actions[i] = core.TurnModifyOrAppendAction{Index: i, Data: t}
	}
	m.Update(core.ConversationOutput{
		SyncPrimitives: core.SyncPrimitiveModifyOrAppendTurns{Actions: actions},
	})
	model, cmd := m.Update(tea.WindowSizeMsg{Width: benchmarkWidth, Height: benchmarkHeight})
	benchCmdSink = cmd
	m = model.(*ReadWriteModel)
	return m
}

func benchmarkHistory(tb testing.TB) []core.Turn {
	tb.Helper()
	turn := readBenchmarkTurn(tb)
	history := make([]core.Turn, benchmarkTurnCopies)
	for i := range history {
		history[i] = cloneTurn(turn)
	}
	return history
}

func firstFoldClickMessages(tb testing.TB, m *ReadWriteModel) (tea.MouseClickMsg, tea.MouseReleaseMsg) {
	tb.Helper()
	for scrolls := 0; scrolls < 2000; scrolls++ {
		rendered := m.overlay.Render()
		for y, line := range strings.Split(rendered, "\n") {
			if strings.Contains(line, "Tool:") || strings.Contains(line, "▸ echo") || strings.Contains(line, "▸ add") {
				return tea.MouseClickMsg{Button: tea.MouseLeft, X: 2, Y: y},
					tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 2, Y: y}
			}
		}
		_, _ = m.overlay.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	}
	tb.Fatalf("no visible tool section found")
	return tea.MouseClickMsg{}, tea.MouseReleaseMsg{}
}

func readBenchmarkTurn(tb testing.TB) core.Turn {
	tb.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "agent_interact_turn_fixture.json"))
	if err != nil {
		tb.Fatalf("read benchmark fixture: %v", err)
	}
	var fixture benchmarkTurnFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		tb.Fatalf("decode benchmark fixture: %v", err)
	}
	turn, err := decodeBenchmarkTurn(fixture.Messages)
	if err != nil {
		tb.Fatalf("decode benchmark turn: %v", err)
	}
	return turn
}

func encodeBenchmarkTurn(turn core.Turn) []benchmarkFixtureMessage {
	out := make([]benchmarkFixtureMessage, 0, len(turn))
	for _, msg := range turn {
		switch m := msg.(type) {
		case core.TextMsg:
			out = append(out, benchmarkFixtureMessage{
				Type:     "text",
				RoleName: m.RoleName,
				Content:  m.Content,
			})
		case core.AssistantMsg:
			out = append(out, benchmarkFixtureMessage{
				Type:            "assistant",
				Content:         m.Content,
				Reasoning:       redactedBenchmarkReasoning(m.Reasoning),
				ReasoningActive: m.ReasoningActive,
				Signature:       m.ReasoningSignature,
				ToolCalls:       append([]core.ToolCall(nil), m.ToolCalls...),
			})
		case core.ToolResultMsg:
			out = append(out, benchmarkFixtureMessage{
				Type:    "tool_result",
				CallID:  m.CallID,
				Name:    m.Name,
				Content: m.Content,
			})
		}
	}
	return out
}

func redactedBenchmarkReasoning(reasoning string) string {
	if strings.TrimSpace(reasoning) == "" {
		return ""
	}
	return benchmarkReasoning
}

func decodeBenchmarkTurn(messages []benchmarkFixtureMessage) (core.Turn, error) {
	turn := make(core.Turn, 0, len(messages))
	for i, msg := range messages {
		switch msg.Type {
		case "text":
			turn = append(turn, core.TextMsg{RoleName: msg.RoleName, Content: msg.Content})
		case "assistant":
			turn = append(turn, core.AssistantMsg{
				Content:            msg.Content,
				Reasoning:          msg.Reasoning,
				ReasoningActive:    msg.ReasoningActive,
				ReasoningSignature: msg.Signature,
				ToolCalls:          append([]core.ToolCall(nil), msg.ToolCalls...),
			})
		case "tool_result":
			turn = append(turn, core.ToolResultMsg{CallID: msg.CallID, Name: msg.Name, Content: msg.Content})
		default:
			return nil, fmt.Errorf("message[%d]: unknown type %q", i, msg.Type)
		}
	}
	return turn, nil
}

func cloneTurn(turn core.Turn) core.Turn {
	out := make(core.Turn, len(turn))
	for i, msg := range turn {
		out[i] = msg.Clone()
	}
	return out
}

func countToolCalls(turn core.Turn) int {
	count := 0
	for _, msg := range turn {
		if assistant, ok := msg.(core.AssistantMsg); ok {
			count += len(assistant.ToolCalls)
		}
	}
	return count
}

func countToolResults(turn core.Turn) int {
	count := 0
	for _, msg := range turn {
		if _, ok := msg.(core.ToolResultMsg); ok {
			count++
		}
	}
	return count
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type benchmarkToolRegistry []core.Tool

func (r benchmarkToolRegistry) ToolSchemas() []core.ToolSchema {
	schemas := make([]core.ToolSchema, 0, len(r))
	for _, tool := range r {
		schemas = append(schemas, core.ToolSchema{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Schema(),
		})
	}
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})
	return schemas
}

func (r benchmarkToolRegistry) Name() string {
	return fmt.Sprintf("TestEnv: %d tools", len(r))
}

func (r benchmarkToolRegistry) RunCalls(ctx context.Context, calls []core.ToolCall) ([]core.ToolResultMsg, error) {
	tools := make(map[string]core.Tool, len(r))
	for _, tool := range r {
		tools[tool.Name()] = tool
	}
	results := make([]core.ToolResultMsg, len(calls))
	for i, call := range calls {
		tool, ok := tools[call.Name]
		if !ok {
			results[i] = core.ToolResultMsg{CallID: call.ID, Name: call.Name, Content: fmt.Sprintf("unknown tool %q", call.Name)}
			continue
		}
		result, err := tool.Execute(ctx, json.RawMessage(call.Arguments))
		if err != nil {
			results[i] = core.ToolResultMsg{CallID: call.ID, Name: call.Name, Content: fmt.Sprintf("error: %v", err)}
			continue
		}
		results[i] = core.ToolResultMsg{CallID: call.ID, Name: call.Name, Content: result}
	}
	return results, nil
}

type echoTool struct{}

func (echoTool) Name() string        { return "echo" }
func (echoTool) Description() string { return "回显输入文本，不做任何修改" }
func (echoTool) ReadOnly() bool      { return true }
func (echoTool) Schema() json.RawMessage {
	return mustMarshalBenchmarkJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{"type": "string", "description": "要回显的文本"},
		},
		"required": []string{"text"},
	})
}
func (echoTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	return params.Text, nil
}

type addTool struct{}

func (addTool) Name() string        { return "add" }
func (addTool) Description() string { return "计算两个整数之和" }
func (addTool) ReadOnly() bool      { return true }
func (addTool) Schema() json.RawMessage {
	return mustMarshalBenchmarkJSON(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": map[string]any{"type": "integer", "description": "第一个加数"},
			"b": map[string]any{"type": "integer", "description": "第二个加数"},
		},
		"required": []string{"a", "b"},
	})
}
func (addTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var params struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d + %d = %d", params.A, params.B, params.A+params.B), nil
}

func mustMarshalBenchmarkJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
