package transcript

import (
	"strings"
	"testing"

	"mini_agent/core"
	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
)

func TestTranscriptAnimationStopsWhenIdle(t *testing.T) {
	tr := NewTranscript(nil)
	tr.SetWidth(80)
	cmd := tr.SetTurns([]BlockTurnUpdate{
		{Idx: 0, Turn: core.Turn{
			core.TextMsg{RoleName: "user", Content: "hello"},
			core.AssistantMsg{Content: "done"},
		}},
	})
	if cmd != nil {
		t.Fatal("completed block should not start animation")
	}

	_, cmd = tr.Update(common.AnimationTickMsg{})
	if cmd != nil {
		t.Fatal("completed block should not continue animation")
	}
}

func TestTranscriptStartsAnimationFromBlockUpdate(t *testing.T) {
	tr := NewTranscript(nil)
	tr.SetWidth(80)

	cmd := tr.SetTurns([]BlockTurnUpdate{
		{Idx: 0, Turn: core.Turn{core.TextMsg{RoleName: "user", Content: "hello"}}},
	})

	if cmd != nil {
		t.Fatal("transcript should not schedule animation ticks directly")
	}
}

func TestTranscriptSchedulesAnimationFromReasoningEvent(t *testing.T) {
	tr := NewTranscript(nil)
	tr.SetWidth(80)
	tr.Update(core.ConversationOutput{SyncPrimitives: core.SyncPrimitiveNewTurn{}})
	tr.Update(core.ConversationOutput{
		SyncPrimitives: core.SyncPrimitiveAppendMessages{
			Messages: []core.Message{core.TextMsg{RoleName: "user", Content: "hello"}},
		},
	})

	tr.Update(core.ConversationOutput{
		SyncPrimitives: core.SyncPrimitiveStartResponding{},
		AfterEvent:     core.KeyNotifyReasoningStart{},
	})
	tr.Update(core.ConversationOutput{
		SyncPrimitives: core.SyncPrimitiveDeltaChunk{
			Chunk: core.Chunk{Reasoning: "thinking"},
		},
	})

	_, _ = tr.Update(common.AnimationTickMsg{})
}

func TestTranscriptCopyNoticeWrapsTranscript(t *testing.T) {
	oldWriter := common.SetClipboardWriter(func(string) error { return nil })
	defer func() { common.SetClipboardWriter(oldWriter) }()

	tr := NewTranscript(nil)
	tr.Update(tea.WindowSizeMsg{Width: 40, Height: 5})
	tr.SetTurns([]BlockTurnUpdate{
		{Idx: 0, Turn: core.Turn{core.TextMsg{RoleName: "user", Content: "hello world"}}},
	})

	overlay := &common.SelectionOverlay{Inner: tr, NoticeText: "输出已复制"}
	overlay.Sel = common.Selection{
		Active: true,
		Anchor: common.SelectPos{Line: 0, Col: 0},
		Head:   common.SelectPos{Line: 0, Col: 5},
	}

	_, cmd := overlay.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("copy release should return clipboard command")
	}
	rendered := overlay.Render()
	if !strings.Contains(rendered, "输出已复制") {
		t.Fatalf("copy notice missing from render: %q", rendered)
	}
}

func TestTranscriptUsesMirrorChangedIndexes(t *testing.T) {
	tr := NewTranscript(core.CloneTurns([]core.Turn{
		{core.TextMsg{RoleName: "user", Content: "first"}},
		{core.TextMsg{RoleName: "user", Content: "second"}},
	}))
	tr.Update(tea.WindowSizeMsg{Width: 80, Height: 5})
	tr.RecomputeContent()

	if tr.IsBlockDirty(0) || tr.IsBlockDirty(1) {
		t.Fatal("blocks should not be dirty after recompute")
	}
	tr.Update(core.ConversationOutput{
		SyncPrimitives: core.SyncPrimitiveAppendMessages{
			Messages: []core.Message{core.TextMsg{RoleName: "system", Content: "patched"}},
		},
	})

	rendered := tr.Render()
	if rendered == "" {
		t.Fatal("render should not be empty after sync")
	}
}
