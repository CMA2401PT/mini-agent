package lazy_scroll_view

import (
	"strings"
	"testing"
	"time"

	"mini_agent/ui/tui/common"

	tea "charm.land/bubbletea/v2"
)

type mockBlock struct {
	content     string
	renders     int
	width       int
	animating   bool
	phaseResult bool
	updated     int
}

func (m *mockBlock) SetWidth(width int) {
	m.width = width
}

func (m *mockBlock) SetComplete() (bool, tea.Cmd) {
	return false, nil
}

func (m *mockBlock) SetPhase(phase any) (bool, tea.Cmd) {
	return m.phaseResult, nil
}

func (m *mockBlock) Render() string {
	m.renders++
	return m.content
}

func (m *mockBlock) Update(msg tea.Msg) (bool, tea.Cmd) {
	m.updated++
	return false, nil
}

func (m *mockBlock) IsAnimating() bool {
	return m.animating
}

func newMockBlock(width int) *mockBlock {
	return &mockBlock{width: width}
}

func TestLocalBlockY(t *testing.T) {
	a := &mockBlock{content: "first"}
	b := &mockBlock{content: "second"}
	lv := NewLazyScrollView(func(width int) *mockBlock { return &mockBlock{} })
	lv.blocks = []BlockMeta[*mockBlock]{{Block: a}, {Block: b}}
	lv.RecomputeContent()

	firstHeight := strings.Count(a.Render(), "\n") + 1
	if got := lv.localBlockY(0); got != 0 {
		t.Fatalf("localBlockY first line = %d, want 0", got)
	}
	if got := lv.localBlockY(firstHeight); got != -1 {
		t.Fatalf("localBlockY separator = %d, want -1", got)
	}
	if got := lv.localBlockY(firstHeight + 1); got != 0 {
		t.Fatalf("localBlockY second block first line = %d, want 0", got)
	}
}

func TestSendBlockMsgForwardsClick(t *testing.T) {
	m := &mockBlock{content: "hello\nagain"}
	lv := NewLazyScrollView(func(width int) *mockBlock { return &mockBlock{} })
	lv.blocks = []BlockMeta[*mockBlock]{{Block: m}}
	lv.RecomputeContent()

	cmd := lv.SendBlockMsg(0, tea.MouseReleaseMsg{Button: tea.MouseLeft}, 0)
	if cmd != nil {
		t.Fatal("expected no command from mock block")
	}
}

func TestRefreshesOnlyBlockCache(t *testing.T) {
	first := &mockBlock{content: "first"}
	second := &mockBlock{content: "second"}
	lv := NewLazyScrollView(func(width int) *mockBlock { return &mockBlock{} })
	lv.blocks = []BlockMeta[*mockBlock]{{Block: first}, {Block: second}}
	lv.RecomputeContent()

	first.renders = 0
	second.renders = 0
	second.content = "second updated"
	lv.RefreshBlocks(1, 1)

	if first.renders != 0 {
		t.Fatalf("first block renders = %d, want 0", first.renders)
	}
	if second.renders != 1 {
		t.Fatalf("second block renders = %d, want 1", second.renders)
	}
	lines := make([]string, 0, lv.totalLines)
	for i := 0; i < lv.totalLines; i++ {
		lines = append(lines, lv.lineAt(i))
	}
	if !strings.Contains(strings.Join(lines, "\n"), "second updated") {
		t.Fatalf("cache did not update changed block: %q", lines)
	}
}

func TestResizeDefersBlockRerender(t *testing.T) {
	first := &mockBlock{content: "first"}
	second := &mockBlock{content: "second"}
	lv := NewLazyScrollView(func(width int) *mockBlock { return &mockBlock{} })
	lv.blocks = []BlockMeta[*mockBlock]{{Block: first}, {Block: second}}
	lv.RecomputeContent()

	first.renders = 0
	second.renders = 0
	lv.Update(tea.WindowSizeMsg{Width: 41, Height: 3})

	if first.renders != 0 || second.renders != 0 {
		t.Fatalf("resize rendered blocks immediately: first=%d second=%d", first.renders, second.renders)
	}
	if !lv.IsBlockDirty(0) || !lv.IsBlockDirty(1) {
		t.Fatal("resize should mark block caches dirty")
	}

	_ = lv.Render()
	if first.renders == 0 {
		t.Fatal("render should refresh visible dirty block")
	}
}

func TestAnimationTickDeliversToAllAnimating(t *testing.T) {
	m := &mockBlock{content: "hello", animating: true}
	lv := NewLazyScrollView(func(width int) *mockBlock { return &mockBlock{} })
	lv.blocks = []BlockMeta[*mockBlock]{{Block: m}}
	lv.RecomputeContent()
	lv.setBlockAnimating(0, true)

	m.updated = 0
	lv.Update(common.AnimationTickMsg(time.Now()))

	if m.updated == 0 {
		t.Fatal("animating block should receive tick")
	}
}

func TestAnimationTickSkipsNonAnimating(t *testing.T) {
	m := &mockBlock{content: "hello", animating: false}
	lv := NewLazyScrollView(func(width int) *mockBlock { return &mockBlock{} })
	lv.blocks = []BlockMeta[*mockBlock]{{Block: m}}
	lv.RecomputeContent()

	m.updated = 0
	lv.Update(common.AnimationTickMsg(time.Now()))

	if m.updated != 0 {
		t.Fatalf("non-animating block should not receive tick, updated=%d", m.updated)
	}
}

func TestAnimationTickRespectsAllBlocks(t *testing.T) {
	idle := &mockBlock{content: "idle", animating: false}
	active := &mockBlock{content: "active", animating: true}
	lv := NewLazyScrollView(func(width int) *mockBlock { return &mockBlock{} })
	lv.blocks = []BlockMeta[*mockBlock]{{Block: idle}, {Block: active}}
	lv.RecomputeContent()
	lv.setBlockAnimating(1, true)

	idle.renders = 0
	active.renders = 0
	idle.updated = 0
	active.updated = 0
	lv.Update(common.AnimationTickMsg(time.Now()))

	if idle.updated != 0 {
		t.Fatalf("idle block updated = %d, want 0", idle.updated)
	}
	if active.updated == 0 {
		t.Fatal("active block should be ticked")
	}
}
