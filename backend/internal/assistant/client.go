package assistant

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ClientConfig mengatur koneksi ke 9router (OpenAI-compatible).
type ClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// ChatRequest adalah permintaan ke endpoint /chat/completions.
type ChatRequest struct {
	Model       string
	Messages    []ChatMessage
	Tools       []ToolDefinition
	Temperature *float64
	MaxTokens   int
}

// ChatResult adalah hasil lengkap satu putaran LLM (teks dan/atau tool_calls).
type ChatResult struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
}

// LLMClient mendefinisikan kontrak panggilan model OpenAI-compatible.
type LLMClient interface {
	Complete(ctx context.Context, req ChatRequest) (*ChatResult, error)
	Stream(ctx context.Context, req ChatRequest, onDelta func(text string) error) (*ChatResult, error)
}

type openAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewOpenAIClient membuat klien OpenAI-compatible untuk 9router.
func NewOpenAIClient(cfg ClientConfig) LLMClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	return &openAIClient{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:     strings.TrimSpace(cfg.APIKey),
		httpClient: &http.Client{Timeout: timeout},
	}
}

type openAIPayload struct {
	Model       string           `json:"model"`
	Messages    []ChatMessage    `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  string           `json:"tool_choice,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream"`
}

func (c *openAIClient) endpoint() string {
	if strings.HasSuffix(c.baseURL, "/chat/completions") {
		return c.baseURL
	}
	return c.baseURL + "/chat/completions"
}

func (c *openAIClient) Complete(ctx context.Context, req ChatRequest) (*ChatResult, error) {
	payload := openAIPayload{
		Model:       req.Model,
		Messages:    req.Messages,
		Tools:       req.Tools,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
	}
	if len(req.Tools) > 0 {
		payload.ToolChoice = "auto"
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("menyusun payload LLM: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("membuat permintaan LLM: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("menghubungi 9router: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("membaca respons LLM: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseUpstreamError(resp.StatusCode, body)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("mengurai JSON respons LLM: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("respons LLM tidak memiliki choices")
	}

	ch := parsed.Choices[0]
	return &ChatResult{
		Content:      ch.Message.Content,
		ToolCalls:    ch.Message.ToolCalls,
		FinishReason: ch.FinishReason,
	}, nil
}

func (c *openAIClient) Stream(ctx context.Context, req ChatRequest, onDelta func(text string) error) (*ChatResult, error) {
	payload := openAIPayload{
		Model:       req.Model,
		Messages:    req.Messages,
		Tools:       req.Tools,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	}
	if len(req.Tools) > 0 {
		payload.ToolChoice = "auto"
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("menyusun payload stream LLM: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("membuat permintaan stream LLM: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("menghubungi 9router stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, parseUpstreamError(resp.StatusCode, body)
	}

	// Bila upstream tidak mengembalikan SSE (mis. JSON tunggal dari proxy),
	// tangani secara defensif agar tetap berhasil.
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			return nil, fmt.Errorf("membaca respons JSON stream fallback: %w", err)
		}
		var parsed struct {
			Choices []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("mengurai JSON fallback stream: %w", err)
		}
		if len(parsed.Choices) == 0 {
			return nil, errors.New("respons LLM fallback tidak memiliki choices")
		}
		ch := parsed.Choices[0]
		if ch.Message.Content != "" && onDelta != nil {
			if err := onDelta(ch.Message.Content); err != nil {
				return nil, err
			}
		}
		return &ChatResult{
			Content:      ch.Message.Content,
			ToolCalls:    ch.Message.ToolCalls,
			FinishReason: ch.FinishReason,
		}, nil
	}

	return parseSSEStream(resp.Body, onDelta)
}

type streamToolCallChunk struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func parseSSEStream(r io.Reader, onDelta func(text string) error) (*ChatResult, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	var contentBuf strings.Builder
	var finishReason string
	toolMap := make(map[int]*ToolCall)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			Choices []struct {
				Delta struct {
					Content   string                `json:"content"`
					ToolCalls []streamToolCallChunk `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Abaikan chunk ping/metadata non-standar yang bukan JSON valid.
			continue
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return nil, fmt.Errorf("9router error: %s", chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		ch := chunk.Choices[0]
		if ch.FinishReason != "" {
			finishReason = ch.FinishReason
		}
		if ch.Delta.Content != "" {
			contentBuf.WriteString(ch.Delta.Content)
			if onDelta != nil {
				if err := onDelta(ch.Delta.Content); err != nil {
					return nil, err
				}
			}
		}
		for _, tc := range ch.Delta.ToolCalls {
			acc, exists := toolMap[tc.Index]
			if !exists {
				acc = &ToolCall{Type: "function"}
				toolMap[tc.Index] = acc
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Type != "" {
				acc.Type = tc.Type
			}
			if tc.Function.Name != "" {
				acc.Function.Name += tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				acc.Function.Arguments += tc.Function.Arguments
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("membaca stream LLM: %w", err)
	}

	var toolCalls []ToolCall
	if len(toolMap) > 0 {
		indices := make([]int, 0, len(toolMap))
		for idx := range toolMap {
			indices = append(indices, idx)
		}
		sort.Ints(indices)
		toolCalls = make([]ToolCall, 0, len(indices))
		for _, idx := range indices {
			tc := *toolMap[idx]
			if tc.Type == "" {
				tc.Type = "function"
			}
			toolCalls = append(toolCalls, tc)
		}
	}

	return &ChatResult{
		Content:      contentBuf.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}, nil
}

func parseUpstreamError(status int, body []byte) error {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	msg := ""
	if json.Unmarshal(body, &errResp) == nil {
		if errResp.Error.Message != "" {
			msg = errResp.Error.Message
		} else if errResp.Message != "" {
			msg = errResp.Message
		}
	}
	if msg == "" {
		msg = strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200]
		}
	}
	return fmt.Errorf("upstream LLM status %d: %s", status, msg)
}
