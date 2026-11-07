package runner

import (
	"testing"
)

// Aturan harian yang disepakati owner:
//
//	Maksimal 2 trade per hari.
//	Trade pertama menang  -> berhenti, tidak boleh trade kedua.
//	Trade pertama kalah   -> boleh trade kedua.
//	Trade kedua apa pun   -> berhenti.
//
// Tes ini menelusuri alur itu langkah demi langkah, bukan memeriksa satu
// status lepas, agar urutannya tidak bisa rusak tanpa ketahuan.
func TestAlurDuaTradePerHari(t *testing.T) {
	policy := DefaultDailyPolicy().Clamp()

	type langkah struct {
		nama   string
		status DailyStatus
		want   RejectReason
	}

	skenario := []struct {
		nama    string
		langkah []langkah
	}{
		{
			nama: "trade pertama menang lalu berhenti",
			langkah: []langkah{
				{"awal hari", DailyStatus{}, RejectNone},
				{"setelah 1 win", DailyStatus{WinCount: 1, TotalTrades: 1}, RejectDailyWin},
			},
		},
		{
			nama: "trade pertama kalah lalu boleh trade kedua",
			langkah: []langkah{
				{"awal hari", DailyStatus{}, RejectNone},
				{"setelah 1 loss", DailyStatus{LossCount: 1, TotalTrades: 1}, RejectNone},
				// Setelah trade kedua menang: win menghentikan hari.
				{"setelah 1 loss + 1 win", DailyStatus{LossCount: 1, WinCount: 1, TotalTrades: 2}, RejectDailyWin},
			},
		},
		{
			nama: "trade pertama kalah, trade kedua kalah",
			langkah: []langkah{
				{"awal hari", DailyStatus{}, RejectNone},
				{"setelah 1 loss", DailyStatus{LossCount: 1, TotalTrades: 1}, RejectNone},
				{"setelah 2 loss", DailyStatus{LossCount: 2, TotalTrades: 2}, RejectDailyQuota},
			},
		},
		{
			nama: "belum ada trade tapi ada posisi hidup",
			langkah: []langkah{
				{"posisi masih open", DailyStatus{OpenCount: 1, TotalTrades: 1}, RejectOpenPosition},
			},
		},
	}

	for _, sk := range skenario {
		t.Run(sk.nama, func(t *testing.T) {
			for _, l := range sk.langkah {
				if got := EvaluateDaily(l.status, policy); got != l.want {
					t.Fatalf("%s: EvaluateDaily(%+v) = %q, ingin %q",
						l.nama, l.status, got, l.want)
				}
			}
		})
	}
}

// Satu loss saja tidak boleh mengunci hari; kuota yang mengunci.
func TestSatuLossTidakMengunciHari(t *testing.T) {
	policy := DefaultDailyPolicy().Clamp()
	status := DailyStatus{LossCount: 1, TotalTrades: 1}

	if reason := EvaluateDaily(status, policy); reason != RejectNone {
		t.Fatalf("satu loss seharusnya masih boleh entry, dapat %q", reason)
	}
}

// Hari baru mengembalikan kuota: status kosong berarti boleh entry lagi.
func TestHariBaruKuotaPulih(t *testing.T) {
	policy := DefaultDailyPolicy().Clamp()

	kemarin := DailyStatus{WinCount: 1, TotalTrades: 2}
	if reason := EvaluateDaily(kemarin, policy); reason == RejectNone {
		t.Fatal("hari yang sudah menang seharusnya terkunci")
	}

	hariIni := DailyStatus{}
	if reason := EvaluateDaily(hariIni, policy); reason != RejectNone {
		t.Fatalf("hari baru seharusnya boleh entry, dapat %q", reason)
	}
}
