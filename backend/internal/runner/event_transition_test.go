package runner_test

import (
	"context"
	"testing"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/model"
)

// kirimStatusRadar mengirim satu status tampilan untuk slot tetap, sama seperti
// engine mengirim kemajuan radar berkala.
func kirimStatusRadar(t *testing.T, svc interface {
	HandleSignal(context.Context, *enginev1.Signal)
}, id string, status enginev1.SignalStatus) {
	t.Helper()
	svc.HandleSignal(context.Background(), &enginev1.Signal{
		SignalId:        id,
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		Htf:             "H4",
		Ltf:             "M15",
		Status:          status,
	})
}

// TestStatusBerulangTidakMencatatPeristiwa memastikan kiriman berkala dengan
// status yang sama tidak menambah baris jejak. Sebelumnya satu slot bisa
// menulis ratusan baris identik sehingga audit perpindahan tahap tidak terbaca.
func TestStatusBerulangTidakMencatatPeristiwa(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "manual", true)
	svc := newService(store, pub)
	const id = "status-XAUUSD-H4-M15"

	kirimStatusRadar(t, svc, id, enginev1.SignalStatus_WAIT_HTF_SWEEP)
	for i := 0; i < 5; i++ {
		kirimStatusRadar(t, svc, id, enginev1.SignalStatus_WAIT_HTF_SWEEP)
	}

	jumlahEvent := 0
	for _, ev := range store.events {
		if ev.SignalID == id {
			jumlahEvent++
		}
	}
	if jumlahEvent != 1 {
		t.Fatalf("peristiwa tercatat = %d, ingin 1 (kiriman berulang diabaikan)", jumlahEvent)
	}
}

// TestPeristiwaMenyimpanStatusAsal memastikan jejak mencatat dari tahap mana
// perpindahan terjadi, sehingga urutan tahap bisa diaudit.
func TestPeristiwaMenyimpanStatusAsal(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	addUser(store, "u1", "plus", "manual", true)
	const id = "status-XAUUSD-H4-M15"

	kirimStatusRadar(t, svc, id, enginev1.SignalStatus_WAIT_HTF_SWEEP)
	kirimStatusRadar(t, svc, id, enginev1.SignalStatus_WAIT_LTF_CISD)

	terakhir := store.events[len(store.events)-1]
	if terakhir.FromStatus != model.SignalStatusWaitHTFSweep {
		t.Fatalf("status asal = %q, ingin WAIT_HTF_SWEEP", terakhir.FromStatus)
	}
	if terakhir.ToStatus != model.SignalStatusWaitLTFCISD {
		t.Fatalf("status tujuan = %q, ingin WAIT_LTF_CISD", terakhir.ToStatus)
	}
}

// TestMundurTahapMencatatPeristiwa memastikan setup yang gagal di tahap lanjut
// dan kembali ke tahap awal tercatat sebagai perpindahan baru, bukan diabaikan.
func TestMundurTahapMencatatPeristiwa(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	addUser(store, "u1", "plus", "manual", true)
	const id = "status-XAUUSD-H4-M15"

	kirimStatusRadar(t, svc, id, enginev1.SignalStatus_WAIT_HTF_SWEEP)
	kirimStatusRadar(t, svc, id, enginev1.SignalStatus_WAIT_LTF_CISD)
	kirimStatusRadar(t, svc, id, enginev1.SignalStatus_WAIT_HTF_SWEEP)

	terakhir := store.events[len(store.events)-1]
	if terakhir.FromStatus != model.SignalStatusWaitLTFCISD {
		t.Fatalf("status asal = %q, ingin WAIT_LTF_CISD", terakhir.FromStatus)
	}
	if terakhir.ToStatus != model.SignalStatusWaitHTFSweep {
		t.Fatalf("status tujuan = %q, ingin WAIT_HTF_SWEEP (mundur)", terakhir.ToStatus)
	}
}
