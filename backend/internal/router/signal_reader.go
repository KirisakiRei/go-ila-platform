package router

import (
	"context"

	"ila/backend/internal/model"
)

// signalByID adalah pembaca sinyal berdasarkan identitasnya.
type signalByID interface {
	FindBySignalID(ctx context.Context, signalID string) (*model.Signal, error)
}

// signalReader menyesuaikan pembaca sinyal repository menjadi antarmuka
// service.SignalDetailReader.
//
// Namanya sengaja berbeda karena repository memakai FindBySignalID (identitas
// bisnis sinyal), sedangkan paket service memakai FindSignal. Penyesuaian ini
// menjaga keduanya tidak perlu saling tahu penamaan internal masing-masing.
type signalReader struct {
	repo signalByID
}

// FindSignal mengembalikan satu sinyal berdasarkan identitasnya.
func (r signalReader) FindSignal(ctx context.Context, signalID string) (*model.Signal, error) {
	return r.repo.FindBySignalID(ctx, signalID)
}
