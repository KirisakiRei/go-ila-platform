package telegram

import "ila/backend/internal/model"

// Tahap adalah posisi setup dalam siklus enam langkah.
//
// Pemetaan ini tinggal di satu tempat karena notifikasi dan halaman pantau
// wajib menyebut angka yang sama. Sebelumnya keduanya punya salinan sendiri,
// dan salinan di notifikasi tidak membedakan tahap 1 dari tahap 2 sehingga
// WAIT_HTF_SWEEP selalu tampil sebagai "Area HTF kena" walau harga belum
// menyentuhnya.
type Tahap int

const (
	// TahapArah: membaca arah timeframe besar, belum ada reaksi harga.
	TahapArah Tahap = 1
	// TahapSweep: area HTF sudah tersentuh, menunggu candle HTF tutup.
	TahapSweep Tahap = 2
	// TahapTrigger: mencari trigger di timeframe kecil.
	TahapTrigger Tahap = 3
	// TahapSiap: entry sudah terbentuk dan menunggu keputusan.
	TahapSiap Tahap = 4
	// TahapPosisi: posisi sudah terbuka.
	TahapPosisi Tahap = 5
	// TahapSelesai: posisi sudah ditutup.
	TahapSelesai Tahap = 6
)

// TotalTahap adalah jumlah langkah dalam satu siklus.
const TotalTahap = 6

// String menampilkan tahap sebagai "1/6".
func (t Tahap) String() string {
	if t < TahapArah || t > TotalTahap {
		return ""
	}
	return angka(int(t)) + "/" + angka(TotalTahap)
}

func angka(n int) string {
	if n >= 0 && n <= 9 {
		return string(rune('0' + n))
	}
	return ""
}

// TahapDariStatusRadar memetakan status menunggu ke tahap siklus.
//
// Status kosong atau bukan status menunggu berarti bukan tahap radar.
func TahapDariStatusRadar(status string) Tahap {
	switch status {
	case model.SignalStatusWaitHTFSweep:
		return TahapArah
	case model.SignalStatusWaitHTFCandleClose:
		return TahapSweep
	case model.SignalStatusWaitLTFCISD:
		return TahapTrigger
	case model.SignalStatusReadyToEntry, model.SignalStatusOpportunity:
		return TahapSiap
	}
	if isILA3Wait(status) {
		return TahapTrigger
	}
	return 0
}

func isILA3Wait(status string) bool {
	const awalan = "WAIT_ILA3_"
	return len(status) > len(awalan) && status[:len(awalan)] == awalan
}

// KalimatTahapRadar memberi kalimat singkat untuk satu tahap radar.
func KalimatTahapRadar(t Tahap, htf, ltf string) string {
	switch t {
	case TahapArah:
		return "Baca arah " + htf
	case TahapSweep:
		return "Tunggu close " + htf
	case TahapTrigger:
		return "Cari entry di " + ltf
	case TahapSiap:
		return "Siap masuk"
	}
	return ""
}
