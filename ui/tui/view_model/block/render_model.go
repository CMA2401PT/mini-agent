package block

import (
	"fmt"
	"mini_agent/core"
	"strings"
	"unicode"
)

type blockSectionModel struct {
	Index   int
	Type    SectionType
	Content string
	Summary string
	StartY  int
	Height  int
}

type toolCallViewModel struct {
	ID        string
	Name      string
	Arguments string
	Result    *core.ToolResultMsg
	Orphan    bool
}

type blockSections []blockSectionModel

func (b blockSections) GetLastReasoning() int {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i].Type == BlockSectionReasoning {
			return b[i].Index
		}
	}
	return -1
}

func (section blockSectionModel) foldedSummary() string {
	if section.Summary != "" {
		return section.Summary
	}
	return "▸ " + singleLine(section.Content)
}

func textBlock(label, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return label + ":"
	}
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return label + ":\n" + strings.Join(lines, "\n")
}

// 从 turn 收集信息块
func collectTurnSections(turn core.Turn) blockSections {
	var out []blockSectionModel
	matchedResults := map[int]bool{}
	var providerName, toolsName string

	fallback := func(value, fallbackValue string) string {
		if strings.TrimSpace(value) == "" {
			return fallbackValue
		}
		return value
	}

	add := func(section blockSectionModel) {
		if strings.TrimSpace(section.Content) == "" {
			return
		}
		section.Index = len(out)
		out = append(out, section)
	}

	findToolResult := func(turn core.Turn, after int, callID string) (core.ToolResultMsg, int, bool) {
		if callID == "" {
			return core.ToolResultMsg{}, -1, false
		}
		for i := after + 1; i < len(turn); i++ {
			msg, ok := turn[i].(core.ToolResultMsg)
			if ok && msg.CallID == callID {
				return msg, i, true
			}
		}
		return core.ToolResultMsg{}, -1, false
	}

	hasPriorToolCall := func(turn core.Turn, before int, callID string) bool {
		if callID == "" {
			return false
		}
		for i := 0; i < before; i++ {
			msg, ok := turn[i].(core.AssistantMsg)
			if !ok {
				continue
			}
			for _, tc := range msg.ToolCalls {
				if tc.ID == callID {
					return true
				}
			}
		}
		return false
	}

	for i, msg := range turn {
		switch m := msg.(type) {
		case core.TrasnparentTextMsg:
			switch m.RoleName {
			case "ProviderName":
				providerName = m.Content
			case "EnvName":
				toolsName = m.Content
			}
		case core.TextMsg:
			switch m.RoleName {
			case "system":
				add(blockSectionModel{Type: BlockSectionSystem, Content: m.Content})
			default:
				add(blockSectionModel{Type: BlockSectionInput, Content: m.Content})
			}
		case core.AssistantMsg:
			if m.ReasoningActive || m.Reasoning != "" || m.ReasoningSignature != "" {
				content := m.Reasoning
				if content == "" && m.ReasoningActive {
					content = "(waiting…)"
				}
				if m.ReasoningSignature != "" {
					if content != "" {
						content += "\n\n"
					}
					content += "[signed: " + m.ReasoningSignature + "]"
				}
				if content != "" {
					add(blockSectionModel{Type: BlockSectionReasoning, Content: content})
				}
			}
			if m.Content != "" {
				add(blockSectionModel{Type: BlockSectionAnswer, Content: m.Content})
			}
			for _, tc := range m.ToolCalls {
				result, resultIdx, done := findToolResult(turn, i, tc.ID)
				if done {
					matchedResults[resultIdx] = true
				}
				tool := toolCallViewModel{
					ID:        fallback(tc.ID, "no-id"),
					Name:      fallback(tc.Name, "tool"),
					Arguments: tc.Arguments,
				}
				if done {
					tool.Result = &result
				}
				section := genToolSection(tool)
				add(section)
			}
		case core.ToolResultMsg:
			if !matchedResults[i] && !hasPriorToolCall(turn, i, m.CallID) {
				result := m
				tool := toolCallViewModel{
					ID:     fallback(m.CallID, "no-id"),
					Name:   fallback(m.Name, "tool"),
					Result: &result,
					Orphan: true,
				}
				section := genToolSection(tool)
				add(section)
			}
		}
	}
	if meta := turnMetaSummary(toolsName, providerName); meta != "" {
		add(blockSectionModel{Type: BlockSectionMeta, Content: meta})
	}
	return out
}

func turnMetaSummary(toolsName, providerName string) string {
	var parts []string
	if providerName = displayName(providerName); providerName != "" {
		parts = append(parts, providerName)
	}
	if toolsName = displayName(toolsName); toolsName != "" {
		parts = append(parts, toolsName)
	}
	if len(parts) == 0 {
		return ""
	}
	return "  ▣  " + strings.Join(parts, " · ")
}

func displayName(name string) string {
	words := strings.FieldsFunc(strings.TrimSpace(name), func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	for i, word := range words {
		switch strings.ToLower(word) {
		case "deepseek":
			word = "DeepSeek"
		case "openai", "oai":
			word = "OpenAI"
		}
		words[i] = titleWord(word)
	}
	out := strings.Join(words, " ")
	return out
}

func titleWord(word string) string {
	if word == "" {
		return ""
	}
	var out strings.Builder
	capNext := true
	for _, r := range word {
		if unicode.IsLetter(r) && capNext {
			out.WriteRune(unicode.ToUpper(r))
			capNext = false
			continue
		}
		out.WriteRune(r)
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			capNext = true
		}
	}
	return out.String()
}

func genToolSection(tool toolCallViewModel) blockSectionModel {
	status := "进行中"
	if tool.Result != nil {
		status = "已完成"
	}
	section := blockSectionModel{Type: BlockSectionTools}
	section.Summary = fmt.Sprintf("▸ %s %s · %s", tool.Name, tool.ID, status)
	parts := []string{
		fmt.Sprintf("Tool: %s", tool.Name),
		fmt.Sprintf("ID: %s", tool.ID),
		fmt.Sprintf("Status: %s", status),
	}
	if !tool.Orphan {
		parts = append(parts, textBlock("Arguments", tool.Arguments))
	}
	if tool.Result != nil {
		resultName := tool.Result.Name
		if resultName == "" {
			resultName = tool.Name
		}
		parts = append(parts,
			fmt.Sprintf("Result Tool: %s", resultName),
			textBlock("Result", tool.Result.Content),
		)
	}
	section.Content = strings.Join(parts, "\n")
	return section
}
