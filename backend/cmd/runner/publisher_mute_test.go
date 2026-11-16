package main

import (
	"context"
	"testing"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// storeUji menimpa hanya dua pembacaan yang dipakai gate notifikasi; sisanya
// memakai interface kosong karena tidak pernah dipanggil di jalur ini.
type storeUji struct {
	repository.RunnerRepository
	signals   map[string]*model.Signal
	positions []model.Position
}

func (s *storeUji) FindSignal(_ context.Context, signalID string) (*model.Signal, error) {
	if signal, ok := s.signals[signalID]; ok {
		return signal, nil
	}
	return nil, apperror.ErrNotFound
}

func (s *storeUji) ListOpenPositionsByUser(_ context.Context, _ string) ([]model.Position, error) {
	return s.positions, nil
}

// TestPosisiSejenisMenahanNotifikasiRadar memastikan kemajuan radar untuk slot
// yang posisinya sudah terbuka tidak dikirim lagi. Sinyal yang membuka posisi
// itu sendiri tetap lolos agar kabar eksekusinya tidak ikut tertahan.
func TestPosisiSejenisMenahanNotifikasiRadar(t *testing.T) {
	store := &storeUji{
		signals: map[string]*model.Signal{
			"sig-asal": {SignalID: "sig-asal", Symbol: "EURUSD", Direction: "BUY", Htf: "H4", Ltf: "M15"},
			"sig-baru": {SignalID: "sig-baru", Symbol: "EURUSD", Direction: "BUY", Htf: "H4", Ltf: "M15"},
			"sig-lain": {SignalID: "sig-lain", Symbol: "EURUSD", Direction: "BUY", Htf: "H1", Ltf: "M5"},
		},
		positions: []model.Position{
			{Symbol: "EURUSD", Direction: "BUY", SourceSignalID: "sig-asal", Status: model.PositionStatusOpen},
		},
	}
	p := publisherUji(nil)
	p.store = store
	ctx := context.Background()

	if !p.sudahPunyaPosisiSejenis(ctx, "u-1", store.signals["sig-baru"]) {
		t.Fatal("sinyal sejenis pada pair dan timeframe yang sama harus ditahan")
	}
	if p.sudahPunyaPosisiSejenis(ctx, "u-1", store.signals["sig-lain"]) {
		t.Fatal("timeframe berbeda tidak boleh ditahan")
	}
	if p.sudahPunyaPosisiSejenis(ctx, "u-1", store.signals["sig-asal"]) {
		t.Fatal("sinyal asal posisi harus tetap dikirim")
	}
}
