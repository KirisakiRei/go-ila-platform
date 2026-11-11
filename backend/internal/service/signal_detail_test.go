package service

import "testing"

// TestParseHtfSweepMenerimaCamelCase memastikan JSON proto dari engine
// (camelCase) terbaca lengkap, termasuk FVG dan lilin sweep.
func TestParseHtfSweepMenerimaCamelCase(t *testing.T) {
	raw := `{
		"detected": true,
		"sweepInProgress": false,
		"direction": "SELL",
		"stopLoss": 2320.5,
		"targetDol": 2380.25,
		"details": "sweep high H4",
		"fvg": {"type": "bearish", "top": 2350.5, "bottom": 2345.25, "mitigated": false},
		"sweepCandle": {"timestamp": "2026-05-10T08:00:00Z", "open": 2340, "high": 2356, "low": 2338, "close": 2350, "isClosed": true},
		"sweptCandle": {"timestamp": "2026-05-10T04:00:00Z", "open": 2335, "high": 2355, "low": 2330, "close": 2340, "isClosed": true}
	}`

	sweep := parseHtfSweep(raw)
	if sweep == nil {
		t.Fatal("parseHtfSweep mengembalikan nil")
	}

	if !sweep.Detected || sweep.Direction != "SELL" {
		t.Errorf("detected/direction tidak terbaca: %+v", sweep)
	}
	if sweep.StopLoss != 2320.5 || sweep.TargetDOL != 2380.25 {
		t.Errorf("stop_loss/target_dol tidak terbaca: %v / %v", sweep.StopLoss, sweep.TargetDOL)
	}
	if sweep.FVG == nil || sweep.FVG.Top != 2350.5 || sweep.FVG.Bottom != 2345.25 {
		t.Errorf("FVG tidak terbaca: %+v", sweep.FVG)
	}
	if sweep.SweptCandle == nil || sweep.SweptCandle.High != 2355 {
		t.Errorf("lilin tersapu tidak terbaca: %+v", sweep.SweptCandle)
	}
}

// TestParseHtfSweepMenerimaProtoNames memastikan baris lama dengan nama
// snake_case tetap terbaca setelah format baru diperkenalkan.
func TestParseHtfSweepMenerimaProtoNames(t *testing.T) {
	raw := `{"detected": true, "sweep_in_progress": true, "direction": "BUY", "stop_loss": 2310.5, "target_dol": 2400}`

	sweep := parseHtfSweep(raw)
	if sweep == nil {
		t.Fatal("parseHtfSweep mengembalikan nil")
	}
	if !sweep.SweepInProgress {
		t.Error("sweep_in_progress tidak terbaca")
	}
	if sweep.StopLoss != 2310.5 || sweep.TargetDOL != 2400 {
		t.Errorf("harga tidak terbaca: %v / %v", sweep.StopLoss, sweep.TargetDOL)
	}
}

func TestParseCISDMembacaLevel(t *testing.T) {
	raw := `{"detected": true, "direction": "BUY", "cisdLevel": 2344.75, "extremeSl": 2338.5, "entryPrice": 2345.1, "details": "break struktur"}`

	cisd := parseCISD(raw)
	if cisd == nil {
		t.Fatal("parseCISD mengembalikan nil")
	}
	if cisd.CISDLevel != 2344.75 || cisd.ExtremeSL != 2338.5 || cisd.EntryPrice != 2345.1 {
		t.Errorf("level CISD tidak terbaca: %+v", cisd)
	}
}

// TestParseRincianJSONRusak memastikan data rusak tidak membuat dashboard
// gagal: rincian yang tidak terbaca cukup diabaikan.
func TestParseRincianJSONRusak(t *testing.T) {
	if got := parseHtfSweep("{bukan json}"); got != nil {
		t.Errorf("JSON rusak seharusnya nil, dapat %+v", got)
	}
	if got := parseCISD(""); got != nil {
		t.Errorf("string kosong seharusnya nil, dapat %+v", got)
	}
}
