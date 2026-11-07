# Analisis Keamanan Kredensial

> Hasil audit menyeluruh atas seluruh rahasia yang dimiliki platform:
> apa yang disimpan, bagaimana dilindungi, dan di mana bisa bocor.
> Setiap klaim di sini sudah diverifikasi dengan uji nyata, bukan asumsi.

---

## 1. Ringkasan

| Rahasia | Disimpan di | Perlindungan | Status |
|---|---|---|---|
| Password akun sistem | `user.password_hash` | Argon2id (bcrypt lama dimigrasi otomatis) | ✅ |
| Token broker (Match-Trader) | `credential` (blob) | AES-256-GCM amplop, kunci per akun | ✅ |
| Sesi login user | cookie `ila_session` | JWT + `HttpOnly` + `SameSite=Strict` | ✅ |
| Token anti-CSRF | cookie `ila_csrf` | acak 32 byte, sekali per sesi | ✅ |
| Token bot Telegram | environment saja | tidak pernah ke database | ✅ |
| Master key | environment saja | tidak pernah ke database | ✅ |
| Token deep-link approve | `approval_token` | SHA-256 (bukan plaintext) | ✅ |
| Sidik perangkat/IP saat daftar | `user` | SHA-256 | ✅ |

**Kesimpulan:** seluruh kredensial sudah dienkripsi atau di-hash dengan benar.
Tiga kebocoran ditemukan lewat audit ini — semuanya sudah diperbaiki.

---

## 2. Bagaimana Kredensial Broker Dilindungi

Model **tiga lapis**, sehingga satu kebocoran tidak membuka semua akun:

```
MASTER_KEY (env)  ──membungkus──▶  user_key (per akun, di DB)
                                        │
                                        └──mengenkripsi──▶  ciphertext (DB)
```

| Lapis | Isi | Lokasi |
|---|---|---|
| 1 | `MASTER_KEY` | environment proses, **tidak** di database |
| 2 | kunci acak per akun, terbungkus master key | kolom `user_key_wrapped` |
| 3 | JSON kredensial terenkripsi | kolom `ciphertext` |

**Konsekuensi yang perlu dipahami:**

- Salinan database **saja** tidak cukup untuk membaca token. Penyerang juga
  butuh `MASTER_KEY`.
- `MASTER_KEY` **hilang** = semua token broker **tidak bisa dibuka lagi**.
  Tidak ada jalur pemulihan. Simpan di tempat terpisah dari backup database.
- Blob terikat ke `broker_account_id` sebagai *associated data*, sehingga
  ciphertext tidak bisa dipindahkan antar akun.

**Algoritma:** AES-256-GCM (terautentikasi: perubahan sekecil apa pun membuat
dekripsi gagal), nonce acak 12 byte per operasi, kunci 32 byte.

---

## 3. Siapa yang Menulis dan Membaca

| Proses | Tulis | Baca | Catatan |
|---|---|---|---|
| `backend-api` | ✅ `Seal` | ❌ | hanya menyegel, tidak pernah membuka |
| `runner` | ❌ | ✅ `Open` | plaintext hanya di memori, dibuang saat sesi ditutup |

Pemisahan ini disengaja: backend-api (yang menghadap internet) **tidak punya
kode** untuk membuka token. Bahkan bila backend-api diambil alih, isi token
tidak bisa dibaca langsung dari situ.

Saat user mencabut akun (`DELETE /accounts/:id`), backend memanggil Runner
`/internal/account/revoke` sehingga sesi broker ditutup dan token dibuang dari
memori — bukan hanya dihapus dari database.

---

## 4. Kebocoran yang Ditemukan dan Diperbaiki

### 4.1. Token bot Telegram bocor lewat pesan error

**Tingkat:** tinggi · **Status:** diperbaiki

Token bot berada di **path URL**:
`https://api.telegram.org/bot<TOKEN>/sendMessage`.

Ketika `http.Do` gagal (DNS mati, timeout, koneksi putus), Go menyertakan URL
lengkap di pesan error. Telah dibuktikan dengan uji:

```
Post "https://api.telegram.org/bot123456:RAHASIA-SANGAT/sendMessage":
dial tcp: connectex: target machine actively refused it
```

Error ini berakhir di **dua tempat**: log (`logger.Warn`) dan kolom
`notification_log.last_error` di database.

**Perbaikan:** `sanitizeTransportError` di `internal/telegram/client.go`
mengganti token dengan `<token-disembunyikan>` sebelum error keluar dari paket.

### 4.2. Token deep-link tercatat di log akses

**Tingkat:** sedang · **Status:** diperbaiki

Middleware logger mencatat `c.Request.URL.Path` apa adanya. Untuk
`/api/v1/approve/<token>`, token konfirmasi entry ikut tertulis ke log —
siapa pun yang bisa membaca log dapat memakai tautan itu.

**Perbaikan:** `safePath` menyamarkan segmen token menjadi
`/api/v1/approve/<disembunyikan>` dan tetap mempertahankan sub-path
`/confirm` atau `/decline` agar log tetap berguna.

### 4.3. Tidak ada lapisan kedua untuk rahasia di `last_error`

**Tingkat:** sedang · **Status:** diperbaiki

Setelah memperbaiki 4.1, masih ada kemungkinan error dari sumber lain
(nantinya cTrader, atau pustaka pihak ketiga) membawa rahasia ke kolom yang
ditampilkan di dasbor user.

**Perbaikan:** paket `pkg/redact` menyaring pola rahasia umum sebelum teks
disimpan atau dicatat:

| Pola | Contoh |
|---|---|
| Token bot Telegram | `123456789:AAF...` |
| JWT tiga bagian | `eyJhbGci....eyJ0b2....sig` |
| Header Bearer | `Bearer abcdef...` |
| Parameter sensitif | `token=`, `password=`, `api_key=` |

Dipakai di `RedactSecrets` jalur Runner dan di notifier sebelum
`MarkMessageFailed`.

---

## 5. Yang Sudah Terverifikasi Aman

Diuji pada server berjalan dengan database PostgreSQL sungguhan:

| Uji | Hasil |
|---|---|
| Token plaintext di kolom `ciphertext` | **0 baris** (tidak ada) |
| Ukuran ciphertext | 161 byte (token 61 byte → terenkripsi penuh) |
| `GET /accounts` memuat token | **tidak ada** |
| Field `token`/`ciphertext`/`nonce` di respons API | **tidak ada** |
| Token di seluruh file log | **tidak ada** |
| Audit log | hanya catatan statis, tanpa nilai rahasia |
| Logger middleware | tidak mencatat header maupun body |
| Engine (`go-trade`) | token dari env, tidak menyimpan ke DB, tidak mencetak |

---

## 6. Rotasi Kunci

`MASTER_KEY_VERSION` memungkinkan penggantian kunci tanpa kehilangan data.

**Langkah rotasi:**

1. Buat kunci baru: `openssl rand -base64 32`
2. Pindahkan kunci lama ke `MASTER_KEYS_LEGACY` dengan format
   `<versi-lama>:<kunci-lama>`
3. Set `MASTER_KEY` = kunci baru, `MASTER_KEY_VERSION` = versi baru
4. Nyalakan ulang. Blob lama tetap terbaca lewat kunci lama.
5. Setelah semua akun menulis ulang kredensialnya, hapus
   `MASTER_KEYS_LEGACY`.

Blob baru otomatis memakai versi kunci aktif. Kunci lama hanya untuk **membaca**.

---

## 7. Aturan Operasional yang Tidak Boleh Dilanggar

| ❌ | Alasan |
|---|---|
| Menaruh `MASTER_KEY` di database | Satu kebocoran database membuka semua token |
| Ikut menyertakan `MASTER_KEY` dalam backup DB | Backup biasanya lebih mudah bocor daripada server |
| Menyalin `.env` ke repo atau image Docker | Image bisa ditarik siapa pun |
| Mencatat isi `broker.Credentials` | Satu baris log membocorkan token permanen |
| Mengirim token lewat Telegram | Riwayat chat di luar kendali kita |
| Memakai kunci < 32 byte atau bukan base64 | Ditolak saat start (memang disengaja) |

---

## 8. Batas yang Perlu Diketahui

Hal-hal berikut **tidak** dilindungi oleh enkripsi dan perlu penanganan lain:

1. **Token yang sudah tersebar** di chat atau log pihak lain tidak bisa
   ditarik kembali. Satu-satunya tindakan adalah rotasi token di broker
   dan hapus tautan Telegram.
2. **Proses yang diambil alih.** Bila penyerang sudah menjalankan kode di
   dalam container Runner, ia bisa membaca token dari memori saat dipakai.
   Enkripsi melindungi *data diam*, bukan *data saat dipakai*.
3. **`MASTER_KEY` di environment.** Siapa pun yang bisa membaca environment
   container bisa membuka semua token. Untuk skala 10–20 user ini wajar;
   bila nanti membesar, pertimbangkan KMS atau Vault.
4. **Password user yang lemah.** Argon2id memperlambat tebakan, tetapi tidak
   memperbaiki password yang mudah ditebak. Panjang minimum 10 karakter
   sudah ditegakkan.

---

## 9. Berkas Terkait

| Berkas | Peran |
|---|---|
| `pkg/cryptox/envelope.go` | enkripsi amplop AES-256-GCM |
| `pkg/redact/redact.go` | penyaring rahasia dari teks |
| `internal/service/account_service.go` | menulis blob terenkripsi |
| `internal/runner/brokerpool.go` | membuka blob, mengelola sesi |
| `internal/telegram/client.go` | `sanitizeTransportError` |
| `internal/middleware/logger.go` | `safePath` |
| `internal/runner/notify/notifier.go` | penyaringan sebelum `MarkMessageFailed` |
