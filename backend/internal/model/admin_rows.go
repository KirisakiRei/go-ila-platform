package model

import "time"

// AdminLegalAcceptance adalah bukti persetujuan dokumen oleh pengguna.
type AdminLegalAcceptance struct {
	Slug       string    `json:"slug"`
	Title      string    `json:"title"`
	Version    string    `json:"version"`
	AcceptedAt time.Time `json:"accepted_at"`
	IPHash     string    `json:"ip_hash"`
}

// AdminNoteRow adalah catatan internal beserta penulisnya.
type AdminNoteRow struct {
	ID         string    `json:"id"`
	AuthorID   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}
