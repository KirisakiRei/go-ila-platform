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
