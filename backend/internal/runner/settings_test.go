package runner_test

import (
	"context"
	"testing"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// ILA 3.0 yang belum dirilis tidak boleh membuat engine dijalankan.
func TestComputeRequirementsIgnoresIla3WhenOff(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	store.users["u1"] = &model.User{Base: model.Base{ID: "u1"}, Tier: "plus", Status: "active"}
	store.subs = append(store.subs, model.Subscription{
		UserID: "u1", Symbol: "EURUSD", Strategy: "ila-3.0", IsActive: true,
	})

	svc := newServiceWithSettings(store, pub, runner.NewSettings(runner.ILA3Off))

	reqs, err := svc.ComputeRequirements(context.Background(), []runner.StrategyLocation{
		{Strategy: "ila-3.0", Addr: "ila3:9090"},
	})
	if err != nil {
		t.Fatalf("ComputeRequirements: %v", err)
	}
	if got := reqs["ila-3.0"]; len(got) != 0 {
		t.Fatalf("ila-3.0 = %v, seharusnya kosong saat belum dirilis", got)
	}
}

// Tahap beta sudah boleh menyiapkan engine untuk user Plus terpilih.
func TestComputeRequirementsIla3Beta(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	store.users["u1"] = &model.User{Base: model.Base{ID: "u1"}, Tier: "plus", Status: "active"}
	store.subs = append(store.subs, model.Subscription{
		UserID: "u1", Symbol: "EURUSD", Strategy: "ila-3.0", IsActive: true,
	})

	svc := newServiceWithSettings(store, pub, runner.NewSettings(runner.ILA3Beta))

	reqs, err := svc.ComputeRequirements(context.Background(), []runner.StrategyLocation{
		{Strategy: "ila-3.0", Addr: "ila3:9090"},
	})
	if err != nil {
		t.Fatalf("ComputeRequirements: %v", err)
	}
	if got := reqs["ila-3.0"]; len(got) != 1 || got[0] != "EURUSD" {
		t.Fatalf("ila-3.0 = %v, ingin [EURUSD]", got)
	}
}

// StrategyStatus memetakan tahap ke label yang dilihat user.
func TestStrategyStatusLabels(t *testing.T) {
	cases := map[string]string{
		runner.ILA3Off:      "coming_soon",
		runner.ILA3Internal: "coming_soon",
		runner.ILA3Beta:     "beta",
		runner.ILA3General:  "active",
	}
	for stage, want := range cases {
		s := runner.NewSettings(stage)
		if got := s.StrategyStatus("ila-3.0"); got != want {
			t.Errorf("tahap %s: status=%q, want %q", stage, got, want)
		}
	}

	s := runner.NewSettings(runner.ILA3Off)
	if got := s.StrategyStatus("ila-2.0"); got != "active" {
		t.Errorf("ILA 2.0 harus selalu aktif, dapat %q", got)
	}
}
