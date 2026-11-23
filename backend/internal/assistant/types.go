package assistant

import "encoding/json"

// ChatMessage merepresentasikan satu pesan dalam format OpenAI-compatible
// yang dikirim ke maupun diterima dari 9router.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall adalah permintaan eksekusi tool dari model LLM.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction memuat nama fungsi dan argumen JSON mentah.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition mendefinisikan satu tool pada payload OpenAI tools[].
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function ToolFunctionSchema `json:"function"`
}

// ToolFunctionSchema menjelaskan nama, deskripsi, dan skema parameter tool.
type ToolFunctionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// StreamEvent dikirim dari AssistantService ke handler SSE dan UI.
type StreamEvent struct {
	Type    string `json:"type"` // status | tool_start | tool_done | action_applied | delta | done | error
	Stage   string `json:"stage,omitempty"`
	Label   string `json:"label,omitempty"`
	Name    string `json:"name,omitempty"`
	OK      *bool  `json:"ok,omitempty"`
	Summary string `json:"summary,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Symbol  string `json:"symbol,omitempty"`
	Text    string `json:"text,omitempty"`
	MsgID   string `json:"message_id,omitempty"`
	Seq     int64  `json:"seq,omitempty"`
	Compact bool   `json:"compacted,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// UsedToolMeta mencatat ringkasan tool yang dipakai untuk disimpan di AIMessage
// agar tetap terlihat saat pengguna memuat ulang halaman riwayat.
type UsedToolMeta struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	OK      bool   `json:"ok"`
	Summary string `json:"summary,omitempty"`
}

// ActionAppliedEvent menandai perubahan state platform yang perlu diketahui UI
// (mis. invalidate daftar watchlist pair).
type ActionAppliedEvent struct {
	Kind   string
	Symbol string
}
