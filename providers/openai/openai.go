// Package openai implements the core.Provider interface for the DeepSeek
// (and OpenAI-compatible) chat completions API.
package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"mini_agent/core"
)

// ============================================================================
// Config
// ============================================================================

// Config configures the DeepSeek (or OpenAI-compatible) provider.
type Config struct {
	BaseURL string // API endpoint，default https://api.deepseek.com
	APIKey  string
	Model   string // model name，default deepseek-chat
	// Effort controls reasoning depth. Valid values:
	//   DeepSeek: "high" or "max" (default "high")
	//   OpenAI-compatible (non-DeepSeek): "low", "medium", "high"
	//   ""  → disabled
	Effort string
	Name   string
}

// New creates a core.Provider from Config.
func New(cfg Config) (core.Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-chat"
	}

	deepseek := isDeepSeek(cfg.BaseURL)
	effort := strings.ToLower(strings.TrimSpace(cfg.Effort))

	if deepseek {
		switch effort {
		case "off", "":
			effort = "" // explicitly disabled
		case "high", "max":
		default:
			return nil, fmt.Errorf("openai: deepseek effort must be high or max, got %q", effort)
		}
	} else if !deepseek {
		switch effort {
		case "max":
			effort = "high"
		case "low", "medium", "high":
		default:
			return nil, fmt.Errorf("openai: effort must be low, medium, or high for non-DeepSeek, got %q", effort)
		}
	}
	name := cfg.Name
	if name == "" {
		if effort == "" {
			name = cfg.Model
		} else {
			name = fmt.Sprintf("%s (%s)", cfg.Model, cfg.Effort)
		}

	}

	return &provider{
		cfg:      cfg,
		effort:   effort,
		deepseek: deepseek,
		http:     &http.Client{Timeout: 5 * time.Minute},
		name:     name,
	}, nil
}

func isDeepSeek(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "api.deepseek.com" || strings.HasSuffix(host, ".deepseek.com")
}

type provider struct {
	cfg      Config
	effort   string // reasoning_effort to send; "" = omit
	deepseek bool   // set thinking.type=enabled for DeepSeek
	http     *http.Client
	name     string
}

func (p *provider) Name() string {
	return p.name
}

func (p *provider) Stream(ctx context.Context, req core.Request) (core.OutStream[core.Chunk], error) {
	body, err := p.buildBody(req)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		strings.TrimRight(p.cfg.BaseURL, "/")+"/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, buf.String())
	}

	out := make(chan core.Chunk)
	go p.parseSSE(ctx, resp, out)
	return core.OutStream[core.Chunk](out), nil
}

func (p *provider) buildBody(req core.Request) ([]byte, error) {
	msgs := make([]wireMessage, len(req.Messages))
	for i, m := range req.Messages {
		wm, err := toWireMessage(m)
		if err != nil {
			return nil, fmt.Errorf("message[%d]: %w", i, err)
		}
		msgs[i] = wm
	}
	tools := make([]wireTool, len(req.Tools))
	for i, t := range req.Tools {
		tools[i] = wireTool{
			Type: "function",
			Function: wireFunc{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}
	r := wireRequest{
		Model:           p.cfg.Model,
		Messages:        msgs,
		Tools:           tools,
		Stream:          true,
		StreamOptions:   &wireStreamOpts{IncludeUsage: true},
		ReasoningEffort: p.effort,
	}
	if p.deepseek {
		if p.effort != "" {
			r.Thinking = &wireThinking{Type: "enabled"}
		} else {
			r.Thinking = &wireThinking{Type: "disabled"}
		}

	}
	if req.Temperature != 0 {
		r.Temperature = req.Temperature
	}
	if req.MaxTokens != 0 {
		r.MaxTokens = req.MaxTokens
	}
	return json.Marshal(r)
}

func toWireMessage(m core.Message) (wireMessage, error) {
	switch mt := m.(type) {
	case core.TextMsg:
		return wireMessage{Role: mt.RoleName, Content: mt.Content}, nil
	case core.AssistantMsg:
		wm := wireMessage{Role: "assistant", Content: mt.Content, ReasoningContent: mt.Reasoning}
		for _, tc := range mt.ToolCalls {
			wm.ToolCalls = append(wm.ToolCalls, wireToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: wireFuncCall{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		return wm, nil
	case core.ToolResultMsg:
		return wireMessage{
			Role:       "tool",
			Content:    mt.Content,
			ToolCallID: mt.CallID,
			Name:       mt.Name,
		}, nil
	default:
		return wireMessage{}, fmt.Errorf("unknown message type %T", m)
	}
}

func (p *provider) parseSSE(ctx context.Context, resp *http.Response, out chan<- core.Chunk) {
	defer resp.Body.Close()
	defer close(out)

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			resp.Body.Close()
		case <-done:
		}
	}()

	toolCalls := map[int]*core.ToolCall{}
	var order []int

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			break
		}

		var sr wireStreamResp
		if err := json.Unmarshal([]byte(data), &sr); err != nil {
			out <- core.Chunk{Err: fmt.Errorf("decode stream: %w", err)}
			return
		}
		if sr.Error != nil {
			out <- core.Chunk{Err: fmt.Errorf("api error: %s", sr.Error.Message)}
			return
		}
		if sr.Usage != nil {
			out <- core.Chunk{Usage: &core.Usage{
				PromptTokens:     sr.Usage.PromptTokens,
				CompletionTokens: sr.Usage.CompletionTokens,
				TotalTokens:      sr.Usage.TotalTokens,
			}}
		}
		if len(sr.Choices) == 0 {
			continue
		}

		for _, c := range sr.Choices {
			delta := c.Delta
			if len(sr.Choices) != 1 {
				panic(fmt.Sprintf("multiple choices in stream response: %d", len(sr.Choices)))
			}
			if delta.ReasoningContent != "" || delta.Content != "" {
				out <- core.Chunk{
					Text:      delta.Content,
					Reasoning: delta.ReasoningContent,
				}
			}
			for _, tc := range delta.ToolCalls {
				cur, ok := toolCalls[tc.Index]
				if !ok {
					cur = &core.ToolCall{}
					toolCalls[tc.Index] = cur
					order = append(order, tc.Index)
				}
				if tc.ID != "" {
					cur.ID = tc.ID
				}
				if tc.Function.Name != "" {
					cur.Name = tc.Function.Name
				}
				cur.Arguments += tc.Function.Arguments
			}
		}
	}

	if err := scanner.Err(); err != nil {
		out <- core.Chunk{Err: fmt.Errorf("read stream: %w", err)}
		return
	}

	sort.Ints(order)
	for _, idx := range order {
		tc := toolCalls[idx]
		if tc.ID == "" {
			panic(fmt.Sprintf("tool call at index %d missing ID", idx))
		}
		out <- core.Chunk{ToolCall: tc}
	}
}

// ============================================================================
// OpenAI wire types (package-private)
// ============================================================================

type wireRequest struct {
	Model           string          `json:"model"`
	Messages        []wireMessage   `json:"messages"`
	Tools           []wireTool      `json:"tools,omitempty"`
	Stream          bool            `json:"stream"`
	StreamOptions   *wireStreamOpts `json:"stream_options,omitempty"`
	Temperature     float64         `json:"temperature,omitempty"`
	MaxTokens       int             `json:"max_tokens,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
	Thinking        *wireThinking   `json:"thinking,omitempty"`
}

type wireThinking struct {
	Type string `json:"type"`
}

type wireStreamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

type wireMessage struct {
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCalls        []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	Name             string         `json:"name,omitempty"`
}

type wireTool struct {
	Type     string   `json:"type"`
	Function wireFunc `json:"function"`
}

type wireFunc struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type wireFuncCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type wireToolCall struct {
	Index    int          `json:"index"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function wireFuncCall `json:"function"`
}

type wireStreamResp struct {
	Choices []struct {
		Delta struct {
			Content          string         `json:"content"`
			ReasoningContent string         `json:"reasoning_content"`
			ToolCalls        []wireToolCall `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}
