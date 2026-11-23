package model

import "time"

// AISession menyimpan satu sesi percakapan berkesinambungan per pengguna
// (1 user = 1 sesi), termasuk ringkasan gulung hasil compaction dan jejak
// waktu pesan terakhir untuk mendeteksi jeda idle.
type AISession struct {
	ID                 string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID             string     `gorm:"type:varchar(36);uniqueIndex;not null" json:"user_id"`
	RollingSummary     string     `gorm:"type:text;not null;default:''" json:"rolling_summary"`
	CompactedBeforeSeq int64      `gorm:"not null;default:0" json:"compacted_before_seq"`
	LastMessageAt      *time.Time `gorm:"index" json:"last_message_at"`
	LastCompactedAt    *time.Time `json:"last_compacted_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (AISession) TableName() string {
	return "ai_session"
}

// AIMessage menyimpan satu pesan dalam sesi ILA AI. Pesan yang sudah
// di-compact ditandai IsCompacted=true agar tidak lagi memenuhi jendela konteks
// LLM, namun tetap tersedia saat pengguna menggulir riwayat chat di UI.
type AIMessage struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	SessionID     string    `gorm:"type:varchar(36);index:idx_ai_msg_session_seq,priority:1;not null" json:"session_id"`
	UserID        string    `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Seq           int64     `gorm:"index:idx_ai_msg_session_seq,priority:2;not null" json:"seq"`
	Role          string    `gorm:"type:varchar(20);not null" json:"role"` // user | assistant | tool
	Content       string    `gorm:"type:text;not null;default:''" json:"content"`
	ToolCallsJSON string    `gorm:"type:text;not null;default:''" json:"tool_calls_json,omitempty"`
	ToolCallID    string    `gorm:"type:varchar(80);not null;default:''" json:"tool_call_id,omitempty"`
	ToolName      string    `gorm:"type:varchar(64);not null;default:''" json:"tool_name,omitempty"`
	ToolsUsedJSON string    `gorm:"type:text;not null;default:''" json:"tools_used_json,omitempty"`
	IsCompacted   bool      `gorm:"not null;default:false;index" json:"is_compacted"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

// TableName menetapkan nama tabel.
func (AIMessage) TableName() string {
	return "ai_message"
}

// AIUserMemory menyimpan catatan fakta awet per pengguna hasil ekstraksi
// compaction (mirip MEMORY.md dinamis per user pada pola nanobot).
type AIUserMemory struct {
	ID             string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID         string    `gorm:"type:varchar(36);uniqueIndex;not null" json:"user_id"`
	MemoryMarkdown string    `gorm:"type:text;not null;default:''" json:"memory_markdown"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (AIUserMemory) TableName() string {
	return "ai_user_memory"
}
