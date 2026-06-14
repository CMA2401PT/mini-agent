package core

import (
	"fmt"
	"strings"
)

// Turn is one conversation turn: user input + assistant reply + tool results.
type Turn []Message

// String returns a debug-friendly representation, one message per line.
func (t Turn) String() string {
	var sb strings.Builder
	for _, msg := range t {
		if s, ok := msg.(fmt.Stringer); ok {
			sb.WriteString(s.String())
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// CloneTurns returns a deep copy of src.
func CloneTurns(src []Turn) []Turn {
	dst := make([]Turn, len(src))
	for i, t := range src {
		dst[i] = make(Turn, len(t))
		for j, m := range t {
			dst[i][j] = m.Clone()
		}
	}
	return dst
}
