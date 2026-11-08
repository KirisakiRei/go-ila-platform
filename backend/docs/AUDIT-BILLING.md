# Audit Integrasi Billing & Payment

Daftar lubang yang ditemukan saat menelusuri alur uang dari checkout sampai
tier naik. Tujuannya satu: **mengganti penyedia pembayaran nanti cukup di
lapisan adapter**, tanpa menyentuh service billing maupun halaman.

## A. Gateway & adapter

| # | Temuan | Dampak | Status |
|---|---|---|---|
| A1 | Hanya satu adapter, tanpa pemilih | ganti penyedia berarti mengubah kode service | selesai: `PAYMENT_PROVIDER` + `payment.NewFromConfig` |
| A2 | Tidak ada mode "tanpa penyedia" | bila penyedia tutup, checkout gagal total | selesai: adapter `manual`, tagihan tetap terbit lalu dilunasi lewat catat manual |
| A3 | `RequestRefund` ada di antarmuka tetapi tidak pernah dipanggil | refund tidak tercatat ke gateway | selesai |
| A4 | Penyebab gagal gateway ditelan | tidak bisa ditelusuri dari log | selesai |

## B. Siklus invoice

| # | Temuan | Dampak | Status |
|---|---|---|---|
| B1 | Invoice `pending` yang lewat tenggat tidak pernah menjadi `expired` | daftar menumpuk, laporan menyesatkan | selesai: `ExpireOverdue` di sweeper |
| B2 | Webhook status `expired`/`cancelled` hanya dicatat di tabel payment, invoice tidak berubah | status invoice bertentangan dengan gateway | selesai |
| B3 | Tenggat invoice tidak dicek saat mencatat pembayaran manual | tagihan mati bisa dilunasi | selesai |
| B4 | `closeInvoice` mencatat audit tanpa `Reason` sehingga pencatatan audit gagal | webhook `expired`/`cancelled` balas galat, invoice tertahan `pending` | selesai: alasan diisi, ditutup tes `TestWebhookBatalMenutupInvoicePending` |

## C. Kredit & kupon

| # | Temuan | Dampak | Status |
|---|---|---|---|
| C1 | Invoice disimpan setelah kredit dipotong | saldo pengguna bisa hilang bila penyimpanan gagal | selesai: invoice disimpan lebih dulu |
| C2 | Pemakaian kupon tercatat setelah invoice tersimpan | kupon terbakar bila pencatatan gagal | selesai |

## D. Masa tenggang

| # | Temuan | Dampak | Status |
|---|---|---|---|
| D1 | ~~`DowngradeExpired` hanya menurunkan `subscription_record`~~ | **dugaan awal salah**: repository juga menurunkan `user.tier` ke Free dan mengosongkan tenggat. Tidak ada perubahan. | bukan temuan |

## E. Yang memang disengaja

| Hal | Alasan |
|---|---|
| Refund tidak menurunkan tier | mencabut akses adalah keputusan bisnis terpisah |
| Halaman sukses tidak menaikkan tier | hanya webhook terverifikasi yang menaikkan |
| `manual` menolak seluruh webhook | tidak ada penyedia, jadi tidak ada webhook sah |

## Cara memasang gateway baru

1. Tambah `internal/payment/<nama>.go` yang memenuhi `payment.Gateway`.
2. Daftarkan namanya di `payment.NewFromConfig`.
3. Tambah kredensialnya di `internal/config` dan `deploy/.env.example`.
4. Set `PAYMENT_PROVIDER=<nama>`.

Service billing, invoice, laporan, dan halaman tidak perlu diubah.

## Bukti uji di PostgreSQL

Dijalankan pada basis data sungguhan (Postgres 16), bukan tiruan.

| Skenario | Hasil |
|---|---|
| `PAYMENT_PROVIDER=manual` lalu checkout | `503` + "gunakan catat pembayaran manual"; **tidak ada** invoice `pending` yang tertinggal |
| Webhook saat `manual` | `503`, ditolak sebelum menyentuh data |
| Webhook `pakasir` bertanda tangan salah | `401`; invoice tetap `expired`, tabel `payment` tetap kosong |
| Invoice `pending` lewat tenggat lalu penyapu berjalan | `pending` → `expired` |
| Catat bayar manual pada invoice `expired` | ditolak `409 CONFLICT` |
| Pelunasan manual dengan nominal kurang | ditolak sebelum mencatat apa pun |

Kedua penyedia (`pakasir`, `manual`) start tanpa panic; provider yang tidak
dikenal jatuh ke `manual` dan tercatat di log.
