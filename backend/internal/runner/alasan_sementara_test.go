package runner

import (
	"testing"
	"time"
)

// Kasus 16 Nov 2026: sinyal USTEC terbit Minggu malam NY, ditolak dengan
// "Pasar sedang tutup", lalu kedaluwarsa. Alasan sesaat itu tidak boleh
// menemani hasil akhir karena membingungkan pembaca riwayat.
func TestAlasanSesaatDigantiAlasanAkhir(t *testing.T) {
	hasil := alasanTerbaru(string(RejectMarketClosed), string(RejectExpired))
	if hasil != string(RejectExpired) {
		t.Fatalf("alasan = %q, ingin alasan kedaluwarsa", hasil)
	}
}

// Alasan akhir tidak boleh ditimpa alasan sesaat: setup yang sudah berakhir
// tidak bisa kembali menjadi "pasar tutup".
func TestAlasanAkhirTidakDitimpaSesaat(t *testing.T) {
	hasil := alasanTerbaru(string(RejectExpired), string(RejectMarketClosed))
	if hasil != string(RejectExpired) {
		t.Fatalf("alasan = %q, ingin tetap kedaluwarsa", hasil)
	}
}

// Alasan sesaat terbaru tetap berlaku selama sinyal belum berakhir.
func TestAlasanSesaatTerbaruMenimpaSesaatLama(t *testing.T) {
	hasil := alasanTerbaru(string(RejectMarketClosed), string(RejectDailyQuota))
	if hasil != string(RejectDailyQuota) {
		t.Fatalf("alasan = %q, ingin alasan kuota yang lebih baru", hasil)
	}
}

func TestAlasanKosongTidakMenghapusAlasanLama(t *testing.T) {
	if hasil := alasanTerbaru(string(RejectMarketClosed), ""); hasil != string(RejectMarketClosed) {
		t.Fatalf("alasan = %q, ingin alasan lama dipertahankan", hasil)
	}
	if hasil := alasanTerbaru("", string(RejectExpired)); hasil != string(RejectExpired) {
		t.Fatalf("alasan = %q, ingin alasan baru dipakai", hasil)
	}
}

// Alasan pasar harus dinilai pada jam sinyal, bukan jam pemeriksaan.
// 16 Nov 2026 11:45 WIB = Minggu 23:45 NY: sesi sudah buka sejak 17:00 NY,
// jadi "Pasar sedang tutup" salah untuk waktu itu.
func TestPasarDinilaiPadaJamSinyal(t *testing.T) {
	terbit := time.Date(2026, 11, 16, 4, 45, 0, 0, time.UTC) // 11:45 WIB, 23:45 NY Minggu
	if !ExpectedOpen(terbit) {
		t.Fatal("Minggu 23:45 NY seharusnya sudah buka")
	}
	// Sabtu pagi NY memang tutup, jadi perbedaannya bukan sekadar selalu true.
	sabtu := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	if ExpectedOpen(sabtu) {
		t.Fatal("Sabtu NY harus tutup")
	}
}
