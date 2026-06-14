// Package anthropic implements the core.Provider interface for the DeepSeek
// Anthropic-compatible Messages API (POST /v1/messages, SSE streaming).
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mini_agent/core"
)

// ============================================================================
// Config
// ============================================================================

// Config configures the DeepSeek Anthropic-compatible provider.
type Config struct {
	BaseURL string // API endpoint, default https://api.deepseek.com/anthropic
	APIKey  string
	Model   string // model name, default deepseek-v4-pro
	// Effort controls reasoning depth: low|medium|high|max; "" = thinking disabled
	Effort string
	Name   string
}

// ============================================================================
// New
// ============================================================================

// New creates a core.Provider from Config.
func New(cfg Config) (core.Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com/anthropic"
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-v4-pro"
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic: api_key is required")
	}

	thinking := "enabled"
	effort := strings.ToLower(strings.TrimSpace(cfg.Effort))
	if effort != "" {
		switch effort {
		case "low", "medium":
			effort = "high"
			thinking = "enabled"
		case "high", "max":
			thinking = "enabled"
		case "":
			thinking = "disabled"
		default:
			return nil, fmt.Errorf("anthropic: effort must be low, medium, high, or max, got %q", cfg.Effort)
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
		thinking: thinking == "enabled",
		effort:   effort,
		http:     &http.Client{Timeout: 5 * time.Minute},
		name:     name,
	}, nil
}

// ============================================================================
// Provider
// ============================================================================

type provider struct {
	cfg      Config
	thinking bool
	effort   string // output_config.effort; "" = omit
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
		strings.TrimRight(p.cfg.BaseURL, "/")+"/v1/messages",
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", p.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

// ============================================================================
// buildRequest
// ============================================================================

const defaultMaxTokens = 8192

func (p *provider) buildBody(req core.Request) ([]byte, error) {
	var system []anthTextBlock
	var msgs []anthMessage

	appendContent := func(role string, blocks ...anthContentBlock) {
		if len(blocks) == 0 {
			return
		}
		if n := len(msgs); n > 0 && msgs[n-1].Role == role {
			msgs[n-1].Content = append(msgs[n-1].Content, blocks...)
			return
		}
		msgs = append(msgs, anthMessage{Role: role, Content: blocks})
	}

	for _, m := range req.Messages {
		switch mt := m.(type) {
		case core.TextMsg:
			if mt.RoleName == "system" {
				if mt.Content != "" {
					system = append(system, anthTextBlock{Type: "text", Text: mt.Content})
				}
			} else {
				if mt.Content != "" {
					appendContent("user", anthContentBlock{Type: "text", Text: mt.Content})
				}
			}
		case core.AssistantMsg:
			var blocks []anthContentBlock
			if p.thinking && mt.Reasoning != "" {
				b := anthContentBlock{Type: "thinking", Thinking: mt.Reasoning}
				if mt.ReasoningSignature != "" {
					b.Signature = mt.ReasoningSignature
				}
				blocks = append(blocks, b)
			}
			if mt.Content != "" {
				blocks = append(blocks, anthContentBlock{Type: "text", Text: mt.Content})
			}
			for _, tc := range mt.ToolCalls {
				input := json.RawMessage(tc.Arguments)
				if len(input) == 0 {
					input = json.RawMessage("{}")
				}
				blocks = append(blocks, anthContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
			appendContent("assistant", blocks...)
		case core.ToolResultMsg:
			content := mt.Content
			if content == "" {
				content = "(no output)"
			}
			appendContent("user", anthContentBlock{
				Type:      "tool_result",
				ToolUseID: mt.CallID,
				Content:   content,
			})
		}
	}

	var tools []anthTool
	for _, t := range req.Tools {
		schema := t.Parameters
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		tools = append(tools, anthTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	r := anthRequest{
		Model:     p.cfg.Model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  msgs,
		Tools:     tools,
		Stream:    true,
	}
	if p.thinking {
		r.Thinking = &anthThinkingConfig{Type: "enabled"}
		if p.effort != "" {
			r.OutputConfig = &anthOutputConfig{Effort: p.effort}
		} else {
			r.Thinking = &anthThinkingConfig{Type: "disabled"}
		}
	}
	if req.Temperature != 0 {
		r.Temperature = req.Temperature
	}
	return json.Marshal(r)
}

// ============================================================================
// parseSSE
// ============================================================================

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

	acc := map[int]*core.ToolCall{}
	blockTypes := map[int]string{}
	var inTok, outTok int
	haveInput := false

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}

		var ev streamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			out <- core.Chunk{Err: fmt.Errorf("decode stream: %w", err)}
			return
		}

		switch ev.Type {
		case "message_start":
			if ev.Message != nil && ev.Message.Usage != nil {
				inTok = ev.Message.Usage.InputTokens
				haveInput = true
			}
		case "content_block_start":
			if ev.ContentBlock != nil {
				blockTypes[ev.Index] = ev.ContentBlock.Type
				switch ev.ContentBlock.Type {
				case "tool_use":
					acc[ev.Index] = &core.ToolCall{
						ID:   ev.ContentBlock.ID,
						Name: ev.ContentBlock.Name,
					}
				case "thinking":
					out <- core.Chunk{ReasoningActive: true}
				}
			}
		case "content_block_delta":
			if ev.Delta == nil {
				continue
			}
			switch ev.Delta.Type {
			case "text_delta":
				if ev.Delta.Text != "" {
					out <- core.Chunk{Text: ev.Delta.Text}
				}
			case "thinking_delta":
				if ev.Delta.Thinking != "" {
					out <- core.Chunk{Reasoning: ev.Delta.Thinking}
				}
			case "signature_delta":
				if ev.Delta.Signature != "" {
					out <- core.Chunk{Signature: ev.Delta.Signature}
				}
			case "input_json_delta":
				if tc := acc[ev.Index]; tc != nil {
					tc.Arguments += ev.Delta.PartialJSON
				}
			}
		case "content_block_stop":
			if tc := acc[ev.Index]; tc != nil {
				out <- core.Chunk{ToolCall: tc}
				delete(acc, ev.Index)
			}
			delete(blockTypes, ev.Index)
		case "message_delta":
			if ev.Usage != nil {
				haveInput = true
				outTok = ev.Usage.OutputTokens
			}
			if haveInput {
				out <- core.Chunk{Usage: &core.Usage{
					PromptTokens:     inTok,
					CompletionTokens: outTok,
					TotalTokens:      inTok + outTok,
				}}
			}
		case "message_stop":
		case "error":
			msg := "stream error"
			if ev.Error != nil && ev.Error.Message != "" {
				msg = ev.Error.Message
			}
			out <- core.Chunk{Err: fmt.Errorf("api error: %s", msg)}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		out <- core.Chunk{Err: fmt.Errorf("read stream: %w", err)}
	}
}

// ============================================================================
// Anthropic wire types (package-private)
// ============================================================================

type anthRequest struct {
	Model        string              `json:"model"`
	MaxTokens    int                 `json:"max_tokens"`
	System       []anthTextBlock     `json:"system,omitempty"`
	Messages     []anthMessage       `json:"messages"`
	Tools        []anthTool          `json:"tools,omitempty"`
	Thinking     *anthThinkingConfig `json:"thinking,omitempty"`
	OutputConfig *anthOutputConfig   `json:"output_config,omitempty"`
	Temperature  float64             `json:"temperature,omitempty"`
	Stream       bool                `json:"stream"`
}

type anthThinkingConfig struct {
	Type string `json:"type"`
}

type anthOutputConfig struct {
	Effort string `json:"effort"`
}

type anthTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthMessage struct {
	Role    string             `json:"role"`
	Content []anthContentBlock `json:"content"`
}

type anthContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

type anthTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type streamEvent struct {
	Type    string `json:"type"`
	Index   int    `json:"index"`
	Message *struct {
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
	ContentBlock *struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
	Delta *struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		Thinking    string `json:"thinking"`
		Signature   string `json:"signature"`
		PartialJSON string `json:"partial_json"`
	} `json:"delta"`
	Usage *struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
