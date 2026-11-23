# Asal kode `ila-strategy`

Folder ini adalah **salinan** aturan trading ILA 2.0. Sumbernya:

| Hal | Nilai |
|---|---|
| Repo asal | `go-trade` (engine live, VPS 1) |
| Path asal | `pkg/ila` dan `pkg/indicators` |
| Commit SHA | `b540dee4eacbd24c09d89972acd5c6b4f17a1fb5` |
| Tanggal commit | 2026-11-23 |
| Judul commit | fix(ila): batasi CISD di jendela C3, gugurkan rangkaian/key level lama, dan cegah re-emit sinyal basi |
| Disalin pada | 2026-11-23 |

## Mengapa disalin, bukan diimpor

`pkg/ila` hanya bergantung pada `pkg/indicators` — dua paket murni tanpa
database, gRPC, maupun jaringan. Namun `go-trade` adalah module terpisah yang
tidak dipublikasikan, sehingga mengimpornya langsung akan memaksa build
platform membutuhkan kredensial jaringan. Salinan di dalam repo membuat build
Docker tetap mandiri.

## Konsekuensi yang harus dijaga

Ada **dua salinan** kode yang sama di dua repo. Bila salah satu diubah dan yang
lain tidak, hasil backtest tidak lagi mencerminkan engine live — dan tidak ada
yang akan menyadarinya sampai angkanya meleset.

Aturan yang wajib diikuti:

1. Setiap perubahan aturan ILA 2.0 **wajib disalin ke dua tempat**.
2. Naikkan `Version` di `version.go` setelah menyalin perubahan.
3. Jalankan `scripts/cek-ila-strategy.sh` sebelum commit untuk memastikan
   keduanya masih identik.
4. Perbarui commit SHA di berkas ini setiap kali menyalin ulang.

## Berkas uji

Berkas `*_test.go` **ikut disalin** dan lulus di module ini (`go test ./...`
bersih). Uji itu menjadi jaring pengaman: bila kelak ada yang mengubah aturan
di sini tanpa sengaja, uji akan menangkapnya.

Satu catatan: uji di sini hanya menguji paket `ila` dan `indicators` secara
mandiri. Di `go-trade` sebagian uji menyentuh paket lain (`internal/service`,
feed pasar), dan bagian itu memang tidak ikut — bukan kehilangan, karena
module ini sengaja tidak tahu soal feed maupun database.
