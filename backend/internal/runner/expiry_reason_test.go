package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
	ingest "ila/backend/internal/runner/ingest"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestKedaluwarsaDibedakanDariMenunggu memastikan baris sinyal basi yang
// statusnya masih WAIT_* dilaporkan sebagai "menunggu setup berikutnya",
// bukan "sudah kedaluwarsa". Engine mengirim ulang status menunggu pada slot
// yang sama, sehingga baris lama memang basi tetapi setup-nya belum pernah
// menawarkan entry — menyebutnya "terlambat" menyesatkan trader.
func TestKedaluwarsaDibedakanDariMenunggu(t *testing.T) {
	kasus := []struct {
		nama   string
		status string
		want   runner.RejectReason
	}{
		{
			nama:   "masih menunggu HTF",
			status: model.SignalStatusWaitHTFSweep,
			want:   runner.RejectWaitingNext,
		},
		{
			nama:   "masih menunggu LTF",
			status: model.SignalStatusWaitLTFCISD,
			want:   runner.RejectWaitingNext,
		},
		{
			nama:   "benar-benar siap lalu basi",
			status: model.SignalStatusReadyToEntry,
			want:   runner.RejectExpired,
		},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			store := newFakeStore()
			pub := &fakePublisher{}
			tambahAkun(store, "u1", "plus", "XAUUSD", "ila-2.0")
			store.subs = append(store.subs, model.Subscription{
				UserID: "u1", Symbol: "XAUUSD", IsActive: true,
			})

			sig := readySignal("XAUUSD", time.Hour)
			// Masa berlaku sudah lewat saat sinyal tiba.
			sig.ExpiresAt = timestamppb.New(time.Now().UTC().Add(-time.Minute))

			svc := newService(store, pub)

			stored, ok := ingest.ConvertSignal(sig)
			if !ok {
				t.Fatal("sinyal uji tidak valid")
			}
			// Status menunggu ditiru apa adanya: kolom status disimpan
			// langsung, karena konversi protobuf selalu memetakan ke READY.
			stored.Status = k.status
			svc.ProcessSignalForAccount(context.Background(), "acct-u1", stored)

			us, err := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
			if err != nil {
				t.Fatalf("user signal tidak ditemukan: %v", err)
			}
			if us.Status != model.SignalStatusRejected {
				t.Fatalf("status = %q, ingin REJECTED", us.Status)
			}
			if us.RejectReason != string(k.want) {
				t.Fatalf("alasan = %q, ingin %q", us.RejectReason, string(k.want))
			}
		})
	}
}
