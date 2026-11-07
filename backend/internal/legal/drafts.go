package legal

import (
	"fmt"
	"strings"
)

// Draft adalah isi baku versi 1 setiap dokumen. Disimpan di kode agar
// penerbitan awal tidak bergantung pada data manual, lalu boleh dikembangkan
// admin lewat CMS sebagai versi baru.
type Draft struct {
	Slug            string
	Title           string
	BodyMD          string
	RequireReaccept bool
}

// Drafts mengembalikan lima dokumen versi 1.
//
// Teks ini kerangka yang wajar untuk produk seperti ini, bukan nasihat hukum.
// Bagian bertanda [..] wajib dilengkapi sebelum dipakai komersial.
func Drafts() []Draft {
	return []Draft{
		{
			Slug:            SlugTOS,
			Title:           Title(SlugTOS),
			RequireReaccept: false,
			BodyMD:          strings.TrimSpace(termsOfService),
		},
		{
			Slug:            SlugPrivacy,
			Title:           Title(SlugPrivacy),
			RequireReaccept: false,
			BodyMD:          strings.TrimSpace(privacyPolicy),
		},
		{
			Slug:            SlugRisk,
			Title:           Title(SlugRisk),
			RequireReaccept: false,
			BodyMD:          strings.TrimSpace(riskDisclaimer),
		},
		{
			Slug:            SlugRefund,
			Title:           Title(SlugRefund),
			RequireReaccept: false,
			BodyMD:          strings.TrimSpace(refundPolicy),
		},
		{
			Slug:            SlugFairUse,
			Title:           Title(SlugFairUse),
			RequireReaccept: false,
			BodyMD:          strings.TrimSpace(fairUsePolicy),
		},
	}
}

// DraftBySlug mengembalikan draf satu dokumen.
func DraftBySlug(slug string) (Draft, bool) {
	for _, draft := range Drafts() {
		if draft.Slug == slug {
			return draft, true
		}
	}
	return Draft{}, false
}

// ContactLine menyusun baris kontak yang dipakai beberapa dokumen.
func ContactLine(email string) string {
	if email == "" {
		email = "[alamat surel dukungan]"
	}
	return fmt.Sprintf("Pertanyaan mengenai dokumen ini dapat dikirim ke %s.", email)
}

const termsOfService = `
# Syarat & Ketentuan

**Berlaku sejak:** [tanggal berlaku]
**Penyelenggara:** Imaginary Liquidity — [nama badan usaha], [alamat]

## 1. Definisi

- **Layanan** — platform berlangganan yang menyediakan analisis teknis berupa
  sinyal trading dan, pada paket tertentu, menjalankan eksekusi atas nama User.
- **Sinyal** — pemberitahuan bahwa suatu kondisi teknik terpenuhi pada sebuah
  instrumen, beserta rencana masuk, batas kerugian, dan target.
- **Akun Broker** — akun trading milik User pada broker pihak ketiga.
- **Langganan** — hak memakai Layanan untuk jangka waktu tertentu.

## 2. Ruang Lingkup Layanan

Layanan menyediakan analisis teknik, sinyal, dan (pada paket tertentu)
eksekusi otomatis sesuai aturan yang telah ditetapkan.

Layanan **tidak** menyediakan jasa kustodian dana, tidak mengelola dana User,
dan tidak menjadi perantara transaksi pada pasar. Dana User tetap berada pada
Akun Broker milik User dan tidak pernah berpindah ke Penyelenggara.

## 3. Bukan Nasihat Keuangan

Layanan **bukan** nasihat investasi, bukan rekomendasi keuangan, dan tidak
memperhitungkan kondisi keuangan pribadi User. User disarankan berkonsultasi
dengan penasihat yang berwenang sebelum mengambil keputusan.

## 4. Risiko

Perdagangan mata uang dan instrumen derivatif memiliki risiko kerugian yang
tinggi dan dapat melebihi modal awal. Risiko sepenuhnya diatur dalam
**Disclaimer Risiko Trading** yang merupakan bagian tidak terpisahkan dari
Ketentuan ini.

## 5. Kewajiban User

1. Akun Broker yang dihubungkan adalah milik User sendiri atau yang User berhak
   gunakan.
2. Kredensial yang diberikan sah dan User berwenang memberikannya.
3. User menjaga kerahasiaan akses akun platform miliknya.
4. User mematuhi hukum yang berlaku di wilayahnya.

Menghubungkan Akun Broker milik pihak lain tanpa izin merupakan pelanggaran
**berat**.

## 6. Larangan

1. Menyebarluaskan, menjual ulang, atau membagikan Sinyal tanpa izin tertulis.
2. Memakai satu Langganan untuk beberapa orang.
3. Membuat akun ganda untuk menyalahgunakan masa uji coba.
4. Melakukan manipulasi pembayaran.
5. Membanjiri sistem atau melakukan upaya akses tanpa izin.

## 7. Biaya & Pembayaran

Rincian harga, masa berlaku, dan pengembalian dana diatur dalam
**Kebijakan Pengembalian Dana**.

## 8. Penangguhan & Pemblokiran

Penyelenggara berhak menangguhkan atau memblokir akun apabila terdapat indikasi
pelanggaran terhadap Ketentuan ini, penyalahgunaan sistem, upaya manipulasi,
atau kewajiban pembayaran yang tidak dipenuhi. Alasan pemblokiran disampaikan
melalui kanal yang tersedia dan dicatat dalam catatan audit sistem.

## 9. Batas Tanggung Jawab

Sejauh diperbolehkan hukum yang berlaku, Penyelenggara tidak bertanggung jawab
atas kerugian yang timbul dari penggunaan Layanan, termasuk kerugian akibat
keputusan trading User.

## 10. Perubahan Ketentuan

Ketentuan dapat diperbarui. Setiap perubahan diterbitkan sebagai versi baru
dengan nomor dan tanggal berlaku. Apabila perubahan menyangkut hak atau
kewajiban User, User akan diminta menyetujui versi yang baru sebelum dapat
memakai fitur yang terkena perubahan.

## 11. Hukum yang Berlaku

Ketentuan ini tunduk pada hukum Republik Indonesia. [Pilihan hukum dan forum
penyelesaian sengketa wajib ditinjau ahli hukum.]

## 12. Kontak

[Alamat surel dukungan] / [alamat pos]
`

const privacyPolicy = `
# Kebijakan Privasi

**Berlaku sejak:** [tanggal berlaku]

## 1. Data yang Dikumpulkan

| Kategori | Contoh | Tujuan |
|---|---|---|
| Identitas | Nama, alamat surel | Akun & komunikasi |
| Autentikasi | Hash kata sandi, secret 2FA | Keamanan akun |
| Kredensial broker | Token broker (terenkripsi) | Eksekusi atas nama User |
| Aktivitas | Waktu masuk, hash alamat IP, perangkat | Keamanan & audit |
| Trading | Sinyal diterima, posisi, hasil | Layanan & riwayat |
| Pembayaran | Nomor tagihan, status, referensi gateway | Kewajiban pajak |
| Teknis | User agent, catatan sistem | Perbaikan layanan |

## 2. Yang Tidak Dikumpulkan

| Tidak dikumpulkan | Alasan |
|---|---|
| Nomor kartu kredit | Ditangani gateway; tidak pernah menyentuh server kami |
| Nomor identitas kependudukan | Tidak diperlukan untuk layanan |
| Kontak di perangkat | Tidak relevan |
| Data biometrik | Tidak relevan |

## 3. Cara Data Disimpan

Kata sandi disimpan sebagai hash satu arah dan tidak dapat dibaca ulang.
Kredensial Akun Broker dan secret 2FA disimpan dalam bentuk terenkripsi dengan
kunci yang tidak berada di dalam basis data. Kredensial tidak pernah ditampilkan
kembali kepada siapa pun, termasuk kepada staf.

## 4. Pihak Ketiga

| Pihak | Data yang diterima | Tujuan |
|---|---|---|
| Gateway pembayaran | Nomor tagihan, nominal, alamat surel | Memproses pembayaran |
| Broker | Kredensial, permintaan order | Eksekusi |
| Telegram | ID percakapan, isi notifikasi | Notifikasi |
| Penyedia server | Data tersimpan | Penyelenggaraan layanan |

## 5. Masa Simpan

| Data | Masa simpan |
|---|---|
| Catatan audit & posisi | 1 tahun, lalu diringkas |
| Tagihan & pembayaran | 10 tahun (kewajiban pajak) |
| Catatan aplikasi | 30 hari |
| Kredensial broker | Selama akun terhubung |

## 6. Hak User

| Hak | Cara |
|---|---|
| Melihat data sendiri | Dasbor |
| Meminta salinan data | Melalui dukungan |
| Menghapus akun | Diajukan; data pribadi dihapus, bukti persetujuan dianonimkan |
| Menolak notifikasi | Pengaturan akun |

## 7. Persetujuan

Persetujuan atas Kebijakan ini dicatat beserta nomor versi, waktu, dan hash
alamat IP, sebagai bukti bahwa User telah menyetujui kebijakan yang berlaku.

## 8. Kontak

[Alamat surel dukungan]
`

const riskDisclaimer = `
# Disclaimer Risiko Trading

**Berlaku sejak:** [tanggal berlaku]

## 1. Peringatan Umum

Perdagangan mata uang dan instrumen derivatif memiliki risiko kerugian yang
tinggi dan dapat melebihi modal awal. Tidak semua orang cocok untuk aktivitas
ini.

## 2. Tidak Ada Jaminan Keuntungan

Sistem ini menggunakan analisis teknik berdasarkan metode ILA. Metode tersebut
**tidak** menjamin keuntungan. Hasil masa lalu, baik yang ditampilkan maupun
yang diuji, **tidak** menjadi jaminan hasil di masa depan.

## 3. Sistem Dapat Membuat Kesalahan

Sistem dapat membuat kesalahan karena perhitungan yang kurang tepat, data pasar
yang tidak akurat, keterlambatan koneksi, gangguan jaringan, kesalahan perangkat
lunak, gangguan pihak ketiga (broker, penyedia data, penyedia server), serta
kondisi pasar yang tidak normal.

## 4. Eksekusi Tidak Sempurna

Harga eksekusi dapat berbeda dari harga sinyal karena slippage, spread,
likuiditas, dan kecepatan eksekusi broker. Sistem tidak menjamin eksekusi pada
harga tertentu.

## 5. Tanggung Jawab Tetap pada User

Apabila terjadi kesalahan eksekusi, User tetap bertanggung jawab atas posisi
yang telah terbuka. User dianjurkan memeriksa posisinya secara berkala pada
aplikasi broker.

## 6. Lakukan Riset Sendiri

User dianjurkan mempelajari metode, memahami aturan, dan memantau hasilnya.
Jangan menggunakan dana yang tidak siap untuk dirugikan.

## 7. Batas Tanggung Jawab

Sejauh diperbolehkan hukum yang berlaku, Penyelenggara tidak bertanggung jawab
atas kerugian yang timbul dari penggunaan layanan. [Bagian ini wajib ditinjau
ahli hukum.]

## 8. Pernyataan Persetujuan

> Saya memahami bahwa perdagangan memiliki risiko kehilangan dana, bahwa sistem
> ini tidak menjamin keuntungan, bahwa sistem dapat membuat kesalahan, dan bahwa
> seluruh keputusan trading adalah tanggung jawab saya sendiri.
`

const refundPolicy = `
# Kebijakan Pengembalian Dana

**Berlaku sejak:** [tanggal berlaku]

## 1. Ketentuan Pengembalian

| Situasi | Pengembalian |
|---|---|
| Belum memakai layanan, kurang dari 7 hari | 100% |
| Sudah memakai 1–7 hari | 50% |
| Sudah memakai lebih dari 7 hari | Tidak ada |
| Gangguan dari sisi kami lebih dari 48 jam | Pertimbangan khusus |
| Diblokir karena pelanggaran | Tidak ada |
| Masa uji coba | Tidak ada (gratis) |

## 2. Prosedur

1. Ajukan permintaan melalui kanal dukungan.
2. Keputusan diberikan paling lambat 7 hari kerja.
3. Bila disetujui, dana dikembalikan ke kanal pembayaran asal.

## 3. Hal yang Diperhatikan

Pengembalian dana tidak menghapus kewajiban User atas pelanggaran yang sudah
terjadi. Pemblokiran karena pelanggaran Ketentuan tidak memberikan hak
pengembalian.

## 4. Kontak

[Alamat surel dukungan]
`

const fairUsePolicy = `
# Kebijakan Penggunaan Wajar

**Berlaku sejak:** [tanggal berlaku]

## 1. Yang Dilarang

| Larangan | Contoh | Akibat |
|---|---|---|
| Membagikan sinyal | Mengirim ke grup, menjual sinyal | Blokir permanen |
| Akun berbagi | Satu langganan dipakai beberapa orang | Peringatan, lalu blokir |
| Akun ganda untuk uji coba | Mendaftar berulang dengan surel lain | Uji coba dihentikan |
| Akun broker orang lain | Memakai akun tanpa izin | Blokir segera |
| Manipulasi pembayaran | Klaim palsu sudah membayar | Blokir dan tinjauan |
| Beban berlebihan | Membanjiri API | Pembatasan, lalu blokir |
| Upaya pembobolan | Mencoba mengakses data orang lain | Blokir dan laporan |

## 2. Prosedur Penegakan

1. Temuan dicatat beserta bukti.
2. Tingkat pelanggaran ditentukan: ringan, sedang, berat, atau sangat berat.
3. Tindakan diberikan sesuai tingkat, disertai pemberitahuan alasan.
4. Setiap tindakan tercatat dalam catatan audit sistem.

## 3. Tingkat & Tindakan

| Tingkat | Contoh | Tindakan pertama |
|---|---|---|
| Ringan | Lupa memperpanjang pembayaran | Pengingat |
| Sedang | Akun berbagi | Peringatan |
| Berat | Membagikan sinyal | Blokir |
| Sangat berat | Memakai akun broker orang lain | Blokir dan pertimbangan hukum |

## 4. Kontak

[Alamat surel dukungan]
`
