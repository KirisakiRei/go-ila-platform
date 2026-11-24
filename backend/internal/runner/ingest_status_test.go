package runner_test

import (
	"context"
	"testing"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// kirimStatusMenunggu mengirim satu status tampilan untuk slot tetap.
func kirimStatusMenunggu(t *testing.T, svc *runner.Service, id string, status enginev1.SignalStatus) {
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

// TestHandleSignalMemperbaruiStatusMenunggu memastikan baris status radar ikut
// bergerak. Baris ini memakai ID tetap per slot, sedangkan UpsertSignal sengaja
// tidak menimpa kolom status; tanpa pembaruan khusus, statusnya macet di nilai
// pertama dan radar tampak tidak pernah berubah.
func TestHandleSignalMemperbaruiStatusMenunggu(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})

	const id = "status-XAUUSD-H4-M15"
	kirimStatusMenunggu(t, svc, id, enginev1.SignalStatus_WAIT_HTF_SWEEP)
	if got := store.signals[id].Status; got != model.SignalStatusWaitHTFSweep {
		t.Fatalf("status awal = %q, ingin WAIT_HTF_SWEEP", got)
	}

	kirimStatusMenunggu(t, svc, id, enginev1.SignalStatus_WAIT_LTF_CISD)
	if got := store.signals[id].Status; got != model.SignalStatusWaitLTFCISD {
		t.Errorf("status radar = %q, ingin WAIT_LTF_CISD", got)
	}
}

// TestHandleSignalTidakMengubahStatusLokal memastikan kiriman ulang status
// tampilan tidak menurunkan status lokal yang sudah lebih maju (ACKED).
func TestHandleSignalTidakMengubahStatusLokal(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})

	const id = "status-XAUUSD-H4-M15"
	store.signals[id] = &model.Signal{
		SignalID: id,
		Status:   model.SignalStatusAcked,
	}

	kirimStatusMenunggu(t, svc, id, enginev1.SignalStatus_WAIT_LTF_CISD)

	if got := store.signals[id].Status; got != model.SignalStatusAcked {
		t.Errorf("status = %q, ingin tetap ACKED", got)
	}
}

// TestHandleSignalMemperbaruiArahDanStrukturRadar memastikan baris status radar
// yang berganti bias (mis. BUY -> SELL -> NEUTRAL) ikut memperbarui direction,
// htf_sweep_json, cisd_json, dan engine_version agar tidak bertentangan dengan
// next_action.
func TestHandleSignalMemperbaruiArahDanStrukturRadar(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})

	const id = "status-XAUUSD-D1-H1"
	svc.HandleSignal(context.Background(), &enginev1.Signal{
		SignalId:        id,
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		EngineVersion:   "2.0.0",
		Htf:             "D1",
		Ltf:             "H1",
		Status:          enginev1.SignalStatus_WAIT_LTF_CISD,
		Direction:       enginev1.Direction_BUY,
		Summary:         "lama buy",
		NextAction:      "lama buy",
		HtfSweep:        &enginev1.HTFSweepInfo{Direction: "BULLISH", Details: "FVG bullish lama"},
	})

	svc.HandleSignal(context.Background(), &enginev1.Signal{
		SignalId:        id,
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		EngineVersion:   "2.1.1",
		Htf:             "D1",
		Ltf:             "H1",
		Status:          enginev1.SignalStatus_WAIT_LTF_CISD,
		Direction:       enginev1.Direction_SELL,
		Summary:         "menunggu -CISD di H1",
		NextAction:      "menunggu -CISD di H1",
		HtfSweep:        &enginev1.HTFSweepInfo{Direction: "BEARISH", Details: "Swing High D1 C2 aktif"},
		Cisd:            &enginev1.CISDInfo{Direction: "BEARISH", CisdLevel: 3632.97, Details: "-CISD belum terkonfirmasi"},
	})

	got := store.signals[id]
	if got.Direction != "SELL" {
		t.Fatalf("direction radar = %q, ingin SELL", got.Direction)
	}
	if got.EngineVersion != "2.1.1" {
		t.Errorf("engine_version = %q, ingin 2.1.1", got.EngineVersion)
	}
	if got.CisdJSON == "" || got.HtfSweepJSON == "" {
		t.Errorf("htf_sweep_json / cisd_json harus diperbarui: htf=%q cisd=%q", got.HtfSweepJSON, got.CisdJSON)
	}

	// Kembali ke WAIT_HTF_SWEEP tanpa arah -> direction wajib kembali NEUTRAL.
	svc.HandleSignal(context.Background(), &enginev1.Signal{
		SignalId:        id,
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		EngineVersion:   "2.1.1",
		Htf:             "D1",
		Ltf:             "H1",
		Status:          enginev1.SignalStatus_WAIT_HTF_SWEEP,
		Direction:       enginev1.Direction_DIRECTION_UNSPECIFIED,
		Summary:         "belum ada C2",
		NextAction:      "belum ada C2",
	})
	if got := store.signals[id]; got.Direction != "NEUTRAL" || got.Status != model.SignalStatusWaitHTFSweep || got.HtfSweepJSON != "" || got.CisdJSON != "" {
		t.Errorf("reset radar ke WAIT_HTF_SWEEP gagal: dir=%q status=%q htf=%q cisd=%q", got.Direction, got.Status, got.HtfSweepJSON, got.CisdJSON)
	}
}
