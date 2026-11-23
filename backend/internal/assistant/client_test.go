package assistant

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAIClient_StreamTextAndToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path salah: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("auth header salah: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Halo "}}]}`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"trader"}}]}`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_","arguments":"{\"symbol\":"}}]}}]}`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"signals","arguments":"\"XAUUSD\"}"}}]},"finish_reason":"tool_calls"}]}`)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer srv.Close()

	client := NewOpenAIClient(ClientConfig{
		BaseURL: srv.URL + "/v1",
		APIKey:  "sk-test",
		Timeout: 5 * time.Second,
	})

	var streamed strings.Builder
	res, err := client.Stream(context.Background(), ChatRequest{
		Model:    "gpt-test",
		Messages: []ChatMessage{{Role: "user", Content: "cek"}},
	}, func(delta string) error {
		streamed.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}
	if streamed.String() != "Halo trader" || res.Content != "Halo trader" {
		t.Fatalf("konten salah: %q / %q", streamed.String(), res.Content)
	}
	if len(res.ToolCalls) != 1 {
		t.Fatalf("diharapkan 1 tool call, dapat %d", len(res.ToolCalls))
	}
	tc := res.ToolCalls[0]
	if tc.ID != "call_1" || tc.Function.Name != "get_signals" || tc.Function.Arguments != `{"symbol":"XAUUSD"}` {
		t.Fatalf("akumulasi tool call salah: %+v", tc)
	}
}
