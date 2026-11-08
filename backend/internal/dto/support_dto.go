package dto

import "time"

// SupportNoteItem adalah satu catatan jurnal dukungan.
type SupportNoteItem struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ActorID   string    `json:"actor_id"`
	Channel   string    `json:"channel"`
	Body      string    `json:"body"`
	IsPinned  bool      `json:"is_pinned"`
	CreatedAt time.Time `json:"created_at"`
}

// SupportNoteRequest adalah permintaan menambah catatan dukungan.
type SupportNoteRequest struct {
	Body     string `json:"body" binding:"required,min=2,max=2000"`
	Channel  string `json:"channel" binding:"omitempty,oneof=internal telegram email phone"`
	IsPinned bool   `json:"is_pinned"`
}
