package block

import (
	"strings"
	"testing"
	"time"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func testSectionFolded(block *Block, index int) bool {
	if index < 0 || index >= len(block.SectionVisualStates) {
		return false
	}
	return sectionFolded(block.SectionVisualStates[index])
}

func testReasoningState(block *Block, index int) *ReasoningVisualState {
	if index < 0 || index >= len(block.SectionVisualStates) {
		return nil
	}
	state, _ := block.SectionVisualStates[index].(*ReasoningVisualState)
	return state
}

func TestBlockRenderUsesNaturalHeight(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.TextMsg{RoleName: "user", Content: "hello"},
		core.AssistantMsg{Reasoning: "thinking", Content: "answer"},
	}})

	got := block.Render()
	for _, want := range []string{"hello", "Reasoning:", "thinking", "answer"} {
		if !strings.Contains(got, want) {
			t.Fatalf("render missing %q: %q", want, got)
		}
	}
}

func TestBlockClickTogglesInputFromLocalWidget(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.TextMsg{RoleName: "user", Content: "hello\nagain"},
	}})

	changed, cmd := block.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 1, Y: 1})
	if cmd != nil {
		t.Fatal("expected no command")
	}
	if !changed {
		t.Fatal("expected local widget click to change block")
	}
	if !testSectionFolded(block, 0) {
		t.Fatal("expected input section to fold")
	}
}

func TestBlockDoesNotFoldSingleLineInput(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.TextMsg{RoleName: "user", Content: "hello"},
	}})

	changed, _ := block.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 1, Y: 0})
	if changed {
		t.Fatal("single-line input should not fold")
	}
	if testSectionFolded(block, 0) {
		t.Fatal("single-line input should remain unfolded")
	}
}

func TestBlockRepeatedReasoningSectionsFoldIndependently(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.AssistantMsg{Reasoning: "first reasoning"},
		core.AssistantMsg{Reasoning: "second reasoning"},
	}})

	changed, _ := block.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 2, Y: 0})
	if !changed {
		t.Fatal("expected first reasoning click to change block")
	}
	if !testSectionFolded(block, 0) {
		t.Fatal("first reasoning should be folded")
	}
	if testSectionFolded(block, 1) {
		t.Fatal("second reasoning should remain unfolded")
	}
}

func TestBlockFoldedReasoningShowsOnlyBanner(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.AssistantMsg{Reasoning: "thinking\nmore"},
	}})

	changed, _ := block.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 2, Y: 0})
	if !changed {
		t.Fatal("expected reasoning fold")
	}

	got := block.Render()
	if strings.Contains(got, "thinking") || strings.Contains(got, "more") {
		t.Fatalf("folded reasoning should only show banner, got %q", got)
	}
	if !strings.Contains(got, "Reasoning:") || !strings.Contains(got, "▸") {
		t.Fatalf("folded reasoning missing banner marker: %q", got)
	}
}

func TestBlockToolCallDefaultsFoldedAndExpandsWithResult(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.AssistantMsg{ToolCalls: []core.ToolCall{{
			ID:        "call-1",
			Name:      "read",
			Arguments: `{"path":"README.md"}`,
		}}},
		core.ToolResultMsg{CallID: "call-1", Name: "read", Content: "file content"},
	}})

	got := block.Render()
	if !strings.Contains(got, "read") || !strings.Contains(got, "call-1") || !strings.Contains(got, "已完成") {
		t.Fatalf("folded tool summary missing name/id/status: %q", got)
	}
	if strings.Contains(got, "Arguments:") || strings.Contains(got, "file content") {
		t.Fatalf("folded tool should hide details: %q", got)
	}

	changed, cmd := block.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft, X: 2, Y: 0})
	if cmd != nil {
		t.Fatal("expected no command")
	}
	if !changed {
		t.Fatal("expected tool click to expand section")
	}

	got = block.Render()
	for _, want := range []string{"Arguments:", `{"path":"README.md"}`, "Result:", "file content"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expanded tool missing %q: %q", want, got)
		}
	}
}

func TestBlockToolCallShowsBeforeResult(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.AssistantMsg{ToolCalls: []core.ToolCall{{
			ID:        "call-1",
			Name:      "read",
			Arguments: `{"path":"README.md"}`,
		}}},
	}})

	got := block.Render()
	for _, want := range []string{"read", "call-1", "进行中"} {
		if !strings.Contains(got, want) {
			t.Fatalf("in-flight tool summary missing %q: %q", want, got)
		}
	}
	if strings.Contains(got, "Arguments:") {
		t.Fatalf("in-flight tool should default folded: %q", got)
	}
}

func TestBlockRemovesSpacerBetweenAdjacentToolSections(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.AssistantMsg{ToolCalls: []core.ToolCall{
			{ID: "call-add", Name: "add", Arguments: `{"a":1,"b":2}`},
			{ID: "call-echo", Name: "echo", Arguments: `{"text":"ok"}`},
		}},
		core.ToolResultMsg{CallID: "call-add", Name: "add", Content: "3"},
		core.ToolResultMsg{CallID: "call-echo", Name: "echo", Content: "ok"},
	}})

	got := ansi.Strip(block.Render())
	if strings.Contains(got, "已完成\n\n  ▸ echo") {
		t.Fatalf("adjacent tool sections should not have a blank line: %q", got)
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 2 ||
		!strings.Contains(lines[0], "▸ add call-add · 已完成") ||
		!strings.Contains(lines[1], "▸ echo call-echo · 已完成") {
		t.Fatalf("adjacent tool summaries should remain consecutive lines: %q", got)
	}
}

func TestBlockRendersTransparentMetadataAtBottom(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.TextMsg{RoleName: "user", Content: "hello"},
		core.TrasnparentTextMsg{RoleName: "ProviderName", Content: "deepseek-v4-pro (high)"},
		core.AssistantMsg{Content: "answer"},
		core.TrasnparentTextMsg{RoleName: "EnvName", Content: "build"},
	}})

	got := strings.TrimSpace(ansi.Strip(block.Render()))
	want := "  ▣  DeepSeek V4 Pro (High) · Build"
	if !strings.HasSuffix(got, want) {
		t.Fatalf("metadata should render at bottom as %q, got %q", want, got)
	}
}

func TestBlockRenderIndentsNonInputSections(t *testing.T) {
	block := NewBlock(80)
	block.Update(TurnDataMsg{Turn: core.Turn{
		core.TextMsg{RoleName: "user", Content: "hello\nagain"},
		core.AssistantMsg{Content: "answer\nmore"},
	}})

	got := block.Render()
	lines := strings.Split(got, "\n")
	if strings.HasPrefix(ansi.Strip(lines[0]), "  ") {
		t.Fatalf("input section should not be indented: %q", got)
	}

	foundAnswer := false
	for _, line := range lines {
		stripped := ansi.Strip(line)
		if strings.Contains(stripped, "answer") {
			foundAnswer = true
			if !strings.HasPrefix(stripped, "  ") {
				t.Fatalf("answer section should be indented by relocator: %q", got)
			}
		}
	}
	if !foundAnswer {
		t.Fatalf("answer line not found: %q", got)
	}
}

func TestBlockReasoningAnimationOnlyWhileReasoning(t *testing.T) {
	block := NewBlock(80)

	changed, cmd := block.Update(common.AnimationTickMsg(time.Now()))
	if changed || cmd != nil {
		t.Fatal("idle block should not tick or schedule")
	}

	changed, cmd = block.Update(TurnDataMsg{Turn: core.Turn{core.AssistantMsg{Reasoning: "thinking"}}})
	if !changed || cmd != nil {
		t.Fatal("reasoning update should change without command")
	}
	changed, cmd = block.SetPhase(core.TurnPhaseReasoning)
	if !changed || cmd != nil {
		t.Fatal("reasoning phase should change without command")
	}

	changed, cmd = block.Update(common.AnimationTickMsg(time.Now()))
	if !changed || cmd != nil {
		t.Fatal("active reasoning section should tick without command")
	}
	state := testReasoningState(block, 0)
	if state == nil {
		t.Fatal("expected reasoning visual state")
	}
	if state.Spinner != 1 {
		t.Fatalf("spinner = %d, want 1", state.Spinner)
	}

	changed, cmd = block.SetComplete()
	if !changed || cmd != nil {
		t.Fatal("completion should change without command")
	}
	endedAt := state.EndedAt
	if endedAt.IsZero() {
		t.Fatal("completion should set reasoning EndedAt")
	}

	changed, cmd = block.Update(common.AnimationTickMsg(time.Now()))
	if changed || cmd != nil {
		t.Fatal("completed block should not tick")
	}
	if state.Spinner != 1 {
		t.Fatalf("completed spinner changed to %d", state.Spinner)
	}
	if !state.EndedAt.Equal(endedAt) {
		t.Fatal("completed reasoning EndedAt should remain stable")
	}
}
