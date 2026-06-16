package conversation_single_simple

import (
	"fmt"
	"mini_agent/core"
	"strings"
)

// turnsStringerWithCache formats []core.Turn with caching, assuming only the last
// violateRange turns may change between calls.
type turnsStringerWithCache struct {
	lastStableIndex  int
	lastStableString string
	violateRange     int
}

func newTurnsStringerWithCache(violateRange int) *turnsStringerWithCache {
	return &turnsStringerWithCache{
		lastStableIndex: 0,
		violateRange:    violateRange,
	}
}

func (s *turnsStringerWithCache) markChange(idx int) {
	if idx <= s.lastStableIndex {
		s.lastStableIndex = 0
		s.lastStableString = ""
	}
}

func (s *turnsStringerWithCache) string(turns []core.Turn) string {
	newStableLen := len(turns) - s.violateRange
	if newStableLen < 0 {
		newStableLen = 0
	}

	boundary := s.lastStableIndex
	var result string
	if boundary > 0 {
		var sb strings.Builder
		sb.WriteString(s.lastStableString)
		sb.WriteString(formatTurnsFrom(turns[boundary:], boundary+1))
		result = sb.String()
	} else {
		result = formatTurnsFrom(turns, 1)
	}

	if newStableLen > 0 {
		s.lastStableIndex = newStableLen
		s.lastStableString = formatTurnsFrom(turns[:newStableLen], 1)
	} else {
		s.lastStableIndex = 0
		s.lastStableString = ""
	}

	return result
}

func formatTurnsFrom(turns []core.Turn, startNum int) string {
	var sb strings.Builder
	for i, turn := range turns {
		fmt.Fprintf(&sb, "Turn %d:\n", startNum+i)
		sb.WriteString("  ")
		sb.WriteString(strings.ReplaceAll(strings.TrimRight(turn.String(), "\n"), "\n", "\n  "))
		sb.WriteString("\n")
	}
	return sb.String()
}
