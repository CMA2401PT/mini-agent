package intra_turn

import (
	"fmt"

	"mini_agent/core"
)

// PatchIncompleteCalls fills a placeholder result for any ToolCall missing its
// result.  Orphan ToolResultMsg (CallID not found in any preceding AssistantMsg
// ToolCalls) or a ToolCall with an empty ID cause a panic — they indicate a bug.
func PatchIncompleteCalls(turn core.Turn, placeholder string) []core.Message {
	var attachMessages []core.Message = nil
	// Collect all tool_call IDs.
	callIDs := map[string]bool{}
	for _, m := range turn {
		if a, ok := m.(core.AssistantMsg); ok {
			for _, tc := range a.ToolCalls {
				if tc.ID == "" {
					panic(fmt.Errorf("intra_turn: tool_call with empty ID in assistant message"))
				}
				callIDs[tc.ID] = true
			}
		}
	}

	// Every ToolResultMsg must reference a known tool_call.
	for _, m := range turn {
		if tr, ok := m.(core.ToolResultMsg); ok {
			if !callIDs[tr.CallID] {
				panic(fmt.Errorf("intra_turn: orphan tool_result %q — no matching tool_call", tr.CallID))
			}
		}
	}

	// Patch missing results.
	for _, m := range turn {
		a, ok := m.(core.AssistantMsg)
		if !ok || len(a.ToolCalls) == 0 {
			continue
		}
		for _, tc := range a.ToolCalls {
			if hasResult(turn, tc.ID) {
				continue
			}
			attachMessages = append(attachMessages, core.ToolResultMsg{
				CallID:  tc.ID,
				Name:    tc.Name,
				Content: placeholder,
			})
		}
	}
	return attachMessages
}

func hasResult(turn core.Turn, callID string) bool {
	for _, m := range turn {
		if tr, ok := m.(core.ToolResultMsg); ok && tr.CallID == callID {
			return true
		}
	}
	return false
}
