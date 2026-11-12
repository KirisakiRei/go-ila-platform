package runner_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// TestMaxManualTTLFactorTerkunci menjaga agar UI dan eksekusi memakai angka
// yang sama. Nilai ini dipakai paket service untuk menentukan tombol aktif.
func TestMaxManualTTLFactorTerkunci(t *testing.T) {
	if runner.MaxManualTTLFactor != 4 {
		t.Fatalf("MaxManualTTLFactor = %d, ingin 4", runner.MaxManualTTLFactor)
	}
}

// TestValidateSignalManualMemakaiBatasHarga memastikan perilaku yang diminta:
// sinyal yang sudah lewat masa berlaku tetap boleh dieksekusi lewat tombol
// selama belum melewati batas keras. Kelayakan harga diperiksa terpisah.
func TestValidateSignalManualMemakaiBatasHarga(t *testing.T) {
	now := time.Now().UTC()

	cases := []struct {
		name     string
		manual   bool
		status   string
		pubAgo   time.Duration // PublishedAt = now - pubAgo
		ttl      time.Duration // ExpiresAt = PublishedAt + ttl
		want     runner.RejectReason
		wantDesc string
	}{
		{
			name:     "manual: lewat masa berlaku tetapi di dalam batas keras",
			manual:   true,
			status:   model.SignalStatusReadyToEntry,
			pubAgo:   7 * time.Minute,
			ttl:      5 * time.Minute,
			want:     runner.RejectNone,
			wantDesc: "manual dalam 4x TTL harus lolos",
		},
		{
			name:     "manual: lewat batas keras ditolak",
			manual:   true,
			status:   model.SignalStatusReadyToEntry,
			pubAgo:   25 * time.Minute,
			ttl:      5 * time.Minute,
			want:     runner.RejectExpired,
			wantDesc: "lewat 20 menit harus ditolak",
		},
		{
			name:     "manual: status menunggu tetap ditolak",
			manual:   true,
			status:   "WAIT_HTF_SWEEP",
			pubAgo:   7 * time.Minute,
			ttl:      5 * time.Minute,
			want:     runner.RejectWaitingNext,
			wantDesc: "setup yang belum pernah menawarkan entry bukan soal jam",
		},
		{
			name:     "manual: masih berlaku, tidak berubah",
			manual:   true,
			status:   model.SignalStatusReadyToEntry,
			pubAgo:   1 * time.Minute,
			ttl:      5 * time.Minute,
			want:     runner.RejectNone,
			wantDesc: "perilaku lama harus dipertahankan",
		},
		{
			name:     "manual: tanpa rentang waktu ditolak",
			manual:   true,
			status:   model.SignalStatusReadyToEntry,
			pubAgo:   0, // diisi nol di bawah
			ttl:      0,
			want:     runner.RejectExpired,
			wantDesc: "baris rusak tidak boleh mendapat perpanjangan",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeStore()
			svc := newService(store, &fakePublisher{})

			signal := &model.Signal{
				SignalID:    "sig-uji",
				Symbol:      "XAUUSD",
				Direction:   "BUY",
				Status:      tc.status,
				EntryPrice:  4000,
				StopLoss:    3990,
				TakeProfit:  4025,
				PublishedAt: now.Add(-tc.pubAgo),
				ExpiresAt:   now.Add(-tc.pubAgo).Add(tc.ttl),
			}
			if tc.pubAgo == 0 {
				signal.PublishedAt = time.Time{}
				signal.ExpiresAt = now.Add(-time.Minute)
			}
			store.signals[signal.SignalID] = signal

			user := &model.User{Base: model.Base{ID: "u-uji"}, Name: "u-uji", Tier: "pro", Status: "active"}
			store.users[user.ID] = user

			// Approve menjalankan jalur manual yang sesungguhnya. Penolakan
			// karena saldo/akun tidak dipasang: yang diuji hanya gerbang waktu.
			reason, _ := svc.Approve(context.Background(), user.ID, signal.SignalID, "")

			if tc.manual && tc.want == runner.RejectNone {
				// Jalur manual boleh lanjut melewati gerbang waktu; penolakan
				// berikutnya (akun tidak ada) bukan urusan tes ini.
				if reason == runner.RejectExpired || reason == runner.RejectWaitingNext {
					t.Fatalf("gerbang waktu menolak padahal %s: %v", tc.wantDesc, reason)
				}
				return
			}
			if reason != tc.want {
				t.Fatalf("reason = %v, ingin %v (%s)", reason, tc.want, tc.wantDesc)
			}
		})
	}
}

// TestAutoModeTetapMenghormatiMasaBerlaku memastikan perpanjangan hanya milik
// jalur manual. Eksekusi otomatis harus tetap mengikuti jam: order otomatis
// yang terlambat berarti entry di harga yang sudah jauh berbeda dari setup.
func TestAutoModeTetapMenghormatiMasaBerlaku(t *testing.T) {
	store := newFakeStore()
	addUser(store, "u-auto", "plus", "auto", true)
	svc := newService(store, &fakePublisher{})

	// Lewat masa berlaku 2 menit; TTL 5 menit jadi masih di dalam 4x TTL,
	// tetapi jalur otomatis tidak boleh memakai perpanjangan itu.
	published := time.Now().UTC().Add(-7 * time.Minute)
	raw := &enginev1.Signal{
		SignalId:        "sig-telat",
		Symbol:          "XAUUSD",
		Status:          enginev1.SignalStatus_READY_TO_ENTRY,
		Action:          enginev1.SignalAction_SIGNAL_ACTION_PUBLISH,
		Direction:       enginev1.Direction_BUY,
		EntryPrice:      4000,
		StopLoss:        3990,
		TakeProfit:      4025,
		RiskReward:      2.5,
		PublishedAt:     timestamppb.New(published),
		ExpiresAt:       timestamppb.New(published.Add(5 * time.Minute)),
		StrategyVersion: "ila-2.0",
	}

	svc.HandleSignal(context.Background(), raw)

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u-auto")
	if len(positions) != 0 {
		t.Fatalf("posisi otomatis = %d, ingin 0 untuk sinyal kedaluwarsa", len(positions))
	}
}
